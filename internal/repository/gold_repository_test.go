package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"github.com/tsc-forex/tsc_forex/internal/domain"
	"github.com/tsc-forex/tsc_forex/internal/models"
	"github.com/tsc-forex/tsc_forex/internal/repository"
)

func newSQLiteDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.GoldPrice{}, &models.User{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestGoldRepositorySaveHistoryLatest(t *testing.T) {
	t.Parallel()
	db := newSQLiteDB(t)
	repo := repository.NewGoldRepository(db, nil)
	ctx := context.Background()

	now := time.Now().UTC()
	for i := 0; i < 5; i++ {
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
	if len(rows) != 5 {
		t.Fatalf("history rows = %d", len(rows))
	}

	// History with no filters + default limit.
	all, err := repo.History(ctx, "", time.Time{}, time.Time{}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 5 {
		t.Fatalf("all rows = %d", len(all))
	}

	latest, err := repo.Latest(ctx, domain.SymbolGramGoldTRY)
	if err != nil {
		t.Fatal(err)
	}
	if latest.Price != 2504 {
		t.Fatalf("latest price = %v, want 2504", latest.Price)
	}
}

func TestGoldRepositoryUnsupportedInterval(t *testing.T) {
	t.Parallel()
	db := newSQLiteDB(t)
	repo := repository.NewGoldRepository(db, nil)
	if _, err := repo.Aggregate(context.Background(), domain.SymbolGramGoldTRY, "3s", time.Time{}, time.Time{}); err == nil {
		t.Fatal("expected unsupported interval error")
	}
	if repository.SupportedInterval("1m") != true || repository.SupportedInterval("3s") != false {
		t.Fatal("SupportedInterval wrong")
	}
}

func TestUserRepository(t *testing.T) {
	t.Parallel()
	db := newSQLiteDB(t)
	repo := repository.NewUserRepository(db, nil)
	ctx := context.Background()

	if _, err := repo.ByEmail(ctx, "none@e.com"); err != repository.ErrUserNotFound {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}

	u := &models.User{Email: "a@b.com", PasswordHash: "hash", Role: "user"}
	if err := repo.Create(ctx, u); err != nil {
		t.Fatal(err)
	}
	if u.ID.String() == "00000000-0000-0000-0000-000000000000" {
		t.Fatal("id not assigned")
	}

	got, err := repo.ByEmail(ctx, "a@b.com")
	if err != nil {
		t.Fatal(err)
	}
	if got.Email != "a@b.com" || got.Role != "user" {
		t.Fatalf("got %+v", got)
	}
}
