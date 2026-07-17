package database_test

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/tsc-forex/tsc_forex/internal/config"
	"github.com/tsc-forex/tsc_forex/internal/database"
	"github.com/tsc-forex/tsc_forex/internal/models"
)

// TestNewUnreachable covers the connection-error path (no live Postgres needed).
func TestNewUnreachable(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	_, err := database.New(ctx, config.DatabaseConfig{
		Host:         "127.0.0.1",
		Port:         1, // refused
		User:         "u",
		Password:     "p",
		Name:         "n",
		SSLMode:      "disable",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err == nil {
		t.Fatal("expected connection error")
	}
}

func TestCheckReadyRequiresCurrentSuccessfulMigration(t *testing.T) {
	db, err := gorm.Open(
		sqlite.Open("file:database-ready-"+uuid.NewString()+"?mode=memory&cache=shared"),
		&gorm.Config{Logger: logger.Default.LogMode(logger.Silent)},
	)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	ctx := context.Background()
	if err := database.CheckReady(ctx, nil); err == nil {
		t.Fatal("nil database reported ready")
	}
	if err := database.CheckReady(ctx, db); err == nil {
		t.Fatal("unmigrated database reported ready")
	}
	if err := database.Migrate(db); err != nil {
		t.Fatal(err)
	}
	if err := database.CheckReady(ctx, db); err != nil {
		t.Fatalf("migrated database not ready: %v", err)
	}
	if err := db.Exec("UPDATE app_schema_state SET version = 0 WHERE component = ?", "core").Error; err != nil {
		t.Fatal(err)
	}
	if err := database.CheckReady(ctx, db); err == nil {
		t.Fatal("stale migration marker reported ready")
	}
	if err := db.Exec("DELETE FROM app_schema_state WHERE component = ?", "core").Error; err != nil {
		t.Fatal(err)
	}
	if err := database.CheckReady(ctx, db); err == nil {
		t.Fatal("missing migration marker reported ready")
	}
}

func TestPortfolioSchemaCanRollback(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	tables := []any{
		&models.Portfolio{},
		&models.PortfolioPosition{},
		&models.PortfolioTransaction{},
	}
	if err := db.AutoMigrate(tables...); err != nil {
		t.Fatal(err)
	}
	for _, table := range tables {
		if !db.Migrator().HasTable(table) {
			t.Fatalf("migration did not create %T", table)
		}
	}
	// Mirrors scripts/migrations/20260716_portfolios_down.sql in dependency-safe
	// order and guards against a schema that cannot be cleanly removed.
	if err := db.Migrator().DropTable(
		&models.PortfolioPosition{},
		&models.PortfolioTransaction{},
		&models.Portfolio{},
	); err != nil {
		t.Fatal(err)
	}
	for _, table := range tables {
		if db.Migrator().HasTable(table) {
			t.Fatalf("rollback left table %T", table)
		}
	}
}
