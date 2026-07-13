//go:build integration

package repository_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/tsc-forex/tsc_forex/internal/config"
	"github.com/tsc-forex/tsc_forex/internal/database"
	"github.com/tsc-forex/tsc_forex/internal/domain"
	"github.com/tsc-forex/tsc_forex/internal/repository"
)

// Run with: go test -tags=integration ./internal/repository/...
// Requires a reachable Postgres (env APP_DATABASE_* or defaults).
func TestGoldRepositoryIntegration(t *testing.T) {
	cfg := config.DatabaseConfig{
		Host:         env("APP_DATABASE_HOST", "localhost"),
		Port:         5432,
		User:         env("APP_DATABASE_USER", "tsc_forex"),
		Password:     env("APP_DATABASE_PASSWORD", "tsc_forex"),
		Name:         env("APP_DATABASE_NAME", "tsc_forex"),
		SSLMode:      "disable",
		MaxOpenConns: 5,
		MaxIdleConns: 2,
	}
	ctx := context.Background()
	db, err := database.New(ctx, cfg)
	if err != nil {
		t.Skipf("postgres not available: %v", err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatal(err)
	}
	repo := repository.NewGoldRepository(db, nil)

	now := time.Now().UTC()
	for i := 0; i < 10; i++ {
		g := domain.GramGold{
			Symbol:    domain.SymbolGramGoldTRY,
			Price:     2500 + float64(i),
			MidPrice:  2500 + float64(i),
			Spread:    0.5,
			XAUUSD:    2300,
			USDTRY:    34,
			Provider:  "test",
			Timestamp: now.Add(time.Duration(i) * time.Second),
		}
		if err := repo.Save(ctx, g); err != nil {
			t.Fatal(err)
		}
	}

	rows, err := repo.History(ctx, domain.SymbolGramGoldTRY, now.Add(-time.Minute), now.Add(time.Minute), 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) < 10 {
		t.Fatalf("history rows = %d", len(rows))
	}

	candles, err := repo.Aggregate(ctx, domain.SymbolGramGoldTRY, "1m", now.Add(-time.Minute), now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if len(candles) == 0 {
		t.Fatal("no candles")
	}
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
