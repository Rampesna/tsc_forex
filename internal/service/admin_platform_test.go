package service

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/tsc-forex/tsc_forex/internal/config"
	"github.com/tsc-forex/tsc_forex/internal/models"
)

func TestAdminPlatformUsesSessionsWithoutPushPermission(t *testing.T) {
	t.Parallel()
	db, err := gorm.Open(sqlite.Open("file:admin-platform-"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&models.User{}, &models.RefreshToken{}, &models.DeviceToken{},
		&models.SubscriptionEntitlement{}, &models.WatchlistItem{},
		&models.PriceAlert{}, &models.UserBadge{}, &models.FeatureUsage{},
	); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	android := models.User{
		ID: uuid.New(), Email: "android@example.com", Role: "user",
		Provider: "google", CreatedAt: now, UpdatedAt: now,
	}
	ios := models.User{
		ID: uuid.New(), Email: "ios@example.com", Role: "user",
		Provider: "apple", CreatedAt: now, UpdatedAt: now,
	}
	both := models.User{
		ID: uuid.New(), Email: "both@example.com", Role: "user",
		Provider: "local", CreatedAt: now, UpdatedAt: now,
	}
	unknown := models.User{
		ID: uuid.New(), Email: "unknown@example.com", Role: "user",
		Provider: "local", CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&[]models.User{android, ios, both, unknown}).Error; err != nil {
		t.Fatal(err)
	}
	sessions := []models.RefreshToken{
		{
			ID: uuid.New(), UserID: android.ID, TokenHash: "android-session",
			Platform: "android", ExpiresAt: now.Add(time.Hour), CreatedAt: now,
		},
		{
			ID: uuid.New(), UserID: both.ID, TokenHash: "both-android-session",
			Platform: "android", ExpiresAt: now.Add(time.Hour), CreatedAt: now,
		},
	}
	if err := db.Create(&sessions).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&[]models.DeviceToken{
		{ID: uuid.New(), UserID: ios.ID, Platform: "ios", Token: "ios-push", LastSeenAt: now},
		{ID: uuid.New(), UserID: both.ID, Platform: "ios", Token: "both-ios-push", LastSeenAt: now},
	}).Error; err != nil {
		t.Fatal(err)
	}

	svc := NewAdminService(db, nil, nil, config.AdminConfig{}, config.AIConfig{}, nil)
	page, err := svc.Users(context.Background(), AdminUserQuery{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatal(err)
	}
	got := make(map[string]string, len(page.Items))
	for _, item := range page.Items {
		got[item.Email] = item.DevicePlatform
	}
	if got[android.Email] != "android" || got[ios.Email] != "ios" ||
		got[both.Email] != "both" || got[unknown.Email] != "unknown" {
		t.Fatalf("platforms=%v", got)
	}

	filtered, err := svc.Users(context.Background(), AdminUserQuery{
		Platform: "android", Page: 1, PageSize: 20,
	})
	if err != nil {
		t.Fatal(err)
	}
	if filtered.Total != 2 {
		t.Fatalf("android total=%d items=%+v", filtered.Total, filtered.Items)
	}

	distribution, err := svc.platformDistribution(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]float64{}
	for _, item := range distribution {
		counts[item.Name] = item.Value
	}
	if counts["Android"] != 1 || counts["iOS"] != 1 ||
		counts["Çoklu cihaz"] != 1 || counts["Bilinmiyor"] != 1 {
		t.Fatalf("distribution=%v", counts)
	}
}
