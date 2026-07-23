package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/tsc-forex/tsc_forex/internal/models"
)

func (s *SubscriptionService) SyncAppleTransaction(ctx context.Context, userID uuid.UUID, signedJWS string, recoveryToken ...string) (BillingStatus, error) {
	if !s.Enabled() || s.verifier == nil {
		return BillingStatus{}, ErrBillingDisabled
	}
	info, err := s.verifier.VerifyTransaction(strings.TrimSpace(signedJWS))
	if err != nil {
		return BillingStatus{}, fmt.Errorf("%w: %v", ErrInvalidAppleTransaction, err)
	}
	if err := s.validateAppleInfo(info); err != nil {
		return BillingStatus{}, fmt.Errorf("%w: validate transaction", err)
	}
	if info.AppAccountToken == "" {
		return BillingStatus{}, ErrInvalidAppleTransaction
	}
	if _, parseErr := uuid.Parse(info.AppAccountToken); parseErr != nil {
		return BillingStatus{}, ErrTransactionAccount
	}
	// A mismatched token may be the immutable appAccountToken of an account
	// deleted earlier. applyAppleTransaction permits it only through the
	// one-way Apple-identity tombstone recovery flow.
	token := ""
	if len(recoveryToken) > 0 {
		token = strings.TrimSpace(recoveryToken[0])
	}
	if err := s.applyAppleTransaction(ctx, &userID, info, signedJWS, token); err != nil {
		return BillingStatus{}, err
	}
	return s.Status(ctx, userID, "")
}

// ProcessAppleNotification applies App Store Server Notifications V2. The
// endpoint is idempotent because the verified transaction ID is the audit-row
// primary key and only a newer renewal can replace an entitlement snapshot.
func (s *SubscriptionService) ProcessAppleNotification(ctx context.Context, signedPayload string) error {
	if !s.Enabled() || s.verifier == nil {
		return ErrBillingDisabled
	}
	notification, err := s.verifier.VerifyNotification(strings.TrimSpace(signedPayload))
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidAppleTransaction, err)
	}
	if notification.BundleID != s.cfg.BundleID || !s.environmentAllowed(notification.Environment) {
		return fmt.Errorf("%w: notification bundle/environment rejected", ErrInvalidAppleTransaction)
	}
	// TEST notifications intentionally contain no transaction.
	if strings.TrimSpace(notification.SignedTransactionInfo) == "" {
		return nil
	}
	info, err := s.verifier.VerifyTransaction(notification.SignedTransactionInfo)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidAppleTransaction, err)
	}
	if info.BundleID != notification.BundleID || !strings.EqualFold(info.Environment, notification.Environment) {
		return fmt.Errorf("%w: notification transaction mismatch", ErrInvalidAppleTransaction)
	}
	if err := s.validateAppleInfo(info); err != nil {
		return fmt.Errorf("%w: validate notification transaction", err)
	}
	var userID *uuid.UUID
	if info.AppAccountToken != "" {
		if parsed, parseErr := uuid.Parse(info.AppAccountToken); parseErr == nil {
			var count int64
			if userErr := s.db.WithContext(ctx).Model(&models.User{}).Where("id = ?", parsed).Count(&count).Error; userErr == nil && count == 1 {
				userID = &parsed
			}
		}
	}
	if userID == nil {
		var transaction models.AppleTransaction
		findErr := s.db.WithContext(ctx).
			Where("original_transaction_id = ? AND user_id IS NOT NULL", info.OriginalTransactionID).
			Order("expires_at DESC").First(&transaction).Error
		if findErr == nil && transaction.UserID != nil {
			parsed := *transaction.UserID
			userID = &parsed
		} else {
			var entitlement models.SubscriptionEntitlement
			if entErr := s.db.WithContext(ctx).Where("original_transaction_id = ?", info.OriginalTransactionID).First(&entitlement).Error; entErr == nil {
				parsed := entitlement.UserID
				userID = &parsed
			}
		}
	}
	return s.applyAppleTransaction(ctx, userID, info, notification.SignedTransactionInfo, "")
}

func (s *SubscriptionService) validateAppleInfo(info AppleTransactionInfo) error {
	if info.BundleID != s.cfg.BundleID || info.TransactionID == "" || info.OriginalTransactionID == "" || info.ExpiresAt.IsZero() {
		return ErrInvalidAppleTransaction
	}
	if _, ok := planForProduct(info.ProductID); !ok || !s.environmentAllowed(info.Environment) {
		return ErrInvalidAppleTransaction
	}
	return nil
}

