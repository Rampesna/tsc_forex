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

type fakeGoogleVerifier struct {
	info GooglePlayPurchaseInfo
	err  error
}

func (f fakeGoogleVerifier) Ready() bool { return true }
func (f fakeGoogleVerifier) VerifySubscription(context.Context, string, string, string) (GooglePlayPurchaseInfo, error) {
	return f.info, f.err
}

func TestSyncGooglePurchaseGrantsEntitlement(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:google-billing?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&models.User{}, &models.SubscriptionEntitlement{}, &models.GooglePlayTransaction{},
		&models.FeatureUsage{}, &models.FreeResearchClaim{}, &models.AppAttestKey{},
		&models.PriceAlert{}, &models.WatchlistItem{}, &models.Portfolio{},
	); err != nil {
		t.Fatal(err)
	}
	userID := uuid.New()
	if err := db.Create(&models.User{ID: userID, Email: "play@example.com", PasswordHash: "x"}).Error; err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 7, 24, 12, 0, 0, 0, time.UTC)
	svc := NewSubscriptionService(db, config.BillingConfig{
		Enabled: true, BundleID: "talhacan.tscforex", AllowSandbox: true,
		GooglePackageName: "com.talhacan.tscforex", SandboxMaxLifetime: 24 * time.Hour,
	}, nil).WithGooglePlay(fakeGoogleVerifier{info: GooglePlayPurchaseInfo{
		PackageName:         "com.talhacan.tscforex",
		ProductID:           "talhacan.tscforex.plus.monthly",
		PurchaseToken:       "token-abc",
		OrderID:             "GPA.1234",
		ObfuscatedAccountID: userID.String(),
		Environment:         "Sandbox",
		PurchasedAt:         now,
		ExpiresAt:           now.Add(30 * 24 * time.Hour),
		Acknowledged:        false,
		RawJSON:             []byte(`{"ok":true}`),
	}})
	svc.now = func() time.Time { return now }

	status, err := svc.SyncGooglePurchase(context.Background(), userID, "com.talhacan.tscforex", "talhacan.tscforex.plus.monthly", "token-abc")
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	if status.Plan.Code != PlanPlus {
		t.Fatalf("plan = %s", status.Plan.Code)
	}
	if status.PurchaseChannel != "google_play" {
		t.Fatalf("channel = %s", status.PurchaseChannel)
	}
	if status.Entitlement == nil || status.Entitlement.ProductID != "talhacan.tscforex.plus.monthly" {
		t.Fatalf("entitlement: %+v", status.Entitlement)
	}

	var row models.GooglePlayTransaction
	if err := db.Where("purchase_token = ?", "token-abc").First(&row).Error; err != nil {
		t.Fatal(err)
	}
	if row.UserID == nil || *row.UserID != userID {
		t.Fatalf("transaction user: %+v", row.UserID)
	}
}

func TestSyncGooglePurchaseRejectsOtherAccount(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:google-billing-conflict?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&models.User{}, &models.SubscriptionEntitlement{}, &models.GooglePlayTransaction{},
		&models.FeatureUsage{}, &models.PriceAlert{}, &models.WatchlistItem{}, &models.Portfolio{},
	); err != nil {
		t.Fatal(err)
	}
	owner := uuid.New()
	other := uuid.New()
	_ = db.Create(&models.User{ID: owner, Email: "a@example.com", PasswordHash: "x"})
	_ = db.Create(&models.User{ID: other, Email: "b@example.com", PasswordHash: "x"})

	now := time.Now().UTC()
	svc := NewSubscriptionService(db, config.BillingConfig{
		Enabled: true, AllowSandbox: true, GooglePackageName: "com.talhacan.tscforex",
	}, nil).WithGooglePlay(fakeGoogleVerifier{info: GooglePlayPurchaseInfo{
		PackageName:         "com.talhacan.tscforex",
		ProductID:           "talhacan.tscforex.pro.monthly",
		PurchaseToken:       "shared-token",
		OrderID:             "GPA.999",
		ObfuscatedAccountID: owner.String(),
		Environment:         "Production",
		PurchasedAt:         now,
		ExpiresAt:           now.Add(24 * time.Hour),
		RawJSON:             []byte(`{}`),
	}})
	if _, err := svc.SyncGooglePurchase(context.Background(), owner, "com.talhacan.tscforex", "talhacan.tscforex.pro.monthly", "shared-token"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SyncGooglePurchase(context.Background(), other, "com.talhacan.tscforex", "talhacan.tscforex.pro.monthly", "shared-token"); err == nil {
		t.Fatal("expected account conflict")
	}
}
