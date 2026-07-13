package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/tsc-forex/tsc_forex/internal/domain"
	"github.com/tsc-forex/tsc_forex/internal/models"
	"github.com/tsc-forex/tsc_forex/internal/repository"
)

func newUserAssetsDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&models.PriceTick{}, &models.WatchlistItem{}, &models.PriceAlert{},
		&models.Notification{}, &models.RefreshToken{}, &models.DeviceToken{},
	); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestPriceRepository(t *testing.T) {
	t.Parallel()
	db := newUserAssetsDB(t)
	repo := repository.NewPriceRepository(db, nil)
	ctx := context.Background()
	now := time.Now().UTC()

	for i := 0; i < 3; i++ {
		if err := repo.Save(ctx, domain.InstrumentPrice{
			Code: "GRAM_GOLD", Kind: "gold", Price: 2500 + float64(i),
			Bid: 2499, Ask: 2501, Provider: "test", Timestamp: now.Add(time.Duration(i) * time.Second),
		}); err != nil {
			t.Fatal(err)
		}
	}
	rows, err := repo.History(ctx, "GRAM_GOLD", time.Time{}, time.Time{}, 0)
	if err != nil || len(rows) != 3 {
		t.Fatalf("history rows=%d err=%v", len(rows), err)
	}
	latest, err := repo.Latest(ctx, "GRAM_GOLD")
	if err != nil || latest.Price != 2502 {
		t.Fatalf("latest=%v err=%v", latest.Price, err)
	}
}

func TestWatchlistRepository(t *testing.T) {
	t.Parallel()
	db := newUserAssetsDB(t)
	repo := repository.NewWatchlistRepository(db, nil)
	ctx := context.Background()
	uid := uuid.New()

	if _, err := repo.Add(ctx, uid, "USDTRY"); err != nil {
		t.Fatal(err)
	}
	// Idempotent add.
	if _, err := repo.Add(ctx, uid, "USDTRY"); err != nil {
		t.Fatal(err)
	}
	items, err := repo.List(ctx, uid)
	if err != nil || len(items) != 1 {
		t.Fatalf("items=%d err=%v", len(items), err)
	}
	if err := repo.Remove(ctx, uid, "USDTRY"); err != nil {
		t.Fatal(err)
	}
	items, _ = repo.List(ctx, uid)
	if len(items) != 0 {
		t.Fatalf("after remove items=%d", len(items))
	}
}

func TestAlertAndNotificationRepository(t *testing.T) {
	t.Parallel()
	db := newUserAssetsDB(t)
	alerts := repository.NewAlertRepository(db, nil)
	notifs := repository.NewNotificationRepository(db, nil)
	ctx := context.Background()
	uid := uuid.New()

	a := &models.PriceAlert{ID: uuid.New(), UserID: uid, Code: "GRAM_GOLD", Direction: models.AlertAbove, Threshold: 2600, Active: true}
	if err := alerts.Create(ctx, a); err != nil {
		t.Fatal(err)
	}
	active, err := alerts.ActiveByCode(ctx, "GRAM_GOLD")
	if err != nil || len(active) != 1 {
		t.Fatalf("active=%d err=%v", len(active), err)
	}
	if err := alerts.MarkFired(ctx, a.ID, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if err := alerts.Delete(ctx, uid, a.ID); err != nil {
		t.Fatal(err)
	}
	if err := alerts.Delete(ctx, uid, a.ID); err != repository.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	n := &models.Notification{ID: uuid.New(), UserID: uid, Code: "GRAM_GOLD", Title: "t", Body: "b", Price: 2650, CreatedAt: time.Now().UTC()}
	if err := notifs.Create(ctx, n); err != nil {
		t.Fatal(err)
	}
	list, err := notifs.ListByUser(ctx, uid, 10)
	if err != nil || len(list) != 1 {
		t.Fatalf("notifs=%d err=%v", len(list), err)
	}
	if err := notifs.MarkRead(ctx, uid, n.ID); err != nil {
		t.Fatal(err)
	}
}

func TestRefreshAndDeviceTokenRepository(t *testing.T) {
	t.Parallel()
	db := newUserAssetsDB(t)
	tokens := repository.NewRefreshTokenRepository(db, nil)
	devices := repository.NewDeviceTokenRepository(db, nil)
	ctx := context.Background()
	uid := uuid.New()

	rt := &models.RefreshToken{ID: uuid.New(), UserID: uid, TokenHash: "hash123", ExpiresAt: time.Now().Add(time.Hour), CreatedAt: time.Now().UTC()}
	if err := tokens.Store(ctx, rt); err != nil {
		t.Fatal(err)
	}
	got, err := tokens.ByHash(ctx, "hash123")
	if err != nil || got.ID != rt.ID {
		t.Fatalf("byhash err=%v", err)
	}
	if err := tokens.Revoke(ctx, rt.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := tokens.ByHash(ctx, "missing"); err != repository.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	dt := &models.DeviceToken{UserID: uid, Platform: "ios", Token: "devtok"}
	if err := devices.Upsert(ctx, dt); err != nil {
		t.Fatal(err)
	}
	// Upsert again with same token → no duplicate.
	if err := devices.Upsert(ctx, &models.DeviceToken{UserID: uid, Platform: "ios", Token: "devtok"}); err != nil {
		t.Fatal(err)
	}
	list, err := devices.ListByUser(ctx, uid)
	if err != nil || len(list) != 1 {
		t.Fatalf("devices=%d err=%v", len(list), err)
	}
}
