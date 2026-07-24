package repository

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/tsc-forex/tsc_forex/internal/models"
)

func TestDeleteUserRemovesPortfolioLedger(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:delete-user-"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.Portfolio{},
		&models.PortfolioPosition{},
		&models.PortfolioTransaction{},
		&models.WatchlistItem{},
		&models.PriceAlert{},
		&models.Notification{},
		&models.DeviceToken{},
		&models.RefreshToken{},
		&models.AuthIdentity{},
		&models.WebAuthnCredential{},
		&models.UserBadge{},
		&models.GamificationEvent{},
		&models.XPLedgerEntry{},
		&models.UserGamification{},
		&models.DailyPlan{},
		&models.LearningProgress{},
		&models.LearningAttempt{},
		&models.DisciplineReview{},
		&models.SeasonScore{},
		&models.PaperTrade{},
		&models.GamificationPrivacy{},
		&models.StudyGroup{},
		&models.StudyGroupMember{},
		&models.SubscriptionEntitlement{},
		&models.AppleTransaction{},
		&models.GooglePlayTransaction{},
		&models.FeatureUsage{},
		&models.FreeResearchClaim{},
		&models.MarketResearchArchive{},
		&models.MarketResearchJob{},
		&models.FeatureArtifact{},
		&models.AppAttestChallenge{},
		&models.AppAttestKey{},
		&models.DeviceCheckRedemption{},
		&models.BillingCapacityLock{},
		&models.SubscriptionRestoreTombstone{},
	); err != nil {
		t.Fatal(err)
	}
	user := models.User{ID: uuid.New(), Email: uuid.NewString() + "@example.com", Role: "user", Provider: "local"}
	portfolio := models.Portfolio{ID: uuid.New(), UserID: user.ID, Name: "Test", BaseCurrency: "TRY"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&portfolio).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.PortfolioPosition{
		ID: uuid.New(), PortfolioID: portfolio.ID, Code: "NVDA", Currency: "USD",
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.PortfolioTransaction{
		ID: uuid.New(), PortfolioID: portfolio.ID, UserID: user.ID,
		ClientID: "delete-test", Type: "buy", Code: "NVDA", Currency: "USD",
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.AppleTransaction{
		TransactionID: "delete-user-tx", OriginalTransactionID: "delete-user-original", UserID: &user.ID,
		ProductID: "talhacan.tscforex.plus.monthly", AppAccountToken: user.ID.String(),
		Environment: "Production", PurchasedAt: time.Now().UTC(), ExpiresAt: time.Now().UTC().AddDate(0, 1, 0),
		JWSSHA256: strings.Repeat("a", 64),
	}).Error; err != nil {
		t.Fatal(err)
	}

	repo := NewUserRepository(db, nil)
	if err := repo.DeleteUser(context.Background(), user.ID); !errors.Is(err, ErrSubscriptionRecoveryRequired) {
		t.Fatalf("delete without recovery token err=%v", err)
	}
	if _, err := repo.ByID(context.Background(), user.ID); err != nil {
		t.Fatalf("failed deletion must rollback user: %v", err)
	}
	recoveryToken := "local-subscription-recovery-token-0001"
	if err := repo.DeleteUser(context.Background(), user.ID, recoveryToken); err != nil {
		t.Fatal(err)
	}
	var tombstone models.SubscriptionRestoreTombstone
	if err := db.First(&tombstone, "original_transaction_id = ?", "delete-user-original").Error; err != nil ||
		tombstone.RecoveryTokenHash != subscriptionRestoreHash("recovery-token", recoveryToken) || tombstone.ProviderSubjectHash != "" {
		t.Fatalf("tombstone=%+v err=%v", tombstone, err)
	}
	googleUser := models.User{ID: uuid.New(), Email: uuid.NewString() + "@example.com", Role: "user", Provider: "google"}
	if err := db.Create(&googleUser).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.AuthIdentity{UserID: googleUser.ID, Provider: "google", Subject: "stable-google-subject"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.AppleTransaction{
		TransactionID: "delete-google-tx", OriginalTransactionID: "delete-google-original", UserID: &googleUser.ID,
		ProductID: "talhacan.tscforex.plus.monthly", AppAccountToken: googleUser.ID.String(),
		Environment: "Production", PurchasedAt: time.Now().UTC(), ExpiresAt: time.Now().UTC().AddDate(0, 1, 0),
		JWSSHA256: strings.Repeat("b", 64),
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := repo.DeleteUser(context.Background(), googleUser.ID); err != nil {
		t.Fatalf("provider-bound deletion should not require a recovery secret: %v", err)
	}
	var providerTombstone models.SubscriptionRestoreTombstone
	if err := db.First(&providerTombstone, "original_transaction_id = ?", "delete-google-original").Error; err != nil ||
		providerTombstone.IdentityProvider != "google" ||
		providerTombstone.ProviderSubjectHash != subscriptionRestoreHash("google-subject", "stable-google-subject") ||
		providerTombstone.RecoveryTokenHash != "" {
		t.Fatalf("provider tombstone=%+v err=%v", providerTombstone, err)
	}
	for name, model := range map[string]any{
		"portfolios":   &models.Portfolio{},
		"positions":    &models.PortfolioPosition{},
		"transactions": &models.PortfolioTransaction{},
	} {
		var count int64
		if err := db.Model(model).Count(&count).Error; err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("%s left %d rows", name, count)
		}
	}
}
