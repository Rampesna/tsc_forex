package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/tsc-forex/tsc_forex/internal/models"
)

const googleEntitlementPrefix = "gplay:"

func googleOriginalID(purchaseToken string) string {
	digest := sha256.Sum256([]byte(strings.TrimSpace(purchaseToken)))
	return googleEntitlementPrefix + hex.EncodeToString(digest[:16])
}

func (s *SubscriptionService) GoogleBillingReady() bool {
	return s != nil && s.googleVerifier != nil && s.googleVerifier.Ready()
}

// SyncGooglePurchase verifies a Play Billing purchase token with Google and
// upserts the account entitlement. Apple sync paths are untouched.
func (s *SubscriptionService) SyncGooglePurchase(ctx context.Context, userID uuid.UUID, packageName, productID, purchaseToken string) (BillingStatus, error) {
	if !s.Enabled() {
		return BillingStatus{}, ErrBillingDisabled
	}
	if !s.GoogleBillingReady() {
		return BillingStatus{}, ErrGoogleBillingUnavailable
	}
	info, err := s.googleVerifier.VerifySubscription(ctx, packageName, productID, purchaseToken)
	if err != nil {
		return BillingStatus{}, err
	}
	if err := s.validateGoogleInfo(info); err != nil {
		return BillingStatus{}, err
	}
	// Prefer binding via obfuscated account id when Play returns one.
	if info.ObfuscatedAccountID != "" {
		tokenUser, parseErr := uuid.Parse(info.ObfuscatedAccountID)
		if parseErr != nil || tokenUser != userID {
			return BillingStatus{}, ErrTransactionAccount
		}
	}
	if err := s.applyGooglePurchase(ctx, &userID, info); err != nil {
		return BillingStatus{}, err
	}
	return s.Status(ctx, userID, "")
}

func (s *SubscriptionService) validateGoogleInfo(info GooglePlayPurchaseInfo) error {
	pkg := strings.TrimSpace(s.cfg.GooglePackageName)
	if pkg == "" {
		pkg = "com.talhacan.tscforex"
	}
	if info.PackageName != pkg || info.PurchaseToken == "" || info.ProductID == "" || info.ExpiresAt.IsZero() {
		return ErrInvalidGooglePurchase
	}
	if _, ok := planForProduct(info.ProductID); !ok {
		return ErrInvalidGooglePurchase
	}
	switch strings.ToLower(strings.TrimSpace(info.Environment)) {
	case "production":
		return nil
	case "sandbox":
		if s.cfg.AllowSandbox {
			return nil
		}
		return ErrInvalidGooglePurchase
	default:
		return ErrInvalidGooglePurchase
	}
}

