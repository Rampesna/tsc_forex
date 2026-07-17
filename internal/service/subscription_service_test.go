package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/tsc-forex/tsc_forex/internal/config"
	"github.com/tsc-forex/tsc_forex/internal/models"
)

const subscriptionTestBundle = "talhacan.tscforex"

var subscriptionTestNow = time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)

type fakeAppleVerifier struct {
	transactions    map[string]AppleTransactionInfo
	notifications   map[string]AppleNotificationInfo
	txErr           map[string]error
	notificationErr map[string]error
}

func (f *fakeAppleVerifier) VerifyTransaction(signedJWS string) (AppleTransactionInfo, error) {
	if err := f.txErr[signedJWS]; err != nil {
		return AppleTransactionInfo{}, err
	}
	info, ok := f.transactions[signedJWS]
	if !ok {
		return AppleTransactionInfo{}, errors.New("unknown transaction")
	}
	return info, nil
}

func (f *fakeAppleVerifier) VerifyNotification(signedJWS string) (AppleNotificationInfo, error) {
	if err := f.notificationErr[signedJWS]; err != nil {
		return AppleNotificationInfo{}, err
	}
	info, ok := f.notifications[signedJWS]
	if !ok {
		return AppleNotificationInfo{}, errors.New("unknown notification")
	}
	return info, nil
}

func newSubscriptionTestService(t *testing.T, verifier AppleSignedDataVerifier) (*SubscriptionService, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(
		sqlite.Open("file:subscription-"+uuid.NewString()+"?mode=memory&cache=shared"),
		&gorm.Config{Logger: logger.Default.LogMode(logger.Silent)},
	)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(
		&models.User{},
		&models.AuthIdentity{},
		&models.SubscriptionEntitlement{},
		&models.AppleTransaction{},
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
		&models.WatchlistItem{},
		&models.PriceAlert{},
		&models.Portfolio{},
	); err != nil {
		t.Fatal(err)
	}
	svc := NewSubscriptionService(db, config.BillingConfig{
		Enabled: true, BundleID: subscriptionTestBundle, AllowSandbox: true,
	}, verifier)
	svc.now = func() time.Time { return subscriptionTestNow }
	return svc, db
}

type fakeAppAttestVerifier struct {
	mu             sync.Mutex
	attestCalls    int
	assertionCalls int
}

func (f *fakeAppAttestVerifier) Ready() bool { return true }

func (f *fakeAppAttestVerifier) VerifyAttestation(_ []byte, _ []byte, _ string) (AppAttestResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.attestCalls++
	return AppAttestResult{PublicKeyDER: []byte("fake-public-key"), Receipt: []byte("fake-receipt"), Environment: "production"}, nil
}

func (f *fakeAppAttestVerifier) VerifyAssertion(_ []byte, _ []byte, _ []byte, previous uint32) (uint32, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.assertionCalls++
	return previous + 1, nil
}

type fakeDeviceCheckVerifier struct {
	mu                 sync.Mutex
	claimed            bool
	queryCalls         int
	markCalls          int
	failAfterMarkOnce  bool
	failBeforeMarkOnce bool
}

func (f *fakeDeviceCheckVerifier) Ready() bool { return true }

func (f *fakeDeviceCheckVerifier) FreeTrialClaimed(_ context.Context, _ string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.queryCalls++
	return f.claimed, nil
}

func (f *fakeDeviceCheckVerifier) MarkFreeTrialClaimed(_ context.Context, _ string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.markCalls++
	if f.failBeforeMarkOnce {
		f.failBeforeMarkOnce = false
		return ErrDeviceCheckUnavailable
	}
	f.claimed = true
	if f.failAfterMarkOnce {
		f.failAfterMarkOnce = false
		return ErrDeviceCheckUnavailable
	}
	return nil
}

func enableFakeDeviceTrust(svc *SubscriptionService, app *fakeAppAttestVerifier, device *fakeDeviceCheckVerifier) {
	svc.cfg.AppAttestEnabled = true
	svc.cfg.DeviceCheckEnabled = true
	svc.appAttest = app
	svc.deviceCheck = device
}

func registerFakeAppAttestKey(t *testing.T, svc *SubscriptionService, userID uuid.UUID, deviceID, keyID, requestID, code string) AppAttestProof {
	t.Helper()
	ctx := context.Background()
	first, err := svc.IssueAppAttestChallenge(ctx, userID, deviceID, keyID, requestID, code)
	if err != nil || !first.NeedsKeyAttestation {
		t.Fatalf("first challenge=%+v err=%v", first, err)
	}
	if err := svc.RegisterAppAttestKey(ctx, userID, deviceID, keyID, first.ChallengeID, []byte("fake-attestation")); err != nil {
		t.Fatal(err)
	}
	second, err := svc.IssueAppAttestChallenge(ctx, userID, deviceID, keyID, requestID, code)
	if err != nil || second.NeedsKeyAttestation {
		t.Fatalf("assertion challenge=%+v err=%v", second, err)
	}
	return AppAttestProof{
		KeyID: keyID, ChallengeID: second.ChallengeID, Assertion: []byte("fake-assertion"),
		DeviceCheckToken: base64.StdEncoding.EncodeToString(make([]byte, 32)),
	}
}

func seedSubscriptionEntitlement(t *testing.T, db *gorm.DB, userID uuid.UUID, planCode, originalID string) models.SubscriptionEntitlement {
	t.Helper()
	plan := planForCode(planCode)
	entitlement := models.SubscriptionEntitlement{
		UserID: userID, ProductID: plan.ProductID, PlanCode: plan.Code,
		OriginalTransactionID: originalID, LatestTransactionID: originalID + "-latest",
		Environment: "Production", Status: "active",
		PeriodStart: subscriptionTestNow.AddDate(0, 0, -2), ExpiresAt: subscriptionTestNow.AddDate(0, 1, 0),
	}
	if err := db.Create(&entitlement).Error; err != nil {
		t.Fatal(err)
	}
	return entitlement
}

func requireQuotaError(t *testing.T, err error, code, feature string, limit, used int) {
	t.Helper()
	var quotaErr *QuotaError
	if !errors.As(err, &quotaErr) {
		t.Fatalf("error = %v, want *QuotaError", err)
	}
	if quotaErr.Code != code || quotaErr.Feature != feature || quotaErr.Limit != limit || quotaErr.Used != used {
		t.Fatalf("quota error = %+v, want code=%s feature=%s limit=%d used=%d", quotaErr, code, feature, limit, used)
	}
}

