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

const (
	deviceRedemptionPrepared = "prepared"
	deviceRedemptionUpdating = "updating"
	deviceRedemptionVerified = "verified"
	deviceRedemptionDenied   = "denied"

	deviceRedemptionLeaseDuration = 30 * time.Second
)

func deviceCheckTokenHash(token string) string {
	digest := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return hex.EncodeToString(digest[:])
}

// authorizeFreeResearchProof is a small crash-safe saga. App Attest is first
// committed with a durable "prepared" journal. Before calling Apple's
// irreversible update_two_bits, the journal moves to "updating". If the
// process dies after Apple accepts the update, a retry sees "updating", queries
// bit 0=true and reconciles to "verified" instead of burning the trial.
func (s *SubscriptionService) authorizeFreeResearchProof(
	ctx context.Context,
	userID uuid.UUID,
	deviceID, requestKey, scope string,
	proof AppAttestProof,
) error {
	if !s.freeTrialReady() {
		if !s.appAttestReady() {
			return ErrDeviceAttestationUnavailable
		}
		return ErrDeviceCheckUnavailable
	}
	deviceHash, err := deviceDigest(deviceID)
	if err != nil {
		return err
	}
	var durable models.DeviceCheckRedemption
	if err := s.db.WithContext(ctx).Where("user_id = ? AND request_key = ?", userID, requestKey).First(&durable).Error; err == nil {
		if durable.DeviceHash != deviceHash || durable.Scope != scope {
			return ErrRequestConflict
		}
		if durable.Status == deviceRedemptionVerified {
			return nil
		}
	}
	if !validDeviceCheckToken(proof.DeviceCheckToken) {
		return ErrInvalidDeviceCheckToken
	}
	tokenHash := deviceCheckTokenHash(proof.DeviceCheckToken)
	ownerToken, ownerHash, err := randomCapability()
	if err != nil {
		return err
	}
	var redemption models.DeviceCheckRedemption
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		findErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ? AND request_key = ?", userID, requestKey).First(&redemption).Error
		if findErr == nil {
			if redemption.DeviceHash != deviceHash || redemption.Scope != scope {
				return ErrRequestConflict
			}
			if redemption.Status == deviceRedemptionDenied {
				return ErrFreeResearchUnavailable
			}
			return nil
		}
		if !errors.Is(findErr, gorm.ErrRecordNotFound) {
			return findErr
		}
		var tokenOwnerCount int64
		if err := tx.Model(&models.DeviceCheckRedemption{}).Where("token_hash = ?", tokenHash).Count(&tokenOwnerCount).Error; err != nil {
			return err
		}
		if tokenOwnerCount > 0 {
			return ErrFreeResearchUnavailable
		}
		if err := s.verifyAppAttestAssertionTx(ctx, tx, userID, deviceID, requestKey, scope, proof, s.now()); err != nil {
			return err
		}
		redemption = models.DeviceCheckRedemption{
			UserID: userID, RequestKey: requestKey, DeviceHash: deviceHash, Scope: scope,
			TokenHash: tokenHash, Status: deviceRedemptionPrepared,
			CreatedAt: s.now(), UpdatedAt: s.now(),
		}
		created := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&redemption)
		if created.Error != nil {
			return created.Error
		}
		if created.RowsAffected != 1 {
			return ErrFreeResearchUnavailable
		}
		return nil
	})
	if err != nil || redemption.Status == deviceRedemptionVerified {
		return err
	}
	claimed, err := s.deviceCheck.FreeTrialClaimed(ctx, proof.DeviceCheckToken)
	if err != nil {
		return err
	}
	if claimed {
		if redemption.Status == deviceRedemptionUpdating {
			return s.reconcileVerifiedDeviceRedemption(ctx, userID, requestKey)
		}
		_ = s.setDeviceRedemptionStatus(ctx, userID, requestKey, deviceRedemptionPrepared, deviceRedemptionDenied)
		return ErrFreeResearchUnavailable
	}
	acquired, err := s.claimDeviceRedemptionLease(ctx, userID, requestKey, ownerHash)
	if err != nil {
		return err
	}
	if !acquired {
		return ErrOperationInProgress
	}
	if err := s.deviceCheck.MarkFreeTrialClaimed(ctx, proof.DeviceCheckToken); err != nil {
		return err
	}
	return s.completeDeviceRedemptionLease(ctx, userID, requestKey, ownerToken)
}

