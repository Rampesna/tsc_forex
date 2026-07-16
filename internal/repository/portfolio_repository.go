package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/tsc-forex/tsc_forex/internal/metrics"
	"github.com/tsc-forex/tsc_forex/internal/models"
)

var (
	ErrPortfolioNotFound   = errors.New("portfolio not found")
	ErrTransactionNotFound = errors.New("portfolio transaction not found")
)

// PortfolioProjector deterministically rebuilds the materialized positions and
// cash balance from the complete ordered ledger.
type PortfolioProjector func(
	portfolioID uuid.UUID,
	transactions []models.PortfolioTransaction,
) ([]models.PortfolioPosition, decimal.Decimal, error)

// PortfolioRepository persists user-owned portfolios and guarantees that a
// ledger mutation, projection rebuild and version increment commit atomically.
type PortfolioRepository interface {
	Create(ctx context.Context, portfolio *models.Portfolio) error
	List(ctx context.Context, userID uuid.UUID) ([]models.Portfolio, error)
	Get(ctx context.Context, userID, portfolioID uuid.UUID) (models.Portfolio, error)
	Update(ctx context.Context, portfolio *models.Portfolio) error
	Delete(ctx context.Context, userID, portfolioID uuid.UUID) error

	Positions(ctx context.Context, userID, portfolioID uuid.UUID) ([]models.PortfolioPosition, error)
	Transactions(ctx context.Context, userID, portfolioID uuid.UUID, limit int, before time.Time) ([]models.PortfolioTransaction, error)
	Transaction(ctx context.Context, userID, portfolioID, transactionID uuid.UUID) (models.PortfolioTransaction, error)
	CreateTransaction(
		ctx context.Context,
		userID, portfolioID uuid.UUID,
		transaction *models.PortfolioTransaction,
		project PortfolioProjector,
	) (models.PortfolioTransaction, bool, error)
	UpdateTransaction(
		ctx context.Context,
		userID, portfolioID uuid.UUID,
		transaction *models.PortfolioTransaction,
		project PortfolioProjector,
	) (models.PortfolioTransaction, error)
	DeleteTransaction(
		ctx context.Context,
		userID, portfolioID, transactionID uuid.UUID,
		project PortfolioProjector,
	) error
}

type portfolioRepository struct{ gormBase }

func NewPortfolioRepository(db *gorm.DB, m *metrics.Metrics) PortfolioRepository {
	return &portfolioRepository{gormBase{db: db, metrics: m}}
}

func (r *portfolioRepository) Create(ctx context.Context, portfolio *models.Portfolio) error {
	start := time.Now()
	err := r.db.WithContext(ctx).Create(portfolio).Error
	r.observe(start, err)
	return err
}

func (r *portfolioRepository) List(ctx context.Context, userID uuid.UUID) ([]models.Portfolio, error) {
	start := time.Now()
	var rows []models.Portfolio
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("updated_at DESC, id ASC").
		Find(&rows).Error
	r.observe(start, err)
	return rows, err
}

func (r *portfolioRepository) Get(ctx context.Context, userID, portfolioID uuid.UUID) (models.Portfolio, error) {
	start := time.Now()
	var portfolio models.Portfolio
	err := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", portfolioID, userID).
		First(&portfolio).Error
	r.observe(start, err)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return models.Portfolio{}, ErrPortfolioNotFound
	}
	return portfolio, err
}

func (r *portfolioRepository) Update(ctx context.Context, portfolio *models.Portfolio) error {
	start := time.Now()
	result := r.db.WithContext(ctx).Model(&models.Portfolio{}).
		Where("id = ? AND user_id = ?", portfolio.ID, portfolio.UserID).
		Updates(map[string]any{
			"name":          portfolio.Name,
			"base_currency": portfolio.BaseCurrency,
			"version":       gorm.Expr("version + 1"),
			"updated_at":    time.Now().UTC(),
		})
	r.observe(start, result.Error)
	if result.Error == nil && result.RowsAffected == 0 {
		return ErrPortfolioNotFound
	}
	return result.Error
}

func (r *portfolioRepository) Delete(ctx context.Context, userID, portfolioID uuid.UUID) error {
	start := time.Now()
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var portfolio models.Portfolio
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND user_id = ?", portfolioID, userID).
			First(&portfolio).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrPortfolioNotFound
			}
			return err
		}
		if err := tx.Where("portfolio_id = ?", portfolioID).Delete(&models.PortfolioPosition{}).Error; err != nil {
			return err
		}
		if err := tx.Where("portfolio_id = ?", portfolioID).Delete(&models.PortfolioTransaction{}).Error; err != nil {
			return err
		}
		return tx.Where("id = ? AND user_id = ?", portfolioID, userID).Delete(&models.Portfolio{}).Error
	})
	r.observe(start, err)
	return err
}

func (r *portfolioRepository) Positions(ctx context.Context, userID, portfolioID uuid.UUID) ([]models.PortfolioPosition, error) {
	start := time.Now()
	if _, err := r.Get(ctx, userID, portfolioID); err != nil {
		return nil, err
	}
	var rows []models.PortfolioPosition
	err := r.db.WithContext(ctx).
		Where("portfolio_id = ?", portfolioID).
		Order("code ASC").
		Find(&rows).Error
	r.observe(start, err)
	return rows, err
}