func TestBillingReadinessFailsForEnabledInvalidDeviceCheck(t *testing.T) {
	verifier := &fakeAppleVerifier{transactions: map[string]AppleTransactionInfo{}, notifications: map[string]AppleNotificationInfo{}}
	svc, _ := newSubscriptionTestService(t, verifier)
	svc.cfg.AppAttestEnabled = true
	svc.cfg.DeviceCheckEnabled = true
	svc.appAttest = &fakeAppAttestVerifier{}
	svc.deviceCheck = &fakeDeviceCheckVerifier{}
	if err := svc.ReadinessError(); err != nil {
		t.Fatalf("valid injected verifiers not ready: %v", err)
	}
	svc.deviceCheck = NewAppleDeviceCheckVerifier(svc.cfg)
	if err := svc.ReadinessError(); err == nil {
		t.Fatal("enabled DeviceCheck without mounted key reported ready")
	}
}

func TestSubscriptionFreeResearchIsOncePerUserAndDevice(t *testing.T) {
	svc, db := newSubscriptionTestService(t, nil)
	ctx := context.Background()
	userA, userB, userC := uuid.New(), uuid.New(), uuid.New()
	deviceA := "ios-keychain-device-a-0001"
	deviceB := "ios-keychain-device-b-0002"

	first, err := svc.ConsumeAIResearch(ctx, userA, deviceA, "free-request-a", "GRAM_GOLD")
	if err != nil {
		t.Fatal(err)
	}
	if !first.FreeTrial || first.Plan.Code != PlanFree || first.Used != 1 || first.Remaining != 0 {
		t.Fatalf("first consumption = %+v", first)
	}

	// Network retry with exactly the same request and scope is free.
	retry, err := svc.ConsumeAIResearch(ctx, userA, deviceA, "free-request-a", "GRAM_GOLD")
	if err != nil || retry != first {
		t.Fatalf("idempotent retry = %+v, err=%v", retry, err)
	}
	var claims int64
	if err := db.Model(&models.FreeResearchClaim{}).Count(&claims).Error; err != nil || claims != 1 {
		t.Fatalf("claims=%d err=%v", claims, err)
	}

	if _, err := svc.ConsumeAIResearch(ctx, userA, deviceA, "free-request-a", "USDTRY"); !errors.Is(err, ErrRequestConflict) {
		t.Fatalf("same request with another scope error=%v, want ErrRequestConflict", err)
	}
	if _, err := svc.ConsumeAIResearch(ctx, userA, deviceB, "free-request-b", "USDTRY"); !errors.Is(err, ErrFreeResearchUnavailable) {
		t.Fatalf("same account with new device error=%v, want ErrFreeResearchUnavailable", err)
	}
	if _, err := svc.ConsumeAIResearch(ctx, userB, deviceA, "free-request-c", "ONS_GOLD"); !errors.Is(err, ErrFreeResearchUnavailable) {
		t.Fatalf("new account on claimed device error=%v, want ErrFreeResearchUnavailable", err)
	}
	if _, err := svc.ConsumeAIResearch(ctx, userC, deviceB, "free-request-d", "EURTRY"); err != nil {
		t.Fatalf("independent account/device should receive a trial: %v", err)
	}
}

func TestSubscriptionPaidMonthlyQuotaIdempotencyAndExhaustion(t *testing.T) {
	svc, db := newSubscriptionTestService(t, nil)
	ctx := context.Background()
	userID := uuid.New()
	seedSubscriptionEntitlement(t, db, userID, PlanPlus, "paid-monthly")
	limit := planForCode(PlanPlus).Limits.AIResearch

	first, err := svc.ConsumeAIResearch(ctx, userID, "ignored-for-paid", "paid-request-01", "GRAM_GOLD")
	if err != nil {
		t.Fatal(err)
	}
	if first.FreeTrial || first.Limit != limit || first.Used != 1 || first.Remaining != limit-1 {
		t.Fatalf("first paid consumption = %+v", first)
	}
	retry, err := svc.ConsumeAIResearch(ctx, userID, "ignored-for-paid", "paid-request-01", "GRAM_GOLD")
	if err != nil || retry.Used != 1 || retry.Remaining != limit-1 {
		t.Fatalf("paid retry = %+v err=%v", retry, err)
	}
	if _, err := svc.ConsumeAIResearch(ctx, userID, "ignored-for-paid", "paid-request-01", "USDTRY"); !errors.Is(err, ErrRequestConflict) {
		t.Fatalf("scope conflict error=%v, want ErrRequestConflict", err)
	}

	for i := 2; i <= limit; i++ {
		consumption, consumeErr := svc.ConsumeAIResearch(ctx, userID, "ignored-for-paid", fmt.Sprintf("paid-request-%02d", i), "GRAM_GOLD")
		if consumeErr != nil {
			t.Fatalf("consume %d: %v", i, consumeErr)
		}
		if consumption.Used != i || consumption.Remaining != limit-i {
			t.Fatalf("consume %d = %+v", i, consumption)
		}
	}
	_, err = svc.ConsumeAIResearch(ctx, userID, "ignored-for-paid", "paid-request-over", "GRAM_GOLD")
	requireQuotaError(t, err, "quota_exhausted", FeatureAIResearch, limit, limit)

	status, err := svc.Status(ctx, userID, "")
	if err != nil {
		t.Fatal(err)
	}
	quota := status.Usage[FeatureAIResearch]
	if status.Plan.Code != PlanPlus || quota.Used != limit || quota.Remaining != 0 || quota.PeriodEnd == nil {
		t.Fatalf("billing status=%+v quota=%+v", status, quota)
	}
}