func (s *SubscriptionService) environmentAllowed(environment string) bool {
	switch strings.ToLower(strings.TrimSpace(environment)) {
	case "production":
		return true
	case "sandbox", "xcode":
		return s.cfg.AllowSandbox
	default:
		return false
	}
}

func (s *SubscriptionService) applyAppleTransaction(ctx context.Context, userID *uuid.UUID, info AppleTransactionInfo, rawJWS, recoveryToken string) error {
	plan, ok := planForProduct(info.ProductID)
	if !ok {
		return ErrInvalidAppleTransaction
	}
	digest := sha256.Sum256([]byte(rawJWS))
	now := s.now()
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if userID != nil {
			tokenUser, tokenErr := uuid.Parse(info.AppAccountToken)
			if tokenErr != nil {
				return ErrTransactionAccount
			}
			var tombstoneCount int64
			if err := tx.Model(&models.SubscriptionRestoreTombstone{}).
				Where("original_transaction_id = ?", info.OriginalTransactionID).Count(&tombstoneCount).Error; err != nil {
				return err
			}
			// A tombstone always wins over a freshly supplied appAccountToken.
			// Otherwise a recreated account could choose its own UUID during an
			// upgrade and bypass the recovery proof retained at deletion.
			if tokenUser != *userID || tombstoneCount > 0 {
				if err := s.authorizeSubscriptionRebind(ctx, tx, *userID, info, recoveryToken, now); err != nil {
					return err
				}
			}
		}
		// A subscription family can be restored only to the account that first
		// claimed it. This also prevents copying a JWS between app accounts.
		if userID != nil {
			var count int64
			if err := tx.Model(&models.AppleTransaction{}).
				Where("original_transaction_id = ? AND user_id IS NOT NULL AND user_id <> ?", info.OriginalTransactionID, *userID).
				Count(&count).Error; err != nil {
				return err
			}
			if count > 0 {
				return ErrTransactionAccount
			}
			var entitlementCount int64
			if err := tx.Model(&models.SubscriptionEntitlement{}).
				Where("original_transaction_id = ? AND user_id <> ?", info.OriginalTransactionID, *userID).
				Count(&entitlementCount).Error; err != nil {
				return err
			}
			if entitlementCount > 0 {
				return ErrTransactionAccount
			}
		}

		row := models.AppleTransaction{
			TransactionID: info.TransactionID, OriginalTransactionID: info.OriginalTransactionID,
			UserID: userID, ProductID: info.ProductID, AppAccountToken: info.AppAccountToken,
			Environment: info.Environment, PurchasedAt: info.PurchasedAt, ExpiresAt: info.ExpiresAt,
			RevokedAt: info.RevokedAt, JWSSHA256: hex.EncodeToString(digest[:]),
		}
		var existingTransaction models.AppleTransaction
		transactionErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("transaction_id = ?", info.TransactionID).First(&existingTransaction).Error
		if transactionErr == nil {
			if existingTransaction.OriginalTransactionID != info.OriginalTransactionID ||
				existingTransaction.ProductID != info.ProductID ||
				!strings.EqualFold(existingTransaction.Environment, info.Environment) ||
				!existingTransaction.PurchasedAt.Equal(info.PurchasedAt) || !existingTransaction.ExpiresAt.Equal(info.ExpiresAt) ||
				(existingTransaction.AppAccountToken != "" && existingTransaction.AppAccountToken != info.AppAccountToken) {
				return ErrInvalidAppleTransaction
			}
			if existingTransaction.UserID != nil && userID != nil && *existingTransaction.UserID != *userID {
				return ErrTransactionAccount
			}
			// Revocation is monotonic for an immutable transaction: an older
			// unrevoked JWS can never clear a refund/revoke already observed.
			if existingTransaction.RevokedAt != nil && (row.RevokedAt == nil || existingTransaction.RevokedAt.After(*row.RevokedAt)) {
				row.RevokedAt = existingTransaction.RevokedAt
			}
			updates := map[string]any{"updated_at": now}
			if row.RevokedAt != nil {
				updates["revoked_at"] = row.RevokedAt
			}
			if info.RevokedAt != nil {
				updates["jws_sha256"] = row.JWSSHA256
			}
			if existingTransaction.UserID == nil && userID != nil {
				updates["user_id"] = *userID
				// Account deletion deliberately blanks this value. Restore it only
				// from a freshly verified transaction so a later account deletion
				// can create another correctly bound tombstone.
				updates["app_account_token"] = info.AppAccountToken
			}
			if err := tx.Model(&models.AppleTransaction{}).Where("transaction_id = ?", info.TransactionID).Updates(updates).Error; err != nil {
				return err
			}
		} else if errors.Is(transactionErr, gorm.ErrRecordNotFound) {
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		} else {
			return transactionErr
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
		findErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", *userID).First(&current).Error
		if findErr == nil {
			sameTransaction := info.TransactionID == current.LatestTransactionID
			isNewerTransaction := info.TransactionID != current.LatestTransactionID &&
				entitlementExpiry.After(current.ExpiresAt) && !info.PurchasedAt.Before(current.PeriodStart)
			if !sameTransaction && !isNewerTransaction {
				return nil
			}
			if sameTransaction && current.RevokedAt != nil && row.RevokedAt == nil {
				row.RevokedAt = current.RevokedAt
				status = "revoked"
			}
		} else if !errors.Is(findErr, gorm.ErrRecordNotFound) {
			return findErr
		}
		entitlement := models.SubscriptionEntitlement{
			UserID: *userID, ProductID: info.ProductID, PlanCode: plan.Code,
			OriginalTransactionID: info.OriginalTransactionID, LatestTransactionID: info.TransactionID,
			Environment: info.Environment, Status: status, PeriodStart: info.PurchasedAt,
			ExpiresAt: entitlementExpiry, RevokedAt: row.RevokedAt,
		}
		return tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "user_id"}}, DoUpdates: clause.AssignmentColumns([]string{
				"product_id", "plan_code", "original_transaction_id", "latest_transaction_id", "environment",
				"status", "period_start", "expires_at", "revoked_at", "updated_at",
			}),
		}).Create(&entitlement).Error
	})
}

