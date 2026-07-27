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