func TestSubscriptionCapacityLimitsCountOnlyCurrentResources(t *testing.T) {
	svc, db := newSubscriptionTestService(t, nil)
	ctx := context.Background()
	userID := uuid.New()

	for i := 0; i < freePlan.Limits.Watchlist; i++ {
		if err := db.Create(&models.WatchlistItem{UserID: userID, Code: fmt.Sprintf("ASSET_%02d", i)}).Error; err != nil {
			t.Fatal(err)
		}
	}
	requireQuotaError(t, svc.EnsureCapacity(ctx, userID, FeatureWatchlist), "quota_exhausted", FeatureWatchlist, freePlan.Limits.Watchlist, freePlan.Limits.Watchlist)

	if err := db.Create(&models.Portfolio{UserID: userID, Name: "Ana"}).Error; err != nil {
		t.Fatal(err)
	}
	requireQuotaError(t, svc.EnsureCapacity(ctx, userID, FeaturePortfolios), "quota_exhausted", FeaturePortfolios, freePlan.Limits.Portfolios, freePlan.Limits.Portfolios)

	for i := 0; i < freePlan.Limits.ActiveAlerts; i++ {
		if err := db.Create(&models.PriceAlert{UserID: userID, Code: fmt.Sprintf("FX_%02d", i), Direction: models.AlertAbove, Threshold: 1, Active: true}).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Create(&models.PriceAlert{UserID: userID, Code: "INACTIVE", Direction: models.AlertBelow, Threshold: 1, Active: false}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&models.PriceAlert{}).Where("user_id = ? AND code = ?", userID, "INACTIVE").Update("active", false).Error; err != nil {
		t.Fatal(err)
	}
	requireQuotaError(t, svc.EnsureCapacity(ctx, userID, FeatureActiveAlerts), "quota_exhausted", FeatureActiveAlerts, freePlan.Limits.ActiveAlerts, freePlan.Limits.ActiveAlerts)

	if err := db.Where("user_id = ?", userID).First(&models.WatchlistItem{}).Delete(&models.WatchlistItem{}).Error; err != nil {
		t.Fatal(err)
	}
	if err := svc.EnsureCapacity(ctx, userID, FeatureWatchlist); err != nil {
		t.Fatalf("capacity should reopen after removal: %v", err)
	}
}

func TestSubscriptionResearchArchiveRetentionAndHistory(t *testing.T) {
	svc, db := newSubscriptionTestService(t, nil)
	ctx := context.Background()
	userID := uuid.New()
	seedSubscriptionEntitlement(t, db, userID, PlanPlus, "archive-subscription")

	if err := svc.ArchiveResearch(ctx, userID, "archive-request", "gram_gold", []byte(`{"title":"Altın","model":"internal-provider-model"}`)); err != nil {
		t.Fatal(err)
	}
	// Duplicate request IDs are idempotent and do not replace the first payload.
	if err := svc.ArchiveResearch(ctx, userID, "archive-request", "USDTRY", []byte(`{"title":"Değiştir"}`)); err != nil {
		t.Fatal(err)
	}
	if err := svc.ArchiveResearch(ctx, userID, "invalid-json", "USDTRY", []byte(`not-json`)); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.MarketResearchArchive{
		UserID: userID, RequestKey: "expired-request", Code: "EURTRY", Payload: `{"expired":true}`,
		GeneratedAt: subscriptionTestNow.Add(-48 * time.Hour), ExpiresAt: subscriptionTestNow.Add(-time.Hour),
	}).Error; err != nil {
		t.Fatal(err)
	}

	history, err := svc.ResearchHistory(ctx, userID, 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 1 || history[0].Code != "GRAM_GOLD" || string(history[0].Payload) != `{"title":"Altın"}` {
		t.Fatalf("history=%+v", history)
	}
	wantExpiry := subscriptionTestNow.AddDate(0, 0, planForCode(PlanPlus).Limits.ResearchArchiveDays)
	if !history[0].GeneratedAt.Equal(subscriptionTestNow) || !history[0].ExpiresAt.Equal(wantExpiry) {
		t.Fatalf("archive times generated=%s expires=%s want expires=%s", history[0].GeneratedAt, history[0].ExpiresAt, wantExpiry)
	}
}

func TestSubscriptionAppleSyncValidationAndAccountSharing(t *testing.T) {
	ctx := context.Background()
	userA, userB := uuid.New(), uuid.New()
	base := AppleTransactionInfo{
		TransactionID: "tx-valid", OriginalTransactionID: "original-valid",
		BundleID: subscriptionTestBundle, ProductID: planForCode(PlanPlus).ProductID,
		AppAccountToken: userA.String(),
		Environment:     "Production", PurchasedAt: subscriptionTestNow.Add(-time.Hour),
		ExpiresAt: subscriptionTestNow.AddDate(0, 1, 0),
	}
	verifier := &fakeAppleVerifier{transactions: map[string]AppleTransactionInfo{}, notifications: map[string]AppleNotificationInfo{}}
	svc, _ := newSubscriptionTestService(t, verifier)

	wrongBundle := base
	wrongBundle.BundleID = "com.example.copy"
	verifier.transactions["wrong-bundle"] = wrongBundle
	if _, err := svc.SyncAppleTransaction(ctx, userA, "wrong-bundle"); !errors.Is(err, ErrInvalidAppleTransaction) {
		t.Fatalf("wrong bundle error=%v", err)
	}
	wrongProduct := base
	wrongProduct.ProductID = "talhacan.tscforex.not-a-plan"
	verifier.transactions["wrong-product"] = wrongProduct
	if _, err := svc.SyncAppleTransaction(ctx, userA, "wrong-product"); !errors.Is(err, ErrInvalidAppleTransaction) {
		t.Fatalf("wrong product error=%v", err)
	}
	wrongAccount := base
	wrongAccount.AppAccountToken = userB.String()
	verifier.transactions["wrong-account-token"] = wrongAccount
	if _, err := svc.SyncAppleTransaction(ctx, userA, "wrong-account-token"); !errors.Is(err, ErrTransactionAccount) {
		t.Fatalf("wrong appAccountToken error=%v", err)
	}

	verifier.transactions["valid-a"] = base
	status, err := svc.SyncAppleTransaction(ctx, userA, "valid-a")
	if err != nil {
		t.Fatal(err)
	}
	if status.Plan.Code != PlanPlus || status.Entitlement == nil || status.Entitlement.Status != "active" {
		t.Fatalf("status=%+v", status)
	}
	shared := base
	shared.TransactionID = "tx-shared-copy"
	shared.AppAccountToken = userB.String()
	verifier.transactions["shared-b"] = shared
	if _, err := svc.SyncAppleTransaction(ctx, userB, "shared-b"); !errors.Is(err, ErrTransactionAccount) {
		t.Fatalf("shared original transaction error=%v, want ErrTransactionAccount", err)
	}
}

func TestSubscriptionAppleNotificationRenewsAndRevokes(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	initial := AppleTransactionInfo{
		TransactionID: "tx-initial", OriginalTransactionID: "original-notification",
		BundleID: subscriptionTestBundle, ProductID: planForCode(PlanPro).ProductID,
		AppAccountToken: userID.String(), Environment: "Production",
		PurchasedAt: subscriptionTestNow.AddDate(0, 0, -20), ExpiresAt: subscriptionTestNow.AddDate(0, 0, 10),
	}
	renewal := initial
	renewal.TransactionID = "tx-renewal"
	renewal.PurchasedAt = subscriptionTestNow
	renewal.ExpiresAt = subscriptionTestNow.AddDate(0, 1, 0)
	revoked := renewal
	revokedAt := subscriptionTestNow.Add(time.Hour)
	revoked.RevokedAt = &revokedAt
	verifier := &fakeAppleVerifier{
		transactions: map[string]AppleTransactionInfo{"initial": initial, "renewal": renewal, "revoked": revoked},
		notifications: map[string]AppleNotificationInfo{
			"renew-notification":  {NotificationType: "DID_RENEW", NotificationUUID: "notification-renew", BundleID: subscriptionTestBundle, Environment: "Production", SignedTransactionInfo: "renewal"},
			"revoke-notification": {NotificationType: "REVOKE", NotificationUUID: "notification-revoke", BundleID: subscriptionTestBundle, Environment: "Production", SignedTransactionInfo: "revoked"},
			"wrong-notification":  {NotificationType: "DID_RENEW", NotificationUUID: "notification-wrong", BundleID: "com.example.copy", Environment: "Production", SignedTransactionInfo: "renewal"},
		},
	}
	svc, db := newSubscriptionTestService(t, verifier)
	if _, err := svc.SyncAppleTransaction(ctx, userID, "initial"); err != nil {
		t.Fatal(err)
	}
	if err := svc.ProcessAppleNotification(ctx, "wrong-notification"); !errors.Is(err, ErrInvalidAppleTransaction) {
		t.Fatalf("wrong notification error=%v", err)
	}
	if err := svc.ProcessAppleNotification(ctx, "renew-notification"); err != nil {
		t.Fatal(err)
	}
	var entitlement models.SubscriptionEntitlement
	if err := db.First(&entitlement, "user_id = ?", userID).Error; err != nil {
		t.Fatal(err)
	}
	if entitlement.LatestTransactionID != renewal.TransactionID || !entitlement.ExpiresAt.Equal(renewal.ExpiresAt) || entitlement.Status != "active" {
		t.Fatalf("renewed entitlement=%+v", entitlement)
	}
	if err := svc.ProcessAppleNotification(ctx, "revoke-notification"); err != nil {
		t.Fatal(err)
	}
	if err := db.First(&entitlement, "user_id = ?", userID).Error; err != nil {
		t.Fatal(err)
	}
	if entitlement.Status != "revoked" || entitlement.RevokedAt == nil || !entitlement.RevokedAt.Equal(revokedAt) {
		t.Fatalf("revoked entitlement=%+v", entitlement)
	}
}

func TestAIResearchJobConcurrentReplayStartsOnlyOnceAndCreatorOnlyRefunds(t *testing.T) {
	svc, db := newSubscriptionTestService(t, nil)
	app := &fakeAppAttestVerifier{}
	device := &fakeDeviceCheckVerifier{}
	enableFakeDeviceTrust(svc, app, device)
	userID := uuid.New()
	deviceID := "ios-keychain-concurrent-device-0001"
	requestID := "research-concurrent-request"
	proof := registerFakeAppAttestKey(t, svc, userID, deviceID, "fake-key-concurrent", requestID, "GRAM_GOLD")

	const workers = 12
	results := make(chan AIResearchJob, workers)
	errorsCh := make(chan error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			job, err := svc.BeginAIResearchJob(context.Background(), userID, deviceID, requestID, "GRAM_GOLD", proof)
			results <- job
			errorsCh <- err
		}()
	}
	wg.Wait()
	close(results)
	close(errorsCh)
	for err := range errorsCh {
		if err != nil {
			t.Fatalf("concurrent begin: %v", err)
		}
	}
	created := 0
	creatorToken := ""
	for job := range results {
		if job.Created {
			created++
			creatorToken = job.OwnerToken
		}
	}
	if created != 1 || creatorToken == "" {
		t.Fatalf("created=%d creator token present=%t", created, creatorToken != "")
	}
	var jobs, claims int64
	_ = db.Model(&models.MarketResearchJob{}).Count(&jobs).Error
	_ = db.Model(&models.FreeResearchClaim{}).Count(&claims).Error
	if jobs != 1 || claims != 1 || device.markCalls != 1 || app.assertionCalls != 1 {
		t.Fatalf("jobs=%d claims=%d device marks=%d assertions=%d", jobs, claims, device.markCalls, app.assertionCalls)
	}
	if err := svc.ReleaseAIResearchJob(context.Background(), userID, requestID, "not-the-owner", "test"); !errors.Is(err, ErrRequestConflict) {
		t.Fatalf("non-owner refund err=%v", err)
	}
	if err := svc.AttachAIResearchProvider(context.Background(), userID, requestID, creatorToken, "response-one"); err != nil {
		t.Fatal(err)
	}
	if err := svc.ReleaseAIResearchJob(context.Background(), userID, requestID, creatorToken, "too-late"); !errors.Is(err, ErrRequestConflict) {
		t.Fatalf("pending job refund err=%v", err)
	}
	payload := []byte(`{"title":"durable result"}`)
	if err := svc.CompleteAIResearchJob(context.Background(), userID, requestID, "", "response-one", payload); err != nil {
		t.Fatal(err)
	}
	replay, err := svc.BeginAIResearchJob(context.Background(), userID, deviceID, requestID, "GRAM_GOLD", AppAttestProof{})
	if err != nil || replay.Created || replay.Status != ResearchJobCompleted || string(replay.ResultPayload) != string(payload) {
		t.Fatalf("replay=%+v err=%v", replay, err)
	}
}