func (s *SubscriptionService) applyGooglePurchase(ctx context.Context, userID *uuid.UUID, info GooglePlayPurchaseInfo) error {
	plan, ok := planForProduct(info.ProductID)
	if !ok {
		return ErrInvalidGooglePurchase
	}
	digest := sha256.Sum256(info.RawJSON)
	now := s.now()
	originalID := googleOriginalID(info.PurchaseToken)
	if linked := strings.TrimSpace(info.LinkedPurchaseToken); linked != "" {
		// Keep the subscription family stable across upgrades when Google
		// provides a linked prior token.
		originalID = googleOriginalID(linked)
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if userID != nil {
			var count int64
			if err := tx.Model(&models.GooglePlayTransaction{}).
				Where("purchase_token = ? AND user_id IS NOT NULL AND user_id <> ?", info.PurchaseToken, *userID).
				Count(&count).Error; err != nil {
				return err
			}
			if count > 0 {
				return ErrTransactionAccount
			}
			var entitlementCount int64
			if err := tx.Model(&models.SubscriptionEntitlement{}).
				Where("original_transaction_id = ? AND user_id <> ?", originalID, *userID).
				Count(&entitlementCount).Error; err != nil {
				return err
			}
			if entitlementCount > 0 {
				return ErrTransactionAccount
			}
		}

		row := models.GooglePlayTransaction{
			PurchaseToken: info.PurchaseToken, OrderID: info.OrderID,
			LinkedPurchaseToken: info.LinkedPurchaseToken, UserID: userID,
			PackageName: info.PackageName, ProductID: info.ProductID,
			ObfuscatedAccountID: info.ObfuscatedAccountID, Environment: info.Environment,
			PurchasedAt: info.PurchasedAt, ExpiresAt: info.ExpiresAt, RevokedAt: info.RevokedAt,
			Acknowledged: info.Acknowledged, RawSHA256: hex.EncodeToString(digest[:]),
		}
		var existing models.GooglePlayTransaction
		findErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("purchase_token = ?", info.PurchaseToken).First(&existing).Error
		if findErr == nil {
			if existing.UserID != nil && userID != nil && *existing.UserID != *userID {
				return ErrTransactionAccount
			}
			updates := map[string]any{
				"order_id": info.OrderID, "product_id": info.ProductID,
				"environment": info.Environment, "purchased_at": info.PurchasedAt,
				"expires_at": info.ExpiresAt, "acknowledged": info.Acknowledged,
				"raw_sha256": row.RawSHA256, "updated_at": now,
			}
			if info.RevokedAt != nil {
				updates["revoked_at"] = info.RevokedAt
			}
			if existing.UserID == nil && userID != nil {
				updates["user_id"] = *userID
				updates["obfuscated_account_id"] = info.ObfuscatedAccountID
			}
			if err := tx.Model(&models.GooglePlayTransaction{}).
				Where("purchase_token = ?", info.PurchaseToken).Updates(updates).Error; err != nil {
				return err
			}
			if existing.RevokedAt != nil && row.RevokedAt == nil {
				row.RevokedAt = existing.RevokedAt
			}
		} else if errors.Is(findErr, gorm.ErrRecordNotFound) {
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		} else {
			return findErr
		}
		if userID == nil {
			return nil
		}

		entitlementExpiry := info.ExpiresAt
		if !strings.EqualFold(info.Environment, "Production") {
			maxLifetime := s.cfg.SandboxMaxLifetime
			if maxLifetime <= 0 || maxLifetime > 7*24*time.Hour {
				maxLifetime = 24 * time.Hour
			}
			capExpiry := info.PurchasedAt.Add(maxLifetime)
			if capExpiry.Before(entitlementExpiry) {
				entitlementExpiry = capExpiry
			}
		}
		status := "active"
		if row.RevokedAt != nil {
			status = "revoked"
		} else if !entitlementExpiry.After(now) {
			status = "expired"
		}

		var current models.SubscriptionEntitlement
		curErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", *userID).First(&current).Error
		if curErr == nil {
			sameFamily := current.OriginalTransactionID == originalID
			isNewer := entitlementExpiry.After(current.ExpiresAt) && !info.PurchasedAt.Before(current.PeriodStart)
			if !sameFamily && !isNewer {
				// Preserve a longer-lived active Apple entitlement when this
				// Google purchase would shorten the user's paid window.
				if (current.Provider == "" || strings.EqualFold(current.Provider, "apple")) &&
					current.Status == "active" && current.ExpiresAt.After(entitlementExpiry) {
					return nil
				}
			}
			if sameFamily && current.RevokedAt != nil && row.RevokedAt == nil {
				row.RevokedAt = current.RevokedAt
				status = "revoked"
			}
		} else if !errors.Is(curErr, gorm.ErrRecordNotFound) {
			return curErr
		}

		latestID := info.OrderID
		if latestID == "" {
			latestID = originalID
		}
		entitlement := models.SubscriptionEntitlement{
			UserID: *userID, ProductID: info.ProductID, PlanCode: plan.Code,
			Provider: "google", OriginalTransactionID: originalID, LatestTransactionID: latestID,
			Environment: info.Environment, Status: status, PeriodStart: info.PurchasedAt,
			ExpiresAt: entitlementExpiry, RevokedAt: row.RevokedAt,
		}
		return tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "user_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"product_id", "plan_code", "provider", "original_transaction_id", "latest_transaction_id",
				"environment", "status", "period_start", "expires_at", "revoked_at", "updated_at",
			}),
		}).Create(&entitlement).Error
	})
}
