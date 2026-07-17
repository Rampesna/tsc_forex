package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/tsc-forex/tsc_forex/internal/models"
)

const featureArtifactLeaseDuration = 2 * time.Minute

type FeatureArtifactReservation struct {
	Created     bool
	OwnerToken  string
	Payload     []byte
	ContentType string
	Filename    string
}

// BeginFeatureArtifact atomically reserves quota and one durable output slot.
// Completed retries replay bytes; an in-flight duplicate cannot recompute for
// free against mutable portfolio data.
func (s *SubscriptionService) BeginFeatureArtifact(
	ctx context.Context,
	userID uuid.UUID,
	feature, requestKey, scope string,
) (FeatureArtifactReservation, error) {
	requestKey, scope, err := normalizeRequest(requestKey, scope)
	if err != nil {
		return FeatureArtifactReservation{}, err
	}
	ownerToken, ownerHash, err := randomCapability()
	if err != nil {
		return FeatureArtifactReservation{}, err
	}
	now := s.now()
	var result FeatureArtifactReservation
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing models.FeatureArtifact
		findErr := tx.Where("user_id = ? AND feature = ? AND request_key = ?", userID, feature, requestKey).First(&existing).Error
		if findErr == nil {
			if existing.Scope != scope {
				return ErrRequestConflict
			}
			if existing.Status == ResearchJobCompleted {
				result = FeatureArtifactReservation{
					Payload: append([]byte(nil), existing.Payload...), ContentType: existing.ContentType, Filename: existing.Filename,
				}
				return nil
			}
			if existing.Status != ResearchJobReserved {
				return ErrOperationInProgress
			}
			if existing.LeaseExpiresAt != nil && existing.LeaseExpiresAt.After(now) {
				return ErrOperationInProgress
			}
			leaseExpiresAt := now.Add(featureArtifactLeaseDuration)
			updated := tx.Model(&models.FeatureArtifact{}).
				Where("id = ? AND status = ? AND (lease_expires_at IS NULL OR lease_expires_at <= ?)", existing.ID, ResearchJobReserved, now).
				Updates(map[string]any{
					"owner_token_hash": ownerHash, "lease_expires_at": leaseExpiresAt,
					"attempt_count": gorm.Expr("attempt_count + 1"), "updated_at": now,
				})
			if updated.Error != nil {
				return updated.Error
			}
			if updated.RowsAffected != 1 {
				return ErrOperationInProgress
			}
			result = FeatureArtifactReservation{Created: true, OwnerToken: ownerToken}
			return nil
		}
		if !errors.Is(findErr, gorm.ErrRecordNotFound) {
			return findErr
		}
		leaseExpiresAt := now.Add(featureArtifactLeaseDuration)
		row := models.FeatureArtifact{
			UserID: userID, Feature: feature, RequestKey: requestKey, Scope: scope,
			Status: ResearchJobReserved, OwnerTokenHash: ownerHash, LeaseExpiresAt: &leaseExpiresAt,
			AttemptCount: 1, CreatedAt: now, UpdatedAt: now,
		}
		created := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&row)
		if created.Error != nil {
			return created.Error
		}
		if created.RowsAffected == 0 {
			return ErrOperationInProgress
		}
		ent, active := s.lockedEntitlement(ctx, tx, userID, now)
		if !active {
			return &QuotaError{Feature: feature, Code: "subscription_required"}
		}
		if _, err := s.consumePaid(ctx, tx, userID, ent, feature, requestKey, scope, now); err != nil {
			return err
		}
		result = FeatureArtifactReservation{Created: true, OwnerToken: ownerToken}
		return nil
	})
	return result, err
}

func (s *SubscriptionService) CompleteFeatureArtifact(
	ctx context.Context,
	userID uuid.UUID,
	feature, requestKey, ownerToken string,
	payload []byte,
	contentType, filename string,
) error {
	if len(payload) == 0 || len(payload) > 8<<20 {
		return ErrRequestConflict
	}
	result := s.db.WithContext(ctx).Model(&models.FeatureArtifact{}).
		Where("user_id = ? AND feature = ? AND request_key = ? AND status = ? AND owner_token_hash = ? AND lease_expires_at > ?",
			userID, feature, requestKey, ResearchJobReserved, capabilityHash(ownerToken), s.now()).
		Updates(map[string]any{
			"status": ResearchJobCompleted, "payload": payload, "content_type": contentType,
			"filename": filename, "owner_token_hash": "", "lease_expires_at": nil, "updated_at": s.now(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrRequestConflict
	}
	return nil
}

func (s *SubscriptionService) ReleaseFeatureArtifact(
	ctx context.Context,
	userID uuid.UUID,
	feature, requestKey, ownerToken string,
) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		deleted := tx.Where("user_id = ? AND feature = ? AND request_key = ? AND status = ? AND owner_token_hash = ?",
			userID, feature, requestKey, ResearchJobReserved, capabilityHash(ownerToken)).Delete(&models.FeatureArtifact{})
		if deleted.Error != nil {
			return deleted.Error
		}
		if deleted.RowsAffected != 1 {
			return ErrRequestConflict
		}
		return tx.Where("user_id = ? AND feature = ? AND request_key = ?", userID, feature, requestKey).
			Delete(&models.FeatureUsage{}).Error
	})
}