func TestAIResearchStaleLeaseReclaimsWithoutSpendingQuotaTwice(t *testing.T) {
	svc, db := newSubscriptionTestService(t, nil)
	userID := uuid.New()
	seedSubscriptionEntitlement(t, db, userID, PlanPlus, "research-lease-subscription")
	requestID := "research-stale-lease"

	first, err := svc.BeginAIResearchJob(context.Background(), userID, "paid-device", requestID, "USDTRY", AppAttestProof{})
	if err != nil || !first.Created || first.OwnerToken == "" {
		t.Fatalf("first=%+v err=%v", first, err)
	}
	if replay, replayErr := svc.LeaseAIResearchJob(context.Background(), userID, requestID, "USDTRY"); replayErr != nil || replay.Created {
		t.Fatalf("active lease replay=%+v err=%v", replay, replayErr)
	}

	svc.now = func() time.Time { return subscriptionTestNow.Add(researchJobLeaseDuration + time.Second) }
	reclaimed, err := svc.LeaseAIResearchJob(context.Background(), userID, requestID, "USDTRY")
	if err != nil || !reclaimed.Created || reclaimed.OwnerToken == "" || reclaimed.OwnerToken == first.OwnerToken {
		t.Fatalf("reclaimed=%+v err=%v", reclaimed, err)
	}
	if err := svc.AttachAIResearchProvider(context.Background(), userID, requestID, first.OwnerToken, "stale-owner-response"); !errors.Is(err, ErrRequestConflict) {
		t.Fatalf("stale owner attach err=%v", err)
	}
	if err := svc.AttachAIResearchProvider(context.Background(), userID, requestID, reclaimed.OwnerToken, "reclaimed-response"); err != nil {
		t.Fatal(err)
	}

	var usage int64
	if err := db.Model(&models.FeatureUsage{}).
		Where("user_id = ? AND feature = ? AND request_key = ?", userID, FeatureAIResearch, requestID).
		Count(&usage).Error; err != nil || usage != 1 {
		t.Fatalf("usage=%d err=%v", usage, err)
	}
	var row models.MarketResearchJob
	if err := db.Where("user_id = ? AND request_key = ?", userID, requestID).First(&row).Error; err != nil ||
		row.AttemptCount != 2 || row.Status != ResearchJobPending || row.LeaseExpiresAt != nil {
		t.Fatalf("row=%+v err=%v", row, err)
	}
}