func (s *SubscriptionService) authorizeSubscriptionRebind(
	ctx context.Context,
	tx *gorm.DB,
	userID uuid.UUID,
	info AppleTransactionInfo,
	recoveryToken string,
	now time.Time,
) error {
	var tombstone models.SubscriptionRestoreTombstone
	if err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("original_transaction_id = ?", info.OriginalTransactionID).First(&tombstone).Error; err != nil {
		return ErrTransactionAccount
	}
	if tombstone.ReboundUserID != nil {
		if *tombstone.ReboundUserID != userID {
			return ErrTransactionAccount
		}
		// This account already proved the retained recovery binding. Later
		// renewals/upgrades may legitimately carry its new appAccountToken.
		return nil
	}
	if tombstone.PreviousAppAccountTokenHash != restoreBindingHash("app-account-token", info.AppAccountToken) {
		return ErrTransactionAccount
	}
	matched := false
	if tombstone.RecoveryTokenHash != "" && len(strings.TrimSpace(recoveryToken)) >= 32 && len(strings.TrimSpace(recoveryToken)) <= 256 &&
		restoreBindingHash("recovery-token", recoveryToken) == tombstone.RecoveryTokenHash {
		matched = true
	}
	if !matched && tombstone.IdentityProvider != "" && tombstone.ProviderSubjectHash != "" {
		var identities []models.AuthIdentity
		if err := tx.WithContext(ctx).Where("user_id = ? AND provider = ?", userID, tombstone.IdentityProvider).Find(&identities).Error; err != nil {
			return err
		}
		for _, identity := range identities {
			if restoreBindingHash(tombstone.IdentityProvider+"-subject", identity.Subject) == tombstone.ProviderSubjectHash {
				matched = true
				break
			}
		}
	}
	if !matched {
		return ErrTransactionAccount
	}
	updated := tx.Model(&models.SubscriptionRestoreTombstone{}).
		Where("original_transaction_id = ? AND (rebound_user_id IS NULL OR rebound_user_id = ?)", info.OriginalTransactionID, userID).
		Updates(map[string]any{"rebound_user_id": userID, "rebound_at": now})
	if updated.Error != nil {
		return updated.Error
	}
	if updated.RowsAffected != 1 {
		return ErrTransactionAccount
	}
	return nil
}

func restoreBindingHash(purpose, value string) string {
	digest := sha256.Sum256([]byte("tsc-subscription-restore-v1:" + purpose + ":" + strings.TrimSpace(value)))
	return hex.EncodeToString(digest[:])
}
