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

// ErrNotFound is a generic not-found for user-owned resources.
var ErrNotFound = errors.New("not found")

// WatchlistRepository manages a user's followed instruments.
type WatchlistRepository interface {
	Add(ctx context.Context, userID uuid.UUID, code string) (models.WatchlistItem, error)
	Remove(ctx context.Context, userID uuid.UUID, code string) error
	List(ctx context.Context, userID uuid.UUID) ([]models.WatchlistItem, error)
}

// AlertRepository manages user price alerts.
type AlertRepository interface {
	Create(ctx context.Context, a *models.PriceAlert) error
	Update(ctx context.Context, userID, id uuid.UUID, direction string, threshold float64) (models.PriceAlert, error)
	Delete(ctx context.Context, userID, id uuid.UUID) error
	ListByUser(ctx context.Context, userID uuid.UUID) ([]models.PriceAlert, error)
	ActiveByCode(ctx context.Context, code string) ([]models.PriceAlert, error)
	// ClaimTriggered atomically deactivates an alert and persists its notification
	// only when it still matches the snapshot evaluated by the alert engine. The
	// boolean is true only for the single worker/process that won the claim.
	ClaimTriggered(ctx context.Context, alert models.PriceAlert, notification *models.Notification, at time.Time) (bool, error)
}

// NotificationRepository stores user notifications.
type NotificationRepository interface {
	Create(ctx context.Context, n *models.Notification) error
	ListByUser(ctx context.Context, userID uuid.UUID, limit int) ([]models.Notification, error)
	MarkRead(ctx context.Context, userID, id uuid.UUID) error
	Delete(ctx context.Context, userID, id uuid.UUID) error
	DeleteAll(ctx context.Context, userID uuid.UUID) (int64, error)
}

// RefreshTokenRepository stores rotating, hashed refresh tokens.
type RefreshTokenRepository interface {
	Store(ctx context.Context, t *models.RefreshToken) error
	ByHash(ctx context.Context, hash string) (models.RefreshToken, error)
	Revoke(ctx context.Context, id uuid.UUID) error
	RevokeAllForUser(ctx context.Context, userID uuid.UUID) error
}

// DeviceTokenRepository stores push tokens for user devices.
type DeviceTokenRepository interface {
	Upsert(ctx context.Context, t *models.DeviceToken) error
	ListByUser(ctx context.Context, userID uuid.UUID) ([]models.DeviceToken, error)
}

// --- implementations ---

type gormBase struct {
	db      *gorm.DB
	metrics *metrics.Metrics
}

func (b gormBase) observe(start time.Time, err error) {
	if b.metrics != nil {
		b.metrics.DBLatency.Observe(time.Since(start).Seconds())
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			b.metrics.DBErrors.Inc()
		}
	}
}

// Watchlist ------------------------------------------------------------------

type watchlistRepository struct{ gormBase }

func NewWatchlistRepository(db *gorm.DB, m *metrics.Metrics) WatchlistRepository {
	return &watchlistRepository{gormBase{db, m}}
}

func (r *watchlistRepository) Add(ctx context.Context, userID uuid.UUID, code string) (models.WatchlistItem, error) {
	start := time.Now()
	var item models.WatchlistItem
	// Idempotent: Attrs are only applied on insert, so a repeat add is a no-op.
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND code = ?", userID, code).
		Attrs(models.WatchlistItem{ID: uuid.New(), UserID: userID, Code: code, CreatedAt: time.Now().UTC()}).
		FirstOrCreate(&item).Error
	r.observe(start, err)
	return item, err
}

func (r *watchlistRepository) Remove(ctx context.Context, userID uuid.UUID, code string) error {
	start := time.Now()
	err := r.db.WithContext(ctx).Where("user_id = ? AND code = ?", userID, code).
		Delete(&models.WatchlistItem{}).Error
	r.observe(start, err)
	return err
}

func (r *watchlistRepository) List(ctx context.Context, userID uuid.UUID) ([]models.WatchlistItem, error) {
	start := time.Now()
	var rows []models.WatchlistItem
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at ASC").Find(&rows).Error
	r.observe(start, err)
	return rows, err
}

// Alerts ---------------------------------------------------------------------

type alertRepository struct{ gormBase }

func NewAlertRepository(db *gorm.DB, m *metrics.Metrics) AlertRepository {
	return &alertRepository{gormBase{db, m}}
}

func (r *alertRepository) Create(ctx context.Context, a *models.PriceAlert) error {
	start := time.Now()
	err := r.db.WithContext(ctx).Create(a).Error
	r.observe(start, err)
	return err
}

func (r *alertRepository) Update(ctx context.Context, userID, id uuid.UUID, direction string, threshold float64) (models.PriceAlert, error) {
	start := time.Now()
	var updated models.PriceAlert
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&models.PriceAlert{}).
			Where("id = ? AND user_id = ? AND active = ?", id, userID, true).
			Updates(map[string]any{
				"direction":  direction,
				"threshold":  threshold,
				"updated_at": time.Now().UTC(),
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrNotFound
		}
		return tx.Where("id = ? AND user_id = ?", id, userID).First(&updated).Error
	})
	r.observe(start, err)
	return updated, err
}

func (r *alertRepository) Delete(ctx context.Context, userID, id uuid.UUID) error {
	start := time.Now()
	res := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).Delete(&models.PriceAlert{})
	r.observe(start, res.Error)
	if res.Error == nil && res.RowsAffected == 0 {
		return ErrNotFound
	}
	return res.Error
}

