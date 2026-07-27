package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/tsc-forex/tsc_forex/internal/metrics"
	"github.com/tsc-forex/tsc_forex/internal/models"
)

var ErrManagerNotFound = errors.New("manager not found")

// ManagerRepository is deliberately separate from UserRepository. Operations
// identities must never be resolved through the consumer account table.
type ManagerRepository interface {
	Create(ctx context.Context, manager *models.Manager) error
	ByEmail(ctx context.Context, email string) (models.Manager, error)
	ByID(ctx context.Context, id uuid.UUID) (models.Manager, error)
	Update(ctx context.Context, manager *models.Manager) error
}

// ManagerSessionRepository persists opaque, rotating manager refresh sessions
// in their own table.
type ManagerSessionRepository interface {
	Store(ctx context.Context, session *models.ManagerSession) error
	ByHash(ctx context.Context, hash string) (models.ManagerSession, error)
	Revoke(ctx context.Context, id uuid.UUID) error
	RevokeAllForManager(ctx context.Context, managerID uuid.UUID) error
}

type managerRepository struct{ gormBase }

func NewManagerRepository(db *gorm.DB, m *metrics.Metrics) ManagerRepository {
	return &managerRepository{gormBase{db: db, metrics: m}}
}

func (r *managerRepository) Create(ctx context.Context, manager *models.Manager) error {
	start := time.Now()
	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "email"}},
		DoNothing: true,
	}).Create(manager).Error
	r.observe(start, err)
	return err
}

func (r *managerRepository) ByEmail(ctx context.Context, email string) (models.Manager, error) {
	start := time.Now()
	var manager models.Manager
	err := r.db.WithContext(ctx).Where("email = ?", email).Take(&manager).Error
	r.observe(start, err)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return models.Manager{}, ErrManagerNotFound
	}
	return manager, err
}

func (r *managerRepository) ByID(ctx context.Context, id uuid.UUID) (models.Manager, error) {
	start := time.Now()
	var manager models.Manager
	err := r.db.WithContext(ctx).Where("id = ?", id).Take(&manager).Error
	r.observe(start, err)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return models.Manager{}, ErrManagerNotFound
	}
	return manager, err
}

func (r *managerRepository) Update(ctx context.Context, manager *models.Manager) error {
	start := time.Now()
	err := r.db.WithContext(ctx).Save(manager).Error
	r.observe(start, err)
	return err
}

type managerSessionRepository struct{ gormBase }

func NewManagerSessionRepository(db *gorm.DB, m *metrics.Metrics) ManagerSessionRepository {
	return &managerSessionRepository{gormBase{db: db, metrics: m}}
}

func (r *managerSessionRepository) Store(ctx context.Context, session *models.ManagerSession) error {
	start := time.Now()
	err := r.db.WithContext(ctx).Create(session).Error
	r.observe(start, err)
	return err
}

func (r *managerSessionRepository) ByHash(ctx context.Context, hash string) (models.ManagerSession, error) {
	start := time.Now()
	var session models.ManagerSession
	err := r.db.WithContext(ctx).Where("token_hash = ?", hash).Take(&session).Error
	r.observe(start, err)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return models.ManagerSession{}, ErrNotFound
	}
	return session, err
}

func (r *managerSessionRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	start := time.Now()
	now := time.Now().UTC()
	err := r.db.WithContext(ctx).Model(&models.ManagerSession{}).
		Where("id = ? AND revoked_at IS NULL", id).
		Update("revoked_at", now).Error
	r.observe(start, err)
	return err
}

func (r *managerSessionRepository) RevokeAllForManager(ctx context.Context, managerID uuid.UUID) error {
	start := time.Now()
	now := time.Now().UTC()
	err := r.db.WithContext(ctx).Model(&models.ManagerSession{}).
		Where("manager_id = ? AND revoked_at IS NULL", managerID).
		Update("revoked_at", now).Error
	r.observe(start, err)
	return err
}