func TestAIResearchDispatchedInvalidResultDoesNotRefundQuota(t *testing.T) {
	svc, db := newSubscriptionTestService(t, nil)
	userID := uuid.New()
	seedSubscriptionEntitlement(t, db, userID, PlanPlus, "research-failure-subscription")
	requestID := "research-provider-invalid"
	job, err := svc.BeginAIResearchJob(context.Background(), userID, "paid-device", requestID, "USDTRY", AppAttestProof{})
	if err != nil || !job.Created {
		t.Fatalf("job=%+v err=%v", job, err)
	}
	if err := svc.MarkAIResearchReservationFailed(context.Background(), userID, requestID, job.OwnerToken, "invalid_result"); err != nil {
		t.Fatal(err)
	}
	if err := svc.ReleaseAIResearchJob(context.Background(), userID, requestID, job.OwnerToken, "late_refund"); !errors.Is(err, ErrRequestConflict) {
		t.Fatalf("terminal failure refund err=%v", err)
	}
	replay, err := svc.BeginAIResearchJob(context.Background(), userID, "paid-device", requestID, "USDTRY", AppAttestProof{})
	if err != nil || replay.Created || replay.Status != ResearchJobFailed {
		t.Fatalf("replay=%+v err=%v", replay, err)
	}
	var usage int64
	if err := db.Model(&models.FeatureUsage{}).
		Where("user_id = ? AND feature = ? AND request_key = ?", userID, FeatureAIResearch, requestID).
		Count(&usage).Error; err != nil || usage != 1 {
		t.Fatalf("usage=%d err=%v", usage, err)
	}
}

func TestDeviceCheckSagaReconcilesLostUpdateResponseWithoutBurningRequest(t *testing.T) {
	svc, db := newSubscriptionTestService(t, nil)
	app := &fakeAppAttestVerifier{}
	device := &fakeDeviceCheckVerifier{failAfterMarkOnce: true}
	enableFakeDeviceTrust(svc, app, device)
	userID := uuid.New()
	deviceID := "ios-keychain-crash-recovery-0001"
	requestID := "research-crash-recovery"
	proof := registerFakeAppAttestKey(t, svc, userID, deviceID, "fake-key-crash", requestID, "USDTRY")

	if _, err := svc.BeginAIResearchJob(context.Background(), userID, deviceID, requestID, "USDTRY", proof); !errors.Is(err, ErrDeviceCheckUnavailable) {
		t.Fatalf("first begin err=%v, want lost Apple response", err)
	}
	var journal models.DeviceCheckRedemption
	if err := db.Where("user_id = ? AND request_key = ?", userID, requestID).First(&journal).Error; err != nil || journal.Status != deviceRedemptionUpdating {
		t.Fatalf("journal=%+v err=%v", journal, err)
	}
	job, err := svc.BeginAIResearchJob(context.Background(), userID, deviceID, requestID, "USDTRY", proof)
	if err != nil || !job.Created {
		t.Fatalf("reconciled begin=%+v err=%v", job, err)
	}
	if device.markCalls != 1 || device.queryCalls != 2 {
		t.Fatalf("query=%d mark=%d; retry must reconcile instead of updating twice", device.queryCalls, device.markCalls)
	}
	if err := db.Where("user_id = ? AND request_key = ?", userID, requestID).First(&journal).Error; err != nil || journal.Status != deviceRedemptionVerified {
		t.Fatalf("verified journal=%+v err=%v", journal, err)
	}
}

func TestDeviceCheckStaleUpdatingLeaseRetriesWhenAppleBitIsStillClear(t *testing.T) {
	svc, db := newSubscriptionTestService(t, nil)
	app := &fakeAppAttestVerifier{}
	device := &fakeDeviceCheckVerifier{failBeforeMarkOnce: true}
	enableFakeDeviceTrust(svc, app, device)
	userID := uuid.New()
	deviceID := "ios-keychain-stale-devicecheck-0001"
	requestID := "research-stale-devicecheck"
	proof := registerFakeAppAttestKey(t, svc, userID, deviceID, "fake-key-stale-devicecheck", requestID, "EURTRY")

	if _, err := svc.BeginAIResearchJob(context.Background(), userID, deviceID, requestID, "EURTRY", proof); !errors.Is(err, ErrDeviceCheckUnavailable) {
		t.Fatalf("first begin err=%v", err)
	}
	var journal models.DeviceCheckRedemption
	if err := db.Where("user_id = ? AND request_key = ?", userID, requestID).First(&journal).Error; err != nil ||
		journal.Status != deviceRedemptionUpdating || journal.AttemptCount != 1 || journal.LeaseExpiresAt == nil {
		t.Fatalf("journal after lost update=%+v err=%v", journal, err)
	}

	svc.now = func() time.Time { return subscriptionTestNow.Add(deviceRedemptionLeaseDuration + time.Second) }
	job, err := svc.BeginAIResearchJob(context.Background(), userID, deviceID, requestID, "EURTRY", proof)
	if err != nil || !job.Created {
		t.Fatalf("reclaimed begin=%+v err=%v", job, err)
	}
	if device.markCalls != 2 {
		t.Fatalf("mark calls=%d want=2", device.markCalls)
	}
	var verifiedJournal models.DeviceCheckRedemption
	if err := db.Where("user_id = ? AND request_key = ?", userID, requestID).First(&verifiedJournal).Error; err != nil ||
		verifiedJournal.Status != deviceRedemptionVerified || verifiedJournal.AttemptCount != 2 || verifiedJournal.LeaseExpiresAt != nil {
		t.Fatalf("verified journal=%+v err=%v", verifiedJournal, err)
	}
	var claims int64
	if err := db.Model(&models.FreeResearchClaim{}).Where("user_id = ?", userID).Count(&claims).Error; err != nil || claims != 1 {
		t.Fatalf("claims=%d err=%v", claims, err)
	}
}

func TestDeviceCheckSameTokenRaceHasSingleWinner(t *testing.T) {
	svc, _ := newSubscriptionTestService(t, nil)
	app := &fakeAppAttestVerifier{}
	device := &fakeDeviceCheckVerifier{}
	enableFakeDeviceTrust(svc, app, device)
	userA, userB := uuid.New(), uuid.New()
	sharedToken := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{7}, 32))
	proofA := registerFakeAppAttestKey(t, svc, userA, "ios-device-race-account-a", "key-race-a", "request-race-a", "EURTRY")
	proofB := registerFakeAppAttestKey(t, svc, userB, "ios-device-race-account-b", "key-race-b", "request-race-b", "EURTRY")
	proofA.DeviceCheckToken, proofB.DeviceCheckToken = sharedToken, sharedToken

	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for _, input := range []struct {
		user            uuid.UUID
		device, request string
		proof           AppAttestProof
	}{{userA, "ios-device-race-account-a", "request-race-a", proofA}, {userB, "ios-device-race-account-b", "request-race-b", proofB}} {
		input := input
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := svc.BeginAIResearchJob(context.Background(), input.user, input.device, input.request, "EURTRY", input.proof)
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	success, rejected := 0, 0
	for err := range errs {
		switch {
		case err == nil:
			success++
		case errors.Is(err, ErrFreeResearchUnavailable):
			rejected++
		default:
			t.Fatalf("unexpected race error: %v", err)
		}
	}
	if success != 1 || rejected != 1 {
		t.Fatalf("success=%d rejected=%d", success, rejected)
	}
}

