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
	updated, err := alerts.Update(ctx, uid, a.ID, models.AlertBelow, 2500)
	if err != nil {
		t.Fatalf("update alert: %v", err)
	}
	if updated.Direction != models.AlertBelow || updated.Threshold != 2500 {
		t.Fatalf("updated alert = %+v", updated)
	}
	if _, err := alerts.Update(ctx, uuid.New(), a.ID, models.AlertAbove, 2700); err != repository.ErrNotFound {
		t.Fatalf("cross-user update error = %v, want ErrNotFound", err)
	}

	firedAt := time.Now().UTC()
	triggeredNotification := &models.Notification{
		ID: uuid.New(), UserID: uid, AlertID: &a.ID, Code: a.Code,
		Title: "Alarm", Body: "Eşik geçildi", Price: 2490, CreatedAt: firedAt,
	}
	claimed, err := alerts.ClaimTriggered(ctx, *a, triggeredNotification, firedAt)
	if err != nil {
		t.Fatalf("claim stale alert snapshot: %v", err)
	}
	if claimed {
		t.Fatal("stale pre-edit snapshot must not claim the alert")
	}
	claimed, err = alerts.ClaimTriggered(ctx, updated, triggeredNotification, firedAt)
	if err != nil || !claimed {
		t.Fatalf("first claim = %t, err=%v", claimed, err)
	}
	claimed, err = alerts.ClaimTriggered(ctx, updated, &models.Notification{
		ID: uuid.New(), UserID: uid, AlertID: &a.ID, Code: a.Code,
		Title: "Tekrar", Body: "Gönderilmemeli", Price: 2480, CreatedAt: firedAt.Add(time.Second),
	}, firedAt.Add(time.Second))
	if err != nil || claimed {
		t.Fatalf("second claim = %t, err=%v; want false, nil", claimed, err)
	}
	active, err = alerts.ListByUser(ctx, uid)
	if err != nil || len(active) != 0 {
		t.Fatalf("triggered alert remained in active list: count=%d err=%v", len(active), err)
	}
	var stored models.PriceAlert
	if err := db.First(&stored, "id = ?", a.ID).Error; err != nil {
		t.Fatalf("read triggered alert audit row: %v", err)
	}
	if stored.Active || stored.LastFiredAt == nil {
		t.Fatalf("triggered state not persisted: %+v", stored)
	}
	if _, err := alerts.Update(ctx, uid, a.ID, models.AlertAbove, 2700); err != repository.ErrNotFound {
		t.Fatalf("triggered alert update error = %v, want ErrNotFound", err)
	}
	if err := alerts.Delete(ctx, uid, a.ID); err != nil {
		t.Fatal(err)
	}
	if err := alerts.Delete(ctx, uid, a.ID); err != repository.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	list, err := notifs.ListByUser(ctx, uid, 10)
	if err != nil || len(list) != 1 {
		t.Fatalf("notifs=%d err=%v", len(list), err)
	}
	if list[0].ID != triggeredNotification.ID || list[0].AlertID == nil || *list[0].AlertID != a.ID {
		t.Fatalf("trigger notification = %+v", list[0])
	}
	if err := notifs.MarkRead(ctx, uid, triggeredNotification.ID); err != nil {
		t.Fatal(err)
	}
}

func TestClaimTriggeredRollsBackWhenNotificationCannotBePersisted(t *testing.T) {
	t.Parallel()
	db := newUserAssetsDB(t)
	alerts := repository.NewAlertRepository(db, nil)
	notifs := repository.NewNotificationRepository(db, nil)
	ctx := context.Background()
	uid := uuid.New()
	alert := &models.PriceAlert{
		ID: uuid.New(), UserID: uid, Code: "USDTRY", Direction: models.AlertAbove,
		Threshold: 40, Active: true,
	}
	if err := alerts.Create(ctx, alert); err != nil {
		t.Fatal(err)
	}
	duplicateID := uuid.New()
	if err := notifs.Create(ctx, &models.Notification{
		ID: duplicateID, UserID: uid, Code: "USDTRY", Title: "existing", CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}
	firedAt := time.Now().UTC()
	claimed, err := alerts.ClaimTriggered(ctx, *alert, &models.Notification{
		ID: duplicateID, UserID: uid, AlertID: &alert.ID, Code: alert.Code,
		Title: "duplicate", CreatedAt: firedAt,
	}, firedAt)
	if err == nil || claimed {
		t.Fatalf("claim with failed notification insert = %t, err=%v", claimed, err)
	}
	active, err := alerts.ActiveByCode(ctx, alert.Code)
	if err != nil || len(active) != 1 || active[0].ID != alert.ID || !active[0].Active {
		t.Fatalf("alert transition was not rolled back: active=%+v err=%v", active, err)
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