func (s *SubscriptionService) claimDeviceRedemptionLease(
	ctx context.Context,
	userID uuid.UUID,
	requestKey, ownerHash string,
) (bool, error) {
	now := s.now()
	leaseExpiresAt := now.Add(deviceRedemptionLeaseDuration)
	result := s.db.WithContext(ctx).Model(&models.DeviceCheckRedemption{}).
		Where("user_id = ? AND request_key = ? AND (status = ? OR (status = ? AND (lease_expires_at IS NULL OR lease_expires_at <= ?)))",
			userID, requestKey, deviceRedemptionPrepared, deviceRedemptionUpdating, now).
		Updates(map[string]any{
			"status": deviceRedemptionUpdating, "owner_token_hash": ownerHash,
			"lease_expires_at": leaseExpiresAt, "attempt_count": gorm.Expr("attempt_count + 1"), "updated_at": now,
		})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected == 1, nil
}

func (s *SubscriptionService) completeDeviceRedemptionLease(
	ctx context.Context,
	userID uuid.UUID,
	requestKey, ownerToken string,
) error {
	result := s.db.WithContext(ctx).Model(&models.DeviceCheckRedemption{}).
		Where("user_id = ? AND request_key = ? AND status = ? AND owner_token_hash = ?",
			userID, requestKey, deviceRedemptionUpdating, capabilityHash(ownerToken)).
		Updates(map[string]any{
			"status": deviceRedemptionVerified, "owner_token_hash": "", "lease_expires_at": nil, "updated_at": s.now(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 1 {
		return nil
	}
	var row models.DeviceCheckRedemption
	if err := s.db.WithContext(ctx).Where("user_id = ? AND request_key = ?", userID, requestKey).First(&row).Error; err == nil && row.Status == deviceRedemptionVerified {
		return nil
	}
	return ErrOperationInProgress
}

func (s *SubscriptionService) reconcileVerifiedDeviceRedemption(ctx context.Context, userID uuid.UUID, requestKey string) error {
	result := s.db.WithContext(ctx).Model(&models.DeviceCheckRedemption{}).
		Where("user_id = ? AND request_key = ? AND status = ?", userID, requestKey, deviceRedemptionUpdating).
		Updates(map[string]any{
			"status": deviceRedemptionVerified, "owner_token_hash": "", "lease_expires_at": nil, "updated_at": s.now(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 1 {
		return nil
	}
	var row models.DeviceCheckRedemption
	if err := s.db.WithContext(ctx).Where("user_id = ? AND request_key = ?", userID, requestKey).First(&row).Error; err == nil && row.Status == deviceRedemptionVerified {
		return nil
	}
	return ErrOperationInProgress
}

func (s *SubscriptionService) setDeviceRedemptionStatus(ctx context.Context, userID uuid.UUID, requestKey, from, to string) error {
	result := s.db.WithContext(ctx).Model(&models.DeviceCheckRedemption{}).
		Where("user_id = ? AND request_key = ? AND status = ?", userID, requestKey, from).
		Updates(map[string]any{"status": to, "owner_token_hash": "", "lease_expires_at": nil, "updated_at": s.now()})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 1 {
		return nil
	}
	var row models.DeviceCheckRedemption
	if err := s.db.WithContext(ctx).Where("user_id = ? AND request_key = ?", userID, requestKey).First(&row).Error; err == nil && row.Status == to {
		return nil
	}
	return ErrOperationInProgress
}

func (s *SubscriptionService) requireFreeAuthorizationTx(ctx context.Context, tx *gorm.DB, userID uuid.UUID, deviceID, requestKey, scope string) error {
	deviceHash, err := deviceDigest(deviceID)
	if err != nil {
		return err
	}
	var row models.DeviceCheckRedemption
	if err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ? AND request_key = ?", userID, requestKey).First(&row).Error; err != nil {
		return ErrDeviceAttestationNeeded
	}
	if row.DeviceHash != deviceHash || row.Scope != scope || row.Status != deviceRedemptionVerified {
		return ErrInvalidDeviceAttestation
	}
	return nil
}