func TestFeatureArtifactReplaysOriginalOutputAndRejectsScopeReuse(t *testing.T) {
	svc, db := newSubscriptionTestService(t, nil)
	userID := uuid.New()
	seedSubscriptionEntitlement(t, db, userID, PlanPlus, "artifact-subscription")
	first, err := svc.BeginFeatureArtifact(context.Background(), userID, FeaturePortfolioHealth, "health-request", "portfolio-a")
	if err != nil || !first.Created || first.OwnerToken == "" {
		t.Fatalf("first=%+v err=%v", first, err)
	}
	payload, _ := json.Marshal(map[string]any{"score": 81, "generated_at": "immutable"})
	if err := svc.CompleteFeatureArtifact(context.Background(), userID, FeaturePortfolioHealth, "health-request", first.OwnerToken, payload, "application/json", ""); err != nil {
		t.Fatal(err)
	}
	replay, err := svc.BeginFeatureArtifact(context.Background(), userID, FeaturePortfolioHealth, "health-request", "portfolio-a")
	if err != nil || replay.Created || string(replay.Payload) != string(payload) {
		t.Fatalf("replay=%+v err=%v", replay, err)
	}
	if _, err := svc.BeginFeatureArtifact(context.Background(), userID, FeaturePortfolioHealth, "health-request", "portfolio-b"); !errors.Is(err, ErrRequestConflict) {
		t.Fatalf("cross-scope replay err=%v", err)
	}
}

func TestFeatureArtifactStaleLeaseReclaimsWithoutSpendingQuotaTwice(t *testing.T) {
	svc, db := newSubscriptionTestService(t, nil)
	userID := uuid.New()
	seedSubscriptionEntitlement(t, db, userID, PlanPlus, "artifact-lease-subscription")
	requestID := "artifact-stale-lease"

	first, err := svc.BeginFeatureArtifact(context.Background(), userID, FeaturePortfolioHealth, requestID, "portfolio-a")
	if err != nil || !first.Created || first.OwnerToken == "" {
		t.Fatalf("first=%+v err=%v", first, err)
	}
	if _, err := svc.BeginFeatureArtifact(context.Background(), userID, FeaturePortfolioHealth, requestID, "portfolio-a"); !errors.Is(err, ErrOperationInProgress) {
		t.Fatalf("active duplicate err=%v", err)
	}

	svc.now = func() time.Time { return subscriptionTestNow.Add(featureArtifactLeaseDuration + time.Second) }
	reclaimed, err := svc.BeginFeatureArtifact(context.Background(), userID, FeaturePortfolioHealth, requestID, "portfolio-a")
	if err != nil || !reclaimed.Created || reclaimed.OwnerToken == "" || reclaimed.OwnerToken == first.OwnerToken {
		t.Fatalf("reclaimed=%+v err=%v", reclaimed, err)
	}
	payload := []byte(`{"score":91}`)
	if err := svc.CompleteFeatureArtifact(context.Background(), userID, FeaturePortfolioHealth, requestID, first.OwnerToken, payload, "application/json", ""); !errors.Is(err, ErrRequestConflict) {
		t.Fatalf("stale owner completion err=%v", err)
	}
	if err := svc.CompleteFeatureArtifact(context.Background(), userID, FeaturePortfolioHealth, requestID, reclaimed.OwnerToken, payload, "application/json", ""); err != nil {
		t.Fatal(err)
	}

	var usage int64
	if err := db.Model(&models.FeatureUsage{}).
		Where("user_id = ? AND feature = ? AND request_key = ?", userID, FeaturePortfolioHealth, requestID).
		Count(&usage).Error; err != nil || usage != 1 {
		t.Fatalf("usage=%d err=%v", usage, err)
	}
	var row models.FeatureArtifact
	if err := db.Where("user_id = ? AND feature = ? AND request_key = ?", userID, FeaturePortfolioHealth, requestID).First(&row).Error; err != nil ||
		row.AttemptCount != 2 || row.Status != ResearchJobCompleted || row.LeaseExpiresAt != nil {
		t.Fatalf("row=%+v err=%v", row, err)
	}
}

func TestCapacityCountAndInsertIsAtomicForEveryResource(t *testing.T) {
	svc, db := newSubscriptionTestService(t, nil)
	ctx := context.Background()

	watchUser := uuid.New()
	for i := 0; i < freePlan.Limits.Watchlist+4; i++ {
		_, _, _ = svc.AddWatchlistWithinCapacity(ctx, watchUser, fmt.Sprintf("WATCH_%02d", i))
	}
	var watchCount int64
	_ = db.Model(&models.WatchlistItem{}).Where("user_id = ?", watchUser).Count(&watchCount).Error
	if int(watchCount) != freePlan.Limits.Watchlist {
		t.Fatalf("watchlist count=%d", watchCount)
	}

	alertUser := uuid.New()
	for i := 0; i < freePlan.Limits.ActiveAlerts+4; i++ {
		_ = svc.CreateAlertWithinCapacity(ctx, &models.PriceAlert{UserID: alertUser, Code: fmt.Sprintf("ALERT_%02d", i), Active: true})
	}
	var alertCount int64
	_ = db.Model(&models.PriceAlert{}).Where("user_id = ? AND active = ?", alertUser, true).Count(&alertCount).Error
	if int(alertCount) != freePlan.Limits.ActiveAlerts {
		t.Fatalf("alert count=%d", alertCount)
	}

	portfolioUser := uuid.New()
	for i := 0; i < freePlan.Limits.Portfolios+4; i++ {
		_ = svc.CreatePortfolioWithinCapacity(ctx, &models.Portfolio{UserID: portfolioUser, Name: fmt.Sprintf("P%d", i)})
	}
	var portfolioCount int64
	_ = db.Model(&models.Portfolio{}).Where("user_id = ?", portfolioUser).Count(&portfolioCount).Error
	if int(portfolioCount) != freePlan.Limits.Portfolios {
		t.Fatalf("portfolio count=%d", portfolioCount)
	}
}

