package database

import (
	"context"
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/tsc-forex/tsc_forex/internal/config"
	"github.com/tsc-forex/tsc_forex/internal/models"
)

// New opens a GORM Postgres connection, configures the pool, and pings.
func New(ctx context.Context, cfg config.DatabaseConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Name, cfg.SSLMode,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:                 gormlogger.Default.LogMode(gormlogger.Warn),
		SkipDefaultTransaction: true,
		PrepareStmt:            true,
	})
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("sql db: %w", err)
	}
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(pingCtx); err != nil {
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return db, nil
}

// Migrate creates/updates schema for all models and, on Postgres, enables the
// pgvector extension plus an embeddings table reserved for future AI features
// (price-pattern similarity search, anomaly detection, forecasting).
func Migrate(db *gorm.DB) error {
	if db.Dialector.Name() == "postgres" {
		// Every app pod starts with AutoMigrate. A transaction-scoped advisory
		// lock serializes DDL during rolling deploys and releases automatically
		// on commit, rollback, or connection loss.
		return db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", int64(847326451)).Error; err != nil {
				return fmt.Errorf("migration lock: %w", err)
			}
			return migrateSchema(tx)
		})
	}
	return migrateSchema(db)
}

func migrateSchema(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&models.GoldPrice{},
		&models.User{},
		&models.PriceTick{},
		&models.Candle{},
		&models.WatchlistItem{},
		&models.PriceAlert{},
		&models.Notification{},
		&models.RefreshToken{},
		&models.DeviceToken{},
		&models.WebAuthnCredential{},
		&models.AuthIdentity{},
		&models.UserGamification{},
		&models.UserBadge{},
		&models.GamificationEvent{},
		&models.XPLedgerEntry{},
		&models.DailyPlan{},
		&models.LearningProgress{},
		&models.LearningAttempt{},
		&models.DisciplineReview{},
		&models.SeasonScore{},
		&models.PaperTrade{},
		&models.GamificationPrivacy{},
		&models.StudyGroup{},
		&models.StudyGroupMember{},
		&models.Portfolio{},
		&models.PortfolioPosition{},
		&models.PortfolioTransaction{},
	); err != nil {
		return err
	}

	if db.Dialector.Name() == "postgres" {
		if err := enablePgVector(db); err != nil {
			// Non-fatal: AI features are optional; log-and-continue is done by caller.
			return fmt.Errorf("pgvector setup: %w", err)
		}
		if err := addProductionIndexes(db); err != nil {
			return fmt.Errorf("index setup: %w", err)
		}
	}
	return nil
}

// addProductionIndexes creates the composite / BRIN / partial indexes that make
// the hot queries fast at scale. All statements are idempotent.
//
// Design notes:
//   - price_ticks & gold_prices are append-only time-series. BRIN indexes on
//     created_at are tiny and ideal for time-range scans (retention, charts,
//     24h stats) on naturally time-ordered inserts.
//   - Notifications/alerts are read "by user, newest first" → composite indexes
//     with created_at DESC avoid a sort.
//   - Active alerts are matched on every tick → a partial index on active rows
//     keeps the working set small.
func addProductionIndexes(db *gorm.DB) error {
	stmts := []string{
		// Time-series range scans (charts, stats, retention cleanup).
		`CREATE INDEX IF NOT EXISTS idx_price_ticks_created_brin ON price_ticks USING BRIN (created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_price_ticks_code_created ON price_ticks (code, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_gold_prices_created_brin ON gold_prices USING BRIN (created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_gold_prices_symbol_created ON gold_prices (symbol, created_at DESC)`,

		// Pre-aggregated candles: charts read by (code, resolution, bucket range).
		`CREATE INDEX IF NOT EXISTS idx_candles_code_res_bucket ON candles (code, resolution, bucket DESC)`,

		// Per-user, newest-first reads.
		`CREATE INDEX IF NOT EXISTS idx_notifications_user_created ON notifications (user_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_alerts_user_created ON price_alerts (user_id, created_at DESC)`,

		// Alert matching runs on every tick: only active alerts matter.
		`CREATE INDEX IF NOT EXISTS idx_alerts_active_code ON price_alerts (code) WHERE active = true`,

		// Refresh-token lookups by hash are already unique; add expiry sweep index.
		`CREATE INDEX IF NOT EXISTS idx_refresh_tokens_expires ON refresh_tokens (expires_at)`,

		// Device fan-out for push: list a user's devices quickly.
		`CREATE INDEX IF NOT EXISTS idx_device_tokens_user ON device_tokens (user_id)`,

		// Gamification: daily quests read today's events; the weekly leaderboard
		// sums XP since the week start. Composite (user, created_at) + a
		// (day_key) index keep both cheap.
		`CREATE INDEX IF NOT EXISTS idx_gam_events_user_created ON gamification_events (user_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_gam_events_created ON gamification_events (created_at)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_xp_ledger_user_key ON xp_ledger (user_id, idempotency_key)`,
		`CREATE INDEX IF NOT EXISTS idx_xp_ledger_user_created ON xp_ledger (user_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_learning_attempts_user_created ON learning_attempts (user_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_season_scores_rank ON season_scores (season_code, points DESC, updated_at)`,
		`CREATE INDEX IF NOT EXISTS idx_paper_trades_user_created ON paper_trades (user_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_study_group_members_user ON study_group_members (user_id, joined_at DESC)`,

		// Portfolio hot reads and idempotent offline outbox synchronization.
		`CREATE INDEX IF NOT EXISTS idx_portfolios_user_updated ON portfolios (user_id, updated_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_portfolio_positions_portfolio ON portfolio_positions (portfolio_id, code)`,
		`CREATE INDEX IF NOT EXISTS idx_portfolio_transactions_time ON portfolio_transactions (portfolio_id, occurred_at DESC, id)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_portfolio_transactions_client ON portfolio_transactions (portfolio_id, client_id)`,
	}
	for _, s := range stmts {
		if err := db.Exec(s).Error; err != nil {
			return err
		}
	}
	return nil
}

// enablePgVector installs the extension and an embeddings table. Idempotent.
func enablePgVector(db *gorm.DB) error {
	stmts := []string{
		`CREATE EXTENSION IF NOT EXISTS vector`,
		`CREATE TABLE IF NOT EXISTS price_embeddings (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			code VARCHAR(32) NOT NULL,
			ts TIMESTAMPTZ NOT NULL DEFAULT now(),
			embedding vector(64),
			meta JSONB
		)`,
		`CREATE INDEX IF NOT EXISTS idx_price_embeddings_code ON price_embeddings (code)`,
	}
	for _, s := range stmts {
		if err := db.Exec(s).Error; err != nil {
			return err
		}
	}
	return nil
}

// Close releases the underlying connection pool.
func Close(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
