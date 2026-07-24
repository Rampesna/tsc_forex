package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/tsc-forex/tsc_forex/internal/metrics"
	"github.com/tsc-forex/tsc_forex/internal/models"
)

// ErrUserNotFound indicates no user matched the query.
var ErrUserNotFound = errors.New("user not found")
var ErrSubscriptionRecoveryRequired = errors.New("subscription recovery token required before account deletion")

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
	DeleteUser(ctx context.Context, id uuid.UUID, subscriptionRecoveryToken ...string) error
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
func (r *userRepository) DeleteUser(ctx context.Context, id uuid.UUID, subscriptionRecoveryToken ...string) error {
	start := time.Now()
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		deletedAt := time.Now().UTC()
		var identities []models.AuthIdentity
		if err := tx.Where("user_id = ?", id).Find(&identities).Error; err != nil {
			return err
		}
		var recoveryProvider, recoverySubjectHash string
		for _, provider := range []string{"apple", "google"} {
			for _, identity := range identities {
				if identity.Provider == provider && strings.TrimSpace(identity.Subject) != "" {
					recoveryProvider = provider
					recoverySubjectHash = subscriptionRestoreHash(provider+"-subject", identity.Subject)
					break
				}
			}
			if recoveryProvider != "" {
				break
			}
		}
		recoveryToken := ""
		if len(subscriptionRecoveryToken) > 0 {
			recoveryToken = strings.TrimSpace(subscriptionRecoveryToken[0])
		}
		recoveryTokenHash := ""
		if len(recoveryToken) >= 32 && len(recoveryToken) <= 256 {
			recoveryTokenHash = subscriptionRestoreHash("recovery-token", recoveryToken)
		}
		var appleTransactions []models.AppleTransaction
		if err := tx.Where("user_id = ?", id).Find(&appleTransactions).Error; err != nil {
			return err
		}
		if len(appleTransactions) > 0 && recoverySubjectHash == "" && recoveryTokenHash == "" {
			return ErrSubscriptionRecoveryRequired
		}
		if recoverySubjectHash != "" || recoveryTokenHash != "" {
			for _, transaction := range appleTransactions {
				if strings.TrimSpace(transaction.OriginalTransactionID) == "" || strings.TrimSpace(transaction.AppAccountToken) == "" {
					continue
				}
				tombstone := models.SubscriptionRestoreTombstone{
					OriginalTransactionID:       transaction.OriginalTransactionID,
					PreviousAppAccountTokenHash: subscriptionRestoreHash("app-account-token", transaction.AppAccountToken),
					IdentityProvider:            recoveryProvider,
					ProviderSubjectHash:         recoverySubjectHash,
					RecoveryTokenHash:           recoveryTokenHash,
					DeletedAt:                   deletedAt,
				}
				if err := tx.Clauses(clause.OnConflict{
					Columns: []clause.Column{{Name: "original_transaction_id"}},
					DoUpdates: clause.Assignments(map[string]any{
						"previous_app_account_token_hash": tombstone.PreviousAppAccountTokenHash,
						"identity_provider":               tombstone.IdentityProvider, "provider_subject_hash": tombstone.ProviderSubjectHash,
						"recovery_token_hash": tombstone.RecoveryTokenHash, "deleted_at": deletedAt,
						"rebound_user_id": nil, "rebound_at": nil,
					}),
				}).Create(&tombstone).Error; err != nil {
					return err
				}
			}
		}
		// Keep only purpose-limited, de-linked anti-fraud/transaction records.
		// The device claim prevents repeated free-trial abuse after account
		// recreation; the App Store transaction shell supports refunds/restores.
		if err := tx.Model(&models.FreeResearchClaim{}).Where("user_id = ?", id).
			Updates(map[string]any{"user_id": nil, "account_deleted_at": deletedAt}).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.AppleTransaction{}).Where("user_id = ?", id).
			Updates(map[string]any{"user_id": nil, "app_account_token": ""}).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.GooglePlayTransaction{}).Where("user_id = ?", id).
			Updates(map[string]any{"user_id": nil, "obfuscated_account_id": ""}).Error; err != nil {
			return err
		}
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
			&models.SubscriptionEntitlement{},
			&models.FeatureUsage{},
			&models.MarketResearchArchive{},
			&models.MarketResearchJob{},
			&models.FeatureArtifact{},
			&models.AppAttestChallenge{},
			&models.DeviceCheckRedemption{},
			&models.BillingCapacityLock{},
		}
		for _, m := range owned {
			if err := tx.Where("user_id = ?", id).Delete(m).Error; err != nil {
				return err
			}
		}
		if err := tx.Model(&models.AppAttestKey{}).Where("registered_user_id = ?", id).
			Update("registered_user_id", nil).Error; err != nil {
			return err
		}
		return tx.Where("id = ?", id).Delete(&models.User{}).Error
	})
	r.observe(start, err)
	return err
}

func subscriptionRestoreHash(purpose, value string) string {
	digest := sha256.Sum256([]byte("tsc-subscription-restore-v1:" + purpose + ":" + strings.TrimSpace(value)))
	return hex.EncodeToString(digest[:])
}
