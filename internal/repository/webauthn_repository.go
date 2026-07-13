package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/tsc-forex/tsc_forex/internal/metrics"
	"github.com/tsc-forex/tsc_forex/internal/models"
)

// WebAuthnCredentialRepository stores registered passkeys.
type WebAuthnCredentialRepository interface {
	Create(ctx context.Context, c *models.WebAuthnCredential) error
	ListByUser(ctx context.Context, userID uuid.UUID) ([]models.WebAuthnCredential, error)
	ByCredentialID(ctx context.Context, credentialID []byte) (models.WebAuthnCredential, error)
	UpdateSignCount(ctx context.Context, id uuid.UUID, signCount uint32) error
}

type webAuthnCredentialRepository struct{ gormBase }

// NewWebAuthnCredentialRepository builds the passkey credential repository.
func NewWebAuthnCredentialRepository(db *gorm.DB, m *metrics.Metrics) WebAuthnCredentialRepository {
	return &webAuthnCredentialRepository{gormBase{db, m}}
}

func (r *webAuthnCredentialRepository) Create(ctx context.Context, c *models.WebAuthnCredential) error {
	start := time.Now()
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	c.CreatedAt = time.Now().UTC()
	err := r.db.WithContext(ctx).Create(c).Error
	r.observe(start, err)
	return err
}

func (r *webAuthnCredentialRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]models.WebAuthnCredential, error) {
	start := time.Now()
	var rows []models.WebAuthnCredential
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&rows).Error
	r.observe(start, err)
	return rows, err
}

func (r *webAuthnCredentialRepository) ByCredentialID(ctx context.Context, credentialID []byte) (models.WebAuthnCredential, error) {
	start := time.Now()
	var row models.WebAuthnCredential
	err := r.db.WithContext(ctx).Where("credential_id = ?", credentialID).First(&row).Error
	r.observe(start, err)
	return row, err
}

func (r *webAuthnCredentialRepository) UpdateSignCount(ctx context.Context, id uuid.UUID, signCount uint32) error {
	start := time.Now()
	err := r.db.WithContext(ctx).Model(&models.WebAuthnCredential{}).
		Where("id = ?", id).Update("sign_count", signCount).Error
	r.observe(start, err)
	return err
}
