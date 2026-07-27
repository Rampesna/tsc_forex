package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/tsc-forex/tsc_forex/internal/config"
	"github.com/tsc-forex/tsc_forex/internal/models"
)

func TestAdminTargetingPlanIsAllowListedAndBounded(t *testing.T) {
	t.Parallel()
	svc := NewAdminService(nil, nil, nil, config.AdminConfig{MaxRecipients: 250}, config.AIConfig{}, nil)
	plan := AdminTargetingPlan{
		Audience:      "asset_followers",
		Platforms:     []string{"IOS", "ios", "desktop"},
		PlanCodes:     []string{PlanPro, "root"},
		AssetCodes:    []string{" gram_gold ", "GRAM_GOLD", "x'); drop table users;--"},
		AssetSources:  []string{"watchlist", "database"},
		MaxRecipients: 999999,
	}
	if err := svc.validateTargetingPlan(&plan); err != nil {
		t.Fatal(err)
	}
	if len(plan.Platforms) != 1 || plan.Platforms[0] != "ios" {
		t.Fatalf("platforms=%v", plan.Platforms)
	}
	if len(plan.PlanCodes) != 1 || plan.PlanCodes[0] != PlanPro {
		t.Fatalf("plans=%v", plan.PlanCodes)
	}
	if len(plan.AssetCodes) != 1 || plan.AssetCodes[0] != "GRAM_GOLD" {
		t.Fatalf("asset codes=%v", plan.AssetCodes)
	}
	if len(plan.AssetSources) != 1 || plan.AssetSources[0] != "watchlist" || plan.MaxRecipients != 250 {
		t.Fatalf("sources=%v max=%d", plan.AssetSources, plan.MaxRecipients)
	}
}

func TestAdminCampaignPreviewEncodesEmptyCollectionsAsArrays(t *testing.T) {
	t.Parallel()
	preview := AdminCampaignPreview{}
	normalizeAdminCampaignPreview(&preview)

	raw, err := json.Marshal(preview)
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		Plan struct {
			Platforms    []string `json:"platforms"`
			PlanCodes    []string `json:"plan_codes"`
			AssetCodes   []string `json:"asset_codes"`
			AssetSources []string `json:"asset_sources"`
		} `json:"plan"`
		MatchedAssets []AdminCampaignAsset      `json:"matched_assets"`
		SampleUsers   []AdminCampaignSampleUser `json:"sample_users"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Plan.Platforms == nil || decoded.Plan.PlanCodes == nil ||
		decoded.Plan.AssetCodes == nil || decoded.Plan.AssetSources == nil ||
		decoded.MatchedAssets == nil || decoded.SampleUsers == nil {
		t.Fatalf("empty campaign collections must encode as JSON arrays: %s", raw)
	}
}

func TestRefreshCampaignPreviewReopensExistingDraft(t *testing.T) {
	t.Parallel()
	db, err := gorm.Open(sqlite.Open("file:admin-campaign-refresh-"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.DeviceToken{},
		&models.RefreshToken{},
		&models.NotificationCampaign{},
	); err != nil {
		t.Fatal(err)
	}

	managerID := uuid.New()
	campaignID := uuid.New()
	plan := AdminTargetingPlan{Audience: "all_users", MaxRecipients: 100}
	planJSON, _ := json.Marshal(plan)
	previewJSON, _ := json.Marshal(storedAdminCampaignPreview{Summary: "Mevcut taslak özeti"})
	campaign := models.NotificationCampaign{
		ID: campaignID, ManagerID: managerID, Name: "Devam edilecek taslak",
		Prompt: "Tüm kullanıcılara piyasa özeti hazırla", Status: "draft",
		TargetingPlan: models.JSONDocument(planJSON), Preview: models.JSONDocument(previewJSON),
		Title: "Piyasa özeti", Body: "Güncel piyasa özetini uygulamadan inceleyebilirsiniz.",
		ApprovalFingerprint: strings.Repeat("0", 64),
		CreatedAt:           time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	if err := db.Create(&campaign).Error; err != nil {
		t.Fatal(err)
	}

	svc := NewAdminService(db, nil, nil, config.AdminConfig{MaxRecipients: 1000}, config.AIConfig{}, nil)
	refreshed, err := svc.RefreshCampaignPreview(context.Background(), managerID, campaignID)
	if err != nil {
		t.Fatal(err)
	}
	if refreshed.CampaignID != campaignID || refreshed.Summary != "Mevcut taslak özeti" {
		t.Fatalf("unexpected preview: %+v", refreshed)
	}
	if refreshed.MatchedAssets == nil || refreshed.SampleUsers == nil {
		t.Fatalf("refreshed collections must not be nil: %+v", refreshed)
	}
	if refreshed.ApprovalFingerprint == "" || refreshed.ApprovalFingerprint == strings.Repeat("0", 64) {
		t.Fatal("approval fingerprint was not refreshed")
	}

	var stored models.NotificationCampaign
	if err := db.Take(&stored, "id = ?", campaignID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Status != "draft" || stored.ApprovalFingerprint != refreshed.ApprovalFingerprint {
		t.Fatalf("draft was not refreshed in place: %+v", stored)
	}
	var count int64
	if err := db.Model(&models.NotificationCampaign{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("refresh created another campaign: count=%d", count)
	}
}

func TestCampaignRecipientClaimRecoversStaleLease(t *testing.T) {
	t.Parallel()
	db, err := gorm.Open(sqlite.Open("file:admin-campaign-claim-"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.NotificationCampaignRecipient{}); err != nil {
		t.Fatal(err)
	}
	svc := NewAdminService(db, nil, nil, config.AdminConfig{}, config.AIConfig{}, nil)
	ctx := context.Background()
	campaignID := uuid.New()
	row := models.NotificationCampaignRecipient{
		ID: uuid.New(), CampaignID: campaignID, UserID: uuid.New(), Status: "pending",
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	if err := db.Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	claimed, err := svc.claimCampaignRecipients(ctx, campaignID, 10)
	if err != nil || len(claimed) != 1 || claimed[0].Attempts != 1 {
		t.Fatalf("first claim=%+v err=%v", claimed, err)
	}
	claimed, err = svc.claimCampaignRecipients(ctx, campaignID, 10)
	if err != nil || len(claimed) != 0 {
		t.Fatalf("active lease reclaimed=%+v err=%v", claimed, err)
	}
	if err := db.Model(&models.NotificationCampaignRecipient{}).Where("id = ?", row.ID).
		Updates(map[string]any{"status": "processing", "updated_at": time.Now().UTC().Add(-3 * time.Minute)}).Error; err != nil {
		t.Fatal(err)
	}
	claimed, err = svc.claimCampaignRecipients(ctx, campaignID, 10)
	if err != nil || len(claimed) != 1 || claimed[0].Attempts != 2 {
		t.Fatalf("stale recovery=%+v err=%v", claimed, err)
	}
}