func TestAppleRevocationIsMonotonicAgainstSameAndOlderReplay(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	initial := AppleTransactionInfo{
		TransactionID: "tx-old", OriginalTransactionID: "original-monotonic",
		BundleID: subscriptionTestBundle, ProductID: planForCode(PlanPro).ProductID,
		AppAccountToken: userID.String(), Environment: "Production",
		PurchasedAt: subscriptionTestNow.AddDate(0, 0, -20), ExpiresAt: subscriptionTestNow.AddDate(0, 0, 10),
	}
	latest := initial
	latest.TransactionID = "tx-latest"
	latest.PurchasedAt = subscriptionTestNow
	latest.ExpiresAt = subscriptionTestNow.AddDate(0, 1, 0)
	revoked := latest
	revokedAt := subscriptionTestNow.Add(time.Hour)
	revoked.RevokedAt = &revokedAt
	verifier := &fakeAppleVerifier{transactions: map[string]AppleTransactionInfo{
		"initial": initial, "latest-active": latest, "latest-revoked": revoked,
	}, notifications: map[string]AppleNotificationInfo{}}
	svc, db := newSubscriptionTestService(t, verifier)
	for _, jws := range []string{"initial", "latest-active", "latest-revoked", "latest-active", "initial"} {
		if _, err := svc.SyncAppleTransaction(ctx, userID, jws); err != nil {
			t.Fatalf("sync %s: %v", jws, err)
		}
	}
	var entitlement models.SubscriptionEntitlement
	if err := db.First(&entitlement, "user_id = ?", userID).Error; err != nil {
		t.Fatal(err)
	}
	if entitlement.Status != "revoked" || entitlement.LatestTransactionID != latest.TransactionID || entitlement.RevokedAt == nil {
		t.Fatalf("entitlement reactivated by replay: %+v", entitlement)
	}
	var transaction models.AppleTransaction
	if err := db.First(&transaction, "transaction_id = ?", latest.TransactionID).Error; err != nil || transaction.RevokedAt == nil {
		t.Fatalf("transaction revocation cleared: %+v err=%v", transaction, err)
	}
	newer := latest
	newer.TransactionID = "tx-genuinely-newer"
	newer.PurchasedAt = latest.ExpiresAt
	newer.ExpiresAt = latest.ExpiresAt.AddDate(0, 1, 0)
	verifier.transactions["newer"] = newer
	if _, err := svc.SyncAppleTransaction(ctx, userID, "newer"); err != nil {
		t.Fatal(err)
	}
	if err := db.First(&entitlement, "user_id = ?", userID).Error; err != nil || entitlement.Status != "active" || entitlement.LatestTransactionID != newer.TransactionID {
		t.Fatalf("genuine later renewal not activated: %+v err=%v", entitlement, err)
	}
}