func (r *alertRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]models.PriceAlert, error) {
	start := time.Now()
	var rows []models.PriceAlert
	err := r.db.WithContext(ctx).Where("user_id = ? AND active = ?", userID, true).
		Order("created_at DESC").Find(&rows).Error
	r.observe(start, err)
	return rows, err
}

func (r *alertRepository) ActiveByCode(ctx context.Context, code string) ([]models.PriceAlert, error) {
	start := time.Now()
	var rows []models.PriceAlert
	err := r.db.WithContext(ctx).Where("code = ? AND active = ?", code, true).Find(&rows).Error
	r.observe(start, err)
	return rows, err
}

func (r *alertRepository) ClaimTriggered(ctx context.Context, alert models.PriceAlert, notification *models.Notification, at time.Time) (bool, error) {
	start := time.Now()
	claimed := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&models.PriceAlert{}).
			Where(
				"id = ? AND user_id = ? AND code = ? AND direction = ? AND threshold = ? AND active = ?",
				alert.ID, alert.UserID, alert.Code, alert.Direction, alert.Threshold, true,
			).
			Updates(map[string]any{
				"active":        false,
				"last_fired_at": at,
				"updated_at":    at,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil
		}
		if notification == nil {
			return errors.New("trigger notification is required")
		}
		if err := tx.Create(notification).Error; err != nil {
			return err
		}
		claimed = true
		return nil
	})
	r.observe(start, err)
	if err != nil {
		claimed = false
	}
	return claimed, err
}

// Notifications --------------------------------------------------------------

type notificationRepository struct{ gormBase }

func NewNotificationRepository(db *gorm.DB, m *metrics.Metrics) NotificationRepository {
	return &notificationRepository{gormBase{db, m}}
}

func (r *notificationRepository) Create(ctx context.Context, n *models.Notification) error {
	start := time.Now()
	err := r.db.WithContext(ctx).Create(n).Error
	r.observe(start, err)
	return err
}

func (r *notificationRepository) ListByUser(ctx context.Context, userID uuid.UUID, limit int) ([]models.Notification, error) {
	start := time.Now()
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	var rows []models.Notification
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).
		Order("created_at DESC").Limit(limit).Find(&rows).Error
	r.observe(start, err)
	return rows, err
}

func (r *notificationRepository) MarkRead(ctx context.Context, userID, id uuid.UUID) error {
	start := time.Now()
	now := time.Now().UTC()
	res := r.db.WithContext(ctx).Model(&models.Notification{}).
		Where("id = ? AND user_id = ?", id, userID).Update("read_at", now)
	r.observe(start, res.Error)
	if res.Error == nil && res.RowsAffected == 0 {
		return ErrNotFound
	}
	return res.Error
}

func (r *notificationRepository) Delete(ctx context.Context, userID, id uuid.UUID) error {
	start := time.Now()
	res := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		Delete(&models.Notification{})
	r.observe(start, res.Error)
	if res.Error == nil && res.RowsAffected == 0 {
		return ErrNotFound
	}
	return res.Error
}

func (r *notificationRepository) DeleteAll(ctx context.Context, userID uuid.UUID) (int64, error) {
	start := time.Now()
	res := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Delete(&models.Notification{})
	r.observe(start, res.Error)
	return res.RowsAffected, res.Error
}

// Refresh tokens -------------------------------------------------------------

type refreshTokenRepository struct{ gormBase }

func NewRefreshTokenRepository(db *gorm.DB, m *metrics.Metrics) RefreshTokenRepository {
	return &refreshTokenRepository{gormBase{db, m}}
}

func (r *refreshTokenRepository) Store(ctx context.Context, t *models.RefreshToken) error {
	start := time.Now()
	err := r.db.WithContext(ctx).Create(t).Error
	r.observe(start, err)
	return err
}

func (r *refreshTokenRepository) ByHash(ctx context.Context, hash string) (models.RefreshToken, error) {
	start := time.Now()
	var t models.RefreshToken
	err := r.db.WithContext(ctx).Where("token_hash = ?", hash).First(&t).Error
	r.observe(start, err)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return models.RefreshToken{}, ErrNotFound
	}
	return t, err
}

func (r *refreshTokenRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	start := time.Now()
	now := time.Now().UTC()
	err := r.db.WithContext(ctx).Model(&models.RefreshToken{}).Where("id = ?", id).
		Update("revoked_at", now).Error
	r.observe(start, err)
	return err
}

func (r *refreshTokenRepository) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	start := time.Now()
	now := time.Now().UTC()
	err := r.db.WithContext(ctx).Model(&models.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).Update("revoked_at", now).Error
	r.observe(start, err)
	return err
}

// Device tokens --------------------------------------------------------------

type deviceTokenRepository struct{ gormBase }

func NewDeviceTokenRepository(db *gorm.DB, m *metrics.Metrics) DeviceTokenRepository {
	return &deviceTokenRepository{gormBase{db, m}}
}

func (r *deviceTokenRepository) Upsert(ctx context.Context, t *models.DeviceToken) error {
	start := time.Now()
	var existing models.DeviceToken
	err := r.db.WithContext(ctx).Where("token = ?", t.Token).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if t.ID == uuid.Nil {
			t.ID = uuid.New()
		}
		t.CreatedAt = time.Now().UTC()
		err = r.db.WithContext(ctx).Create(t).Error
		r.observe(start, err)
		return err
	}
	r.observe(start, err)
	return err
}

func (r *deviceTokenRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]models.DeviceToken, error) {
	start := time.Now()
	var rows []models.DeviceToken
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&rows).Error
	r.observe(start, err)
	return rows, err
}