func (r *portfolioRepository) Transactions(
	ctx context.Context,
	userID, portfolioID uuid.UUID,
	limit int,
	before time.Time,
) ([]models.PortfolioTransaction, error) {
	start := time.Now()
	if _, err := r.Get(ctx, userID, portfolioID); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 100
	}
	if limit > 5000 {
		limit = 5000
	}
	query := r.db.WithContext(ctx).
		Where("portfolio_id = ? AND user_id = ?", portfolioID, userID)
	if !before.IsZero() {
		query = query.Where("occurred_at < ?", before)
	}
	var rows []models.PortfolioTransaction
	err := query.Order("occurred_at DESC, id DESC").Limit(limit).Find(&rows).Error
	r.observe(start, err)
	return rows, err
}

func (r *portfolioRepository) Transaction(
	ctx context.Context,
	userID, portfolioID, transactionID uuid.UUID,
) (models.PortfolioTransaction, error) {
	start := time.Now()
	var row models.PortfolioTransaction
	err := r.db.WithContext(ctx).
		Where("id = ? AND portfolio_id = ? AND user_id = ?", transactionID, portfolioID, userID).
		First(&row).Error
	r.observe(start, err)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return models.PortfolioTransaction{}, ErrTransactionNotFound
	}
	return row, err
}

func (r *portfolioRepository) CreateTransaction(
	ctx context.Context,
	userID, portfolioID uuid.UUID,
	transaction *models.PortfolioTransaction,
	project PortfolioProjector,
) (models.PortfolioTransaction, bool, error) {
	start := time.Now()
	var result models.PortfolioTransaction
	created := false
	err := r.withLockedLedger(ctx, userID, portfolioID, project, func(tx *gorm.DB) (bool, error) {
		var existing models.PortfolioTransaction
		err := tx.Where("portfolio_id = ? AND client_id = ?", portfolioID, transaction.ClientID).
			First(&existing).Error
		if err == nil {
			result = existing
			return false, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return false, err
		}
		transaction.ID = uuid.New()
		transaction.PortfolioID = portfolioID
		transaction.UserID = userID
		if err := tx.Create(transaction).Error; err != nil {
			return false, err
		}
		result = *transaction
		created = true
		return true, nil
	})
	r.observe(start, err)
	return result, created, err
}

func (r *portfolioRepository) UpdateTransaction(
	ctx context.Context,
	userID, portfolioID uuid.UUID,
	transaction *models.PortfolioTransaction,
	project PortfolioProjector,
) (models.PortfolioTransaction, error) {
	start := time.Now()
	var result models.PortfolioTransaction
	err := r.withLockedLedger(ctx, userID, portfolioID, project, func(tx *gorm.DB) (bool, error) {
		var existing models.PortfolioTransaction
		if err := tx.Where("id = ? AND portfolio_id = ? AND user_id = ?", transaction.ID, portfolioID, userID).
			First(&existing).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return false, ErrTransactionNotFound
			}
			return false, err
		}
		transaction.PortfolioID = portfolioID
		transaction.UserID = userID
		transaction.CreatedAt = existing.CreatedAt
		if transaction.ClientID == "" {
			transaction.ClientID = existing.ClientID
		}
		if err := tx.Save(transaction).Error; err != nil {
			return false, err
		}
		result = *transaction
		return true, nil
	})
	r.observe(start, err)
	return result, err
}

func (r *portfolioRepository) DeleteTransaction(
	ctx context.Context,
	userID, portfolioID, transactionID uuid.UUID,
	project PortfolioProjector,
) error {
	start := time.Now()
	err := r.withLockedLedger(ctx, userID, portfolioID, project, func(tx *gorm.DB) (bool, error) {
		result := tx.Where(
			"id = ? AND portfolio_id = ? AND user_id = ?",
			transactionID,
			portfolioID,
			userID,
		).Delete(&models.PortfolioTransaction{})
		if result.Error != nil {
			return false, result.Error
		}
		if result.RowsAffected == 0 {
			return false, ErrTransactionNotFound
		}
		return true, nil
	})
	r.observe(start, err)
	return err
}

type ledgerMutation func(tx *gorm.DB) (changed bool, err error)

func (r *portfolioRepository) withLockedLedger(
	ctx context.Context,
	userID, portfolioID uuid.UUID,
	project PortfolioProjector,
	mutate ledgerMutation,
) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var portfolio models.Portfolio
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND user_id = ?", portfolioID, userID).
			First(&portfolio).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrPortfolioNotFound
			}
			return err
		}
		changed, err := mutate(tx)
		if err != nil || !changed {
			return err
		}

		var transactions []models.PortfolioTransaction
		if err := tx.Where("portfolio_id = ?", portfolioID).
			Order("occurred_at ASC, created_at ASC, id ASC").
			Find(&transactions).Error; err != nil {
			return err
		}
		positions, netDepositsTRY, err := project(portfolioID, transactions)
		if err != nil {
			return err
		}
		if err := tx.Where("portfolio_id = ?", portfolioID).
			Delete(&models.PortfolioPosition{}).Error; err != nil {
			return err
		}
		if len(positions) > 0 {
			if err := tx.CreateInBatches(positions, 200).Error; err != nil {
				return err
			}
		}
		return tx.Model(&models.Portfolio{}).
			Where("id = ? AND user_id = ?", portfolioID, userID).
			Updates(map[string]any{
				"net_deposits_try": netDepositsTRY,
				"version":          gorm.Expr("version + 1"),
				"updated_at":       time.Now().UTC(),
			}).Error
	})
}
