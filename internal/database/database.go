package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	gormlogger "gorm.io/gorm/logger"

	"github.com/tsc-forex/tsc_forex/internal/config"
	"github.com/tsc-forex/tsc_forex/internal/models"
)

const (
	currentSchemaVersion          int64 = 2026072705
	clientPlatformBackfillVersion int64 = 2026072705
)

type schemaState struct {
	Component string    `gorm:"type:varchar(64);primaryKey"`
	Version   int64     `gorm:"not null"`
	UpdatedAt time.Time `gorm:"not null"`
}

func (schemaState) TableName() string { return "app_schema_state" }

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
	if err := expandManagerOwnershipColumns(db); err != nil {
		return fmt.Errorf("manager realm migration: %w", err)
	}
	if err := db.AutoMigrate(
		&schemaState{},
		&models.GoldPrice{},
		&models.User{},
		&models.Manager{},
		&models.ManagerSession{},
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
		&models.SubscriptionEntitlement{},
		&models.AppleTransaction{},
		&models.GooglePlayTransaction{},
		&models.FeatureUsage{},
		&models.FreeResearchClaim{},
		&models.MarketResearchArchive{},
		&models.MarketResearchJob{},
		&models.FeatureArtifact{},
		&models.BillingCapacityLock{},
		&models.AppAttestChallenge{},
		&models.AppAttestKey{},
		&models.DeviceCheckRedemption{},
		&models.SubscriptionRestoreTombstone{},
		&models.NotificationCampaign{},
		&models.NotificationCampaignRecipient{},
		&models.AdminAuditLog{},
	); err != nil {
		return err
	}
	if err := contractManagerRealmColumns(db); err != nil {
		return fmt.Errorf("manager realm cleanup: %w", err)
	}

	if db.Dialector.Name() == "postgres" {
		needsClientPlatformBackfill, err := schemaVersionBefore(db, clientPlatformBackfillVersion)
		if err != nil {
			return fmt.Errorf("client platform backfill state: %w", err)
		}
		if needsClientPlatformBackfill {
			if err := backfillLegacyAndroidSessions(db); err != nil {
				return fmt.Errorf("client platform backfill: %w", err)
			}
		}
		if err := enablePgVector(db); err != nil {
			// The production image includes pgvector; failure means this schema
			// migration did not complete and the readiness marker must not advance.
			return fmt.Errorf("pgvector setup: %w", err)
		}
		if err := addProductionIndexes(db); err != nil {
			return fmt.Errorf("index setup: %w", err)
		}
	}
	return recordSchemaReady(db)
}

func schemaVersionBefore(db *gorm.DB, version int64) (bool, error) {
	var state schemaState
	err := db.Where("component = ?", "core").Take(&state).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return true, nil
	case err != nil:
		return false, err
	default:
		return state.Version < version, nil
	}
}

func recordSchemaReady(db *gorm.DB) error {
	state := schemaState{Component: "core", Version: currentSchemaVersion, UpdatedAt: time.Now().UTC()}
	if err := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "component"}},
		DoUpdates: clause.Assignments(map[string]any{
			// A rolling deployment can briefly start an old replica after a new
			// replica (for example when the HPA scales during rollout). Schema
			// readiness is monotonic: an old binary must never lower a marker
			// already advanced by a newer binary.
			"version": gorm.Expr(`
				CASE WHEN app_schema_state.version > EXCLUDED.version
				THEN app_schema_state.version ELSE EXCLUDED.version END
			`),
			"updated_at": gorm.Expr(`
				CASE WHEN app_schema_state.version > EXCLUDED.version
					THEN app_schema_state.updated_at ELSE EXCLUDED.updated_at END
			`),
		}),
	}).Create(&state).Error; err != nil {
		return fmt.Errorf("schema readiness marker: %w", err)
	}
	return nil
}