func TestDeletedAccountSubscriptionRebindRequiresSameAppleIdentity(t *testing.T) {
	ctx := context.Background()
	oldUser, restoredUser, attacker := uuid.New(), uuid.New(), uuid.New()
	info := AppleTransactionInfo{
		TransactionID: "tx-restore", OriginalTransactionID: "original-restore",
		BundleID: subscriptionTestBundle, ProductID: planForCode(PlanPlus).ProductID,
		AppAccountToken: oldUser.String(), Environment: "Production",
		PurchasedAt: subscriptionTestNow, ExpiresAt: subscriptionTestNow.AddDate(0, 1, 0),
	}
	verifier := &fakeAppleVerifier{transactions: map[string]AppleTransactionInfo{"restore": info}, notifications: map[string]AppleNotificationInfo{}}
	svc, db := newSubscriptionTestService(t, verifier)
	for _, user := range []models.User{{ID: restoredUser, Email: "restored@example.com"}, {ID: attacker, Email: "attacker@example.com"}} {
		if err := db.Create(&user).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Create(&models.AuthIdentity{UserID: restoredUser, Provider: "apple", Subject: "apple-owner-subject"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.AuthIdentity{UserID: attacker, Provider: "apple", Subject: "different-subject"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.SubscriptionRestoreTombstone{
		OriginalTransactionID:       info.OriginalTransactionID,
		PreviousAppAccountTokenHash: restoreBindingHash("app-account-token", oldUser.String()),
		IdentityProvider:            "apple",
		ProviderSubjectHash:         restoreBindingHash("apple-subject", "apple-owner-subject"), DeletedAt: subscriptionTestNow,
	}).Error; err != nil {
		t.Fatal(err)
	}
	crafted := info
	crafted.TransactionID = "tx-restore-crafted-account-token"
	crafted.AppAccountToken = attacker.String()
	verifier.transactions["crafted-account-token"] = crafted
	if _, err := svc.SyncAppleTransaction(ctx, attacker, "crafted-account-token"); !errors.Is(err, ErrTransactionAccount) {
		t.Fatalf("tombstone bypass with freshly chosen appAccountToken err=%v", err)
	}
	if _, err := svc.SyncAppleTransaction(ctx, attacker, "restore"); !errors.Is(err, ErrTransactionAccount) {
		t.Fatalf("attacker restore err=%v", err)
	}
	status, err := svc.SyncAppleTransaction(ctx, restoredUser, "restore")
	if err != nil || status.Plan.Code != PlanPlus {
		t.Fatalf("owner restore status=%+v err=%v", status, err)
	}
	if _, err := svc.SyncAppleTransaction(ctx, attacker, "restore"); !errors.Is(err, ErrTransactionAccount) {
		t.Fatalf("rebound subscription hijack err=%v", err)
	}
}

func TestSandboxEntitlementHasPurchaseBoundLifetime(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	info := AppleTransactionInfo{
		TransactionID: "sandbox-tx", OriginalTransactionID: "sandbox-original",
		BundleID: subscriptionTestBundle, ProductID: planForCode(PlanPlus).ProductID,
		AppAccountToken: userID.String(), Environment: "Sandbox",
		PurchasedAt: subscriptionTestNow, ExpiresAt: subscriptionTestNow.AddDate(0, 3, 0),
	}
	verifier := &fakeAppleVerifier{transactions: map[string]AppleTransactionInfo{"sandbox": info}, notifications: map[string]AppleNotificationInfo{}}
	svc, db := newSubscriptionTestService(t, verifier)
	svc.cfg.SandboxMaxLifetime = 24 * time.Hour
	if _, err := svc.SyncAppleTransaction(ctx, userID, "sandbox"); err != nil {
		t.Fatal(err)
	}
	var entitlement models.SubscriptionEntitlement
	if err := db.First(&entitlement, "user_id = ?", userID).Error; err != nil {
		t.Fatal(err)
	}
	want := info.PurchasedAt.Add(24 * time.Hour)
	if !entitlement.ExpiresAt.Equal(want) {
		t.Fatalf("sandbox expiry=%s want=%s", entitlement.ExpiresAt, want)
	}
	svc.now = func() time.Time { return subscriptionTestNow.Add(12 * time.Hour) }
	if _, err := svc.SyncAppleTransaction(ctx, userID, "sandbox"); err != nil {
		t.Fatal(err)
	}
	if err := db.First(&entitlement, "user_id = ?", userID).Error; err != nil || !entitlement.ExpiresAt.Equal(want) {
		t.Fatalf("sandbox replay extended expiry=%s err=%v", entitlement.ExpiresAt, err)
	}
}

func TestDeletedLocalAndGoogleAccountsRestoreWithoutTransactionHijack(t *testing.T) {
	ctx := context.Background()
	oldLocal, localOwner, localAttacker := uuid.New(), uuid.New(), uuid.New()
	oldGoogle, googleOwner, googleAttacker := uuid.New(), uuid.New(), uuid.New()
	localInfo := AppleTransactionInfo{
		TransactionID: "tx-local-restore", OriginalTransactionID: "original-local-restore",
		BundleID: subscriptionTestBundle, ProductID: planForCode(PlanPlus).ProductID,
		AppAccountToken: oldLocal.String(), Environment: "Production",
		PurchasedAt: subscriptionTestNow, ExpiresAt: subscriptionTestNow.AddDate(0, 1, 0),
	}
	googleInfo := localInfo
	googleInfo.TransactionID = "tx-google-restore"
	googleInfo.OriginalTransactionID = "original-google-restore"
	googleInfo.AppAccountToken = oldGoogle.String()
	verifier := &fakeAppleVerifier{transactions: map[string]AppleTransactionInfo{
		"local": localInfo, "google": googleInfo,
	}, notifications: map[string]AppleNotificationInfo{}}
	svc, db := newSubscriptionTestService(t, verifier)
	for _, user := range []models.User{
		{ID: localOwner, Email: "local-owner@example.com"}, {ID: localAttacker, Email: "local-attacker@example.com"},
		{ID: googleOwner, Email: "google-owner@example.com"}, {ID: googleAttacker, Email: "google-attacker@example.com"},
	} {
		if err := db.Create(&user).Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, identity := range []models.AuthIdentity{
		{UserID: googleOwner, Provider: "google", Subject: "stable-google-subject"},
		{UserID: googleAttacker, Provider: "google", Subject: "other-google-subject"},
	} {
		if err := db.Create(&identity).Error; err != nil {
			t.Fatal(err)
		}
	}
	localRecovery := "local-recovery-secret-at-least-32-characters"
	if err := db.Create(&models.SubscriptionRestoreTombstone{
		OriginalTransactionID:       localInfo.OriginalTransactionID,
		PreviousAppAccountTokenHash: restoreBindingHash("app-account-token", oldLocal.String()),
		RecoveryTokenHash:           restoreBindingHash("recovery-token", localRecovery), DeletedAt: subscriptionTestNow,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.SubscriptionRestoreTombstone{
		OriginalTransactionID:       googleInfo.OriginalTransactionID,
		PreviousAppAccountTokenHash: restoreBindingHash("app-account-token", oldGoogle.String()),
		IdentityProvider:            "google", ProviderSubjectHash: restoreBindingHash("google-subject", "stable-google-subject"),
		DeletedAt: subscriptionTestNow,
	}).Error; err != nil {
		t.Fatal(err)
	}

	if _, err := svc.SyncAppleTransaction(ctx, localAttacker, "local", "wrong-recovery-secret-at-least-32-chars"); !errors.Is(err, ErrTransactionAccount) {
		t.Fatalf("local hijack err=%v", err)
	}
	if status, err := svc.SyncAppleTransaction(ctx, localOwner, "local", localRecovery); err != nil || status.Plan.Code != PlanPlus {
		t.Fatalf("local restore status=%+v err=%v", status, err)
	}
	if _, err := svc.SyncAppleTransaction(ctx, googleAttacker, "google"); !errors.Is(err, ErrTransactionAccount) {
		t.Fatalf("google hijack err=%v", err)
	}
	if status, err := svc.SyncAppleTransaction(ctx, googleOwner, "google"); err != nil || status.Plan.Code != PlanPlus {
		t.Fatalf("google restore status=%+v err=%v", status, err)
	}
}

func TestRecoveryTokenRebindRestoresDurableAppAccountBinding(t *testing.T) {
	ctx := context.Background()
	oldUser, restoredUser := uuid.New(), uuid.New()
	recoveryToken := "per-account-recovery-secret-at-least-32-characters"
	info := AppleTransactionInfo{
		TransactionID: "tx-recovery-shell", OriginalTransactionID: "original-recovery-shell",
		BundleID: subscriptionTestBundle, ProductID: planForCode(PlanPlus).ProductID,
		AppAccountToken: oldUser.String(), Environment: "Production",
		PurchasedAt: subscriptionTestNow, ExpiresAt: subscriptionTestNow.AddDate(0, 1, 0),
	}
	verifier := &fakeAppleVerifier{transactions: map[string]AppleTransactionInfo{"restore-shell": info}, notifications: map[string]AppleNotificationInfo{}}
	svc, db := newSubscriptionTestService(t, verifier)
	if err := db.Create(&models.User{ID: restoredUser, Email: "restored-shell@example.com"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.SubscriptionRestoreTombstone{
		OriginalTransactionID:       info.OriginalTransactionID,
		PreviousAppAccountTokenHash: restoreBindingHash("app-account-token", oldUser.String()),
		RecoveryTokenHash:           restoreBindingHash("recovery-token", recoveryToken),
		DeletedAt:                   subscriptionTestNow,
	}).Error; err != nil {
		t.Fatal(err)
	}
	// This is the purpose-limited transaction shell left after the original
	// account was deleted; its raw appAccountToken was deliberately erased.
	if err := db.Create(&models.AppleTransaction{
		TransactionID: info.TransactionID, OriginalTransactionID: info.OriginalTransactionID,
		ProductID: info.ProductID, Environment: info.Environment,
		PurchasedAt: info.PurchasedAt, ExpiresAt: info.ExpiresAt,
		JWSSHA256: strings.Repeat("a", 64),
	}).Error; err != nil {
		t.Fatal(err)
	}

	if _, err := svc.SyncAppleTransaction(ctx, restoredUser, "restore-shell", "wrong-recovery-token-at-least-32-characters"); !errors.Is(err, ErrTransactionAccount) {
		t.Fatalf("wrong recovery token err=%v", err)
	}
	if _, err := svc.SyncAppleTransaction(ctx, restoredUser, "restore-shell", recoveryToken); err != nil {
		t.Fatal(err)
	}
	var transaction models.AppleTransaction
	if err := db.First(&transaction, "transaction_id = ?", info.TransactionID).Error; err != nil ||
		transaction.UserID == nil || *transaction.UserID != restoredUser || transaction.AppAccountToken != oldUser.String() {
		t.Fatalf("restored transaction=%+v err=%v", transaction, err)
	}
	var tombstone models.SubscriptionRestoreTombstone
	if err := db.First(&tombstone, "original_transaction_id = ?", info.OriginalTransactionID).Error; err != nil ||
		tombstone.ReboundUserID == nil || *tombstone.ReboundUserID != restoredUser {
		t.Fatalf("rebound tombstone=%+v err=%v", tombstone, err)
	}
}
