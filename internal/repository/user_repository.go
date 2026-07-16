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
	// DeleteUser permanently removes the user and ALL of their owned data
	// (watchlist, alerts, notifications, devices, identities, sessions,
	// passkeys, gamification). Required for App Store / Play Store account
	// deletion compliance.
	DeleteUser(ctx context.Context, id uuid.UUID) error
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

// DeleteUser removes the user and every row they own, in a single transaction
// so a partial failure cannot leave orphaned personal data behind.
func (r *userRepository) DeleteUser(ctx context.Context, id uuid.UUID) error {
	start := time.Now()
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		portfolioIDs := tx.Model(&models.Portfolio{}).
			Select("id").
			Where("user_id = ?", id)
		if err := tx.Where("portfolio_id IN (?)", portfolioIDs).
			Delete(&models.PortfolioPosition{}).Error; err != nil {
			return err
		}
		ownedGroupIDs := tx.Model(&models.StudyGroup{}).
			Select("id").
			Where("owner_id = ?", id)
		if err := tx.Where("group_id IN (?)", ownedGroupIDs).
			Delete(&models.StudyGroupMember{}).Error; err != nil {
			return err
		}
		if err := tx.Where("owner_id = ?", id).Delete(&models.StudyGroup{}).Error; err != nil {
			return err
		}
		owned := []any{
			&models.WatchlistItem{},
			&models.PriceAlert{},
			&models.Notification{},
			&models.DeviceToken{},
			&models.RefreshToken{},
			&models.AuthIdentity{},
			&models.WebAuthnCredential{},
			&models.UserBadge{},
			&models.GamificationEvent{},
			&models.XPLedgerEntry{},
			&models.UserGamification{},
			&models.DailyPlan{},
			&models.LearningProgress{},
			&models.LearningAttempt{},
			&models.DisciplineReview{},
			&models.SeasonScore{},
			&models.PaperTrade{},
			&models.GamificationPrivacy{},
			&models.StudyGroupMember{},
			&models.PortfolioTransaction{},
			&models.Portfolio{},
		}
		for _, m := range owned {
			if err := tx.Where("user_id = ?", id).Delete(m).Error; err != nil {
				return err
			}
		}
		return tx.Where("id = ?", id).Delete(&models.User{}).Error
	})
	r.observe(start, err)
	return err
}
