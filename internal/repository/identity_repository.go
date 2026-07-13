package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/tsc-forex/tsc_forex/internal/metrics"
	"github.com/tsc-forex/tsc_forex/internal/models"
)

// AuthIdentityRepository binds external provider identities (Google/Apple) to
// local users, enabling multiple providers per account.
type AuthIdentityRepository interface {
	Create(ctx context.Context, a *models.AuthIdentity) error
	ByProviderSubject(ctx context.Context, provider, subject string) (models.AuthIdentity, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]models.AuthIdentity, error)
}

type authIdentityRepository struct{ gormBase }

// NewAuthIdentityRepository builds the provider-identity repository.
func NewAuthIdentityRepository(db *gorm.DB, m *metrics.Metrics) AuthIdentityRepository {
	return &authIdentityRepository{gormBase{db, m}}
}

func (r *authIdentityRepository) Create(ctx context.Context, a *models.AuthIdentity) error {
	start := time.Now()
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	if a.CreatedAt.IsZero() {
		a.CreatedAt = time.Now().UTC()
	}
	err := r.db.WithContext(ctx).Create(a).Error
	r.observe(start, err)
	return err
}

func (r *authIdentityRepository) ByProviderSubject(ctx context.Context, provider, subject string) (models.AuthIdentity, error) {
	start := time.Now()
	var row models.AuthIdentity
	err := r.db.WithContext(ctx).
		Where("provider = ? AND subject = ?", provider, subject).
		First(&row).Error
	r.observe(start, err)
	return row, err
}

func (r *authIdentityRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]models.AuthIdentity, error) {
	start := time.Now()
	var rows []models.AuthIdentity
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&rows).Error
	r.observe(start, err)
	return rows, err
}
