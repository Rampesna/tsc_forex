package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/tsc-forex/tsc_forex/internal/metrics"
	"github.com/tsc-forex/tsc_forex/internal/models"
)

// ErrUserNotFound indicates no user matched the query.
var ErrUserNotFound = errors.New("user not found")

// UserRepository manages authentication users.
type UserRepository interface {
	Create(ctx context.Context, u *models.User) error
	ByEmail(ctx context.Context, email string) (models.User, error)
	ByID(ctx context.Context, id uuid.UUID) (models.User, error)
	Update(ctx context.Context, u *models.User) error
}

type userRepository struct {
	db      *gorm.DB
	metrics *metrics.Metrics
}

// NewUserRepository builds a GORM-backed user repository.
func NewUserRepository(db *gorm.DB, m *metrics.Metrics) UserRepository {
	return &userRepository{db: db, metrics: m}
}

func (r *userRepository) observe(start time.Time, err error) {
	if r.metrics != nil {
		r.metrics.DBLatency.Observe(time.Since(start).Seconds())
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			r.metrics.DBErrors.Inc()
		}
	}
}

func (r *userRepository) Create(ctx context.Context, u *models.User) error {
	start := time.Now()
	err := r.db.WithContext(ctx).Create(u).Error
	r.observe(start, err)
	return err
}

func (r *userRepository) ByEmail(ctx context.Context, email string) (models.User, error) {
	start := time.Now()
	var u models.User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&u).Error
	r.observe(start, err)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return models.User{}, ErrUserNotFound
	}
	return u, err
}

// Update persists mutable user fields (profile, avatar, provider).
func (r *userRepository) Update(ctx context.Context, u *models.User) error {
	start := time.Now()
	err := r.db.WithContext(ctx).Save(u).Error
	r.observe(start, err)
	return err
}

// ByID fetches a user by primary key. Used by the refresh-token rotation flow.
func (r *userRepository) ByID(ctx context.Context, id uuid.UUID) (models.User, error) {
	start := time.Now()
	var u models.User
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&u).Error
	r.observe(start, err)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return models.User{}, ErrUserNotFound
	}
	return u, err
}