// CheckReady verifies both live database connectivity and the durable marker
// written only after every required migration/index step succeeds. It is kept
// deliberately cheap because Kubernetes calls it every few seconds.
func CheckReady(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return errors.New("database unavailable")
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("database handle: %w", err)
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("database ping: %w", err)
	}
	var state schemaState
	if err := db.WithContext(ctx).Where("component = ?", "core").Take(&state).Error; err != nil {
		return fmt.Errorf("schema readiness state: %w", err)
	}
	if state.Version < currentSchemaVersion {
		return fmt.Errorf("schema version %d is older than required %d", state.Version, currentSchemaVersion)
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
		`CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_platform ON refresh_tokens (user_id, platform) WHERE platform IN ('ios', 'android')`,
		`CREATE INDEX IF NOT EXISTS idx_manager_sessions_manager_created ON manager_sessions (manager_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_manager_sessions_expires ON manager_sessions (expires_at)`,

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

		// Billing/quota hot paths: entitlement expiry checks, monthly usage and
		// research retention cleanup all run on authenticated feature requests.
		`CREATE INDEX IF NOT EXISTS idx_entitlements_status_expiry ON subscription_entitlements (status, expires_at)`,
		`CREATE INDEX IF NOT EXISTS idx_apple_transactions_original_expiry ON apple_transactions (original_transaction_id, expires_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_feature_usage_period ON feature_usages (user_id, feature, period_key)`,
		`CREATE INDEX IF NOT EXISTS idx_research_archive_expiry ON market_research_archives (expires_at)`,
		`CREATE INDEX IF NOT EXISTS idx_research_jobs_provider ON market_research_jobs (provider_response_id) WHERE provider_response_id <> ''`,
		`CREATE INDEX IF NOT EXISTS idx_app_attest_challenges_expiry ON app_attest_challenges (expires_at)`,

		// Admin operations: campaign workers claim pending recipients newest
		// first, while the audit screen reads privileged actions by time.
		`CREATE INDEX IF NOT EXISTS idx_campaigns_status_created ON notification_campaigns (status, created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_campaign_recipients_work ON notification_campaign_recipients (campaign_id, status, created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_admin_audit_created ON admin_audit_logs (created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_device_tokens_platform_seen ON device_tokens (platform, last_seen_at DESC)`,
	}
	for _, s := range stmts {
		if err := db.Exec(s).Error; err != nil {
			return err
		}
	}
	return nil
}

// backfillLegacyAndroidSessions repairs the platform blind spot that existed
// before native-client metadata was stored on refresh sessions. The current
// production unknown cohort is the Android closed-beta cohort; users already
// identified by an APNs/FCM token are deliberately excluded. The statement is
// idempotent and never fabricates a push token or notification permission.
func backfillLegacyAndroidSessions(db *gorm.DB) error {
	return db.Exec(`
		UPDATE refresh_tokens AS r
		SET platform = 'android'
		WHERE COALESCE(r.platform, '') = ''
		  AND EXISTS (
			SELECT 1 FROM users AS u
			WHERE u.id = r.user_id AND u.role = 'user'
		  )
		  AND NOT EXISTS (
			SELECT 1 FROM device_tokens AS d
			WHERE d.user_id = r.user_id AND d.platform IN ('ios', 'android')
		  )
	`).Error
}

// expandManagerOwnershipColumns is the expand phase of a zero-downtime schema
// migration. Old pods keep using admin_id/actor_id while new pods use
// manager_id. Legacy columns can be removed in a later release after every old
// pod is guaranteed gone.
func expandManagerOwnershipColumns(db *gorm.DB) error {
	if db.Dialector.Name() != "postgres" {
		return nil
	}
	migrator := db.Migrator()
	if migrator.HasTable("notification_campaigns") {
		if !migrator.HasColumn("notification_campaigns", "manager_id") {
			if err := db.Exec(`ALTER TABLE notification_campaigns ADD COLUMN manager_id UUID`).Error; err != nil {
				return err
			}
		}
		if migrator.HasColumn("notification_campaigns", "admin_id") {
			if err := db.Exec(`UPDATE notification_campaigns SET manager_id = admin_id WHERE manager_id IS NULL`).Error; err != nil {
				return err
			}
			if err := db.Exec(`ALTER TABLE notification_campaigns ALTER COLUMN admin_id DROP NOT NULL`).Error; err != nil {
				return err
			}
		}
	}
	if migrator.HasTable("admin_audit_logs") {
		if !migrator.HasColumn("admin_audit_logs", "manager_id") {
			if err := db.Exec(`ALTER TABLE admin_audit_logs ADD COLUMN manager_id UUID`).Error; err != nil {
				return err
			}
		}
		if migrator.HasColumn("admin_audit_logs", "actor_id") {
			if err := db.Exec(`UPDATE admin_audit_logs SET manager_id = actor_id WHERE manager_id IS NULL`).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

// contractManagerRealmColumns permanently removes the compatibility columns
// used by the former user-backed admin implementation. This runs only after
// the manager tables and replacement ownership columns have been migrated.
func contractManagerRealmColumns(db *gorm.DB) error {
	legacyColumns := []struct {
		table  string
		column string
	}{
		{table: "users", column: "must_change_password"},
		{table: "users", column: "password_changed_at"},
		{table: "notification_campaigns", column: "admin_id"},
		{table: "admin_audit_logs", column: "actor_id"},
	}
	migrator := db.Migrator()
	for _, legacy := range legacyColumns {
		if !migrator.HasTable(legacy.table) || !migrator.HasColumn(legacy.table, legacy.column) {
			continue
		}
		statement := fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", legacy.table, legacy.column)
		if err := db.Exec(statement).Error; err != nil {
			return fmt.Errorf("drop %s: %w", legacy.column, err)
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
