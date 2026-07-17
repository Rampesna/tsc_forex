package service

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/tsc-forex/tsc_forex/internal/models"
)

func (s *SubscriptionService) lockCapacity(ctx context.Context, tx *gorm.DB, userID uuid.UUID, feature string) error {
	row := models.BillingCapacityLock{UserID: userID, Feature: feature, UpdatedAt: s.now()}
	if err := tx.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "feature"}},
		DoNothing: true,
	}).Create(&row).Error; err != nil {
		return err
	}
	// The no-op UPDATE obtains a row write lock on PostgreSQL. This works for a
	// newly inserted row and an existing row, closing the count+insert race
	// across pods before the later SELECT FOR UPDATE.
	if err := tx.WithContext(ctx).Model(&models.BillingCapacityLock{}).
		Where("user_id = ? AND feature = ?", userID, feature).
		UpdateColumn("updated_at", gorm.Expr("updated_at")).Error; err != nil {
		return err
	}
	return tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ? AND feature = ?", userID, feature).First(&row).Error
}

func (s *SubscriptionService) capacityLimitTx(ctx context.Context, tx *gorm.DB, userID uuid.UUID, feature string) (int, error) {
	ent, active := s.lockedEntitlement(ctx, tx, userID, s.now())
	plan := freePlan
	if active {
		plan = planForCode(ent.PlanCode)
	}
	limit := limitForFeature(plan.Limits, feature)
	if limit <= 0 {
		return 0, &QuotaError{Feature: feature, Code: "subscription_required"}
	}
	return limit, nil
}

// AddWatchlistWithinCapacity makes idempotent duplicate adds free while
// serializing every new row against the exact capacity count.
func (s *SubscriptionService) AddWatchlistWithinCapacity(ctx context.Context, userID uuid.UUID, code string) (models.WatchlistItem, bool, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	var item models.WatchlistItem
	created := false
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.lockCapacity(ctx, tx, userID, FeatureWatchlist); err != nil {
			return err
		}
		findErr := tx.Where("user_id = ? AND code = ?", userID, code).First(&item).Error
		if findErr == nil {
			return nil
		}
		if !errors.Is(findErr, gorm.ErrRecordNotFound) {
			return findErr
		}
		limit, err := s.capacityLimitTx(ctx, tx, userID, FeatureWatchlist)
		if err != nil {
			return err
		}
		used, err := s.resourceCount(ctx, tx, userID, FeatureWatchlist)
		if err != nil {
			return err
		}
		if used >= limit {
			return &QuotaError{Feature: FeatureWatchlist, Limit: limit, Used: used, Code: "quota_exhausted"}
		}
		item = models.WatchlistItem{UserID: userID, Code: code, CreatedAt: s.now()}
		if err := tx.Create(&item).Error; err != nil {
			return err
		}
		created = true
		return nil
	})
	return item, created, err
}

func (s *SubscriptionService) CreateAlertWithinCapacity(ctx context.Context, alert *models.PriceAlert) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.lockCapacity(ctx, tx, alert.UserID, FeatureActiveAlerts); err != nil {
			return err
		}
		limit, err := s.capacityLimitTx(ctx, tx, alert.UserID, FeatureActiveAlerts)
		if err != nil {
			return err
		}
		used, err := s.resourceCount(ctx, tx, alert.UserID, FeatureActiveAlerts)
		if err != nil {
			return err
		}
		if used >= limit {
			return &QuotaError{Feature: FeatureActiveAlerts, Limit: limit, Used: used, Code: "quota_exhausted"}
		}
		return tx.Create(alert).Error
	})
}

func (s *SubscriptionService) CreatePortfolioWithinCapacity(ctx context.Context, portfolio *models.Portfolio) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.lockCapacity(ctx, tx, portfolio.UserID, FeaturePortfolios); err != nil {
			return err
		}
		limit, err := s.capacityLimitTx(ctx, tx, portfolio.UserID, FeaturePortfolios)
		if err != nil {
			return err
		}
		used, err := s.resourceCount(ctx, tx, portfolio.UserID, FeaturePortfolios)
		if err != nil {
			return err
		}
		if used >= limit {
			return &QuotaError{Feature: FeaturePortfolios, Limit: limit, Used: used, Code: "quota_exhausted"}
		}
		return tx.Create(portfolio).Error
	})
}
