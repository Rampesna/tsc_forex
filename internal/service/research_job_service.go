package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/tsc-forex/tsc_forex/internal/models"
)

const (
	ResearchJobReserved  = "reserved"
	ResearchJobPending   = "pending"
	ResearchJobCompleted = "completed"
	ResearchJobFailed    = "failed"

	researchJobLeaseDuration = 2 * time.Minute
)

type AIResearchJob struct {
	RequestKey         string
	Code               string
	Status             string
	ProviderResponseID string
	ResultPayload      json.RawMessage
	Created            bool
	OwnerToken         string
	Consumption        Consumption
}

func randomCapability() (string, string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", "", err
	}
	token := hex.EncodeToString(raw)
	digest := sha256.Sum256([]byte(token))
	return token, hex.EncodeToString(digest[:]), nil
}

func capabilityHash(token string) string {
	digest := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return hex.EncodeToString(digest[:])
}

// BeginAIResearchJob atomically creates the durable job and quota reservation.
// A concurrent/replayed request receives the already-created job and never
// receives its creator capability, so it cannot start or refund another job.
func (s *SubscriptionService) BeginAIResearchJob(
	ctx context.Context,
	userID uuid.UUID,
	deviceID, requestKey, code string,
	proof AppAttestProof,
) (AIResearchJob, error) {
	requestKey, code, err := normalizeRequest(requestKey, strings.ToUpper(code))
	if err != nil {
		return AIResearchJob{}, err
	}
	if s == nil || s.db == nil {
		return AIResearchJob{}, ErrBillingDisabled
	}
	now := s.now()
	// Fast replay path: an existing durable job never consumes a second App
	// Attest assertion or DeviceCheck operation. A creator that died before it
	// attached a provider response can be reclaimed after its durable lease.
	if existing, existingErr := s.AIResearchJob(ctx, userID, requestKey, code); existingErr == nil {
		if existing.Status == ResearchJobReserved {
			return s.LeaseAIResearchJob(ctx, userID, requestKey, code)
		}
		return existing, nil
	} else if !errors.Is(existingErr, gorm.ErrRecordNotFound) {
		return AIResearchJob{}, existingErr
	}
	ownerToken, ownerHash, err := randomCapability()
	if err != nil {
		return AIResearchJob{}, err
	}
	if _, active := s.activeEntitlement(ctx, s.db, userID, now); !active {
		var authorizeErr error
		for attempt := 0; attempt < 25; attempt++ {
			authorizeErr = s.authorizeFreeResearchProof(ctx, userID, deviceID, requestKey, code, proof)
			if !errors.Is(authorizeErr, ErrOperationInProgress) {
				break
			}
			select {
			case <-ctx.Done():
				return AIResearchJob{}, ctx.Err()
			case <-time.After(10 * time.Millisecond):
			}
		}
		if authorizeErr != nil {
			return AIResearchJob{}, authorizeErr
		}
	}
	var result AIResearchJob
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing models.MarketResearchJob
		findErr := tx.Where("user_id = ? AND request_key = ?", userID, requestKey).First(&existing).Error
		if findErr == nil {
			if existing.Code != code {
				return ErrRequestConflict
			}
			result, err = claimAIResearchLeaseTx(tx, existing, ownerToken, ownerHash, now)
			return err
		}
		if !errors.Is(findErr, gorm.ErrRecordNotFound) {
			return findErr
		}
		leaseExpiresAt := now.Add(researchJobLeaseDuration)
		row := models.MarketResearchJob{
			UserID: userID, RequestKey: requestKey, Code: code, Status: ResearchJobReserved,
			OwnerTokenHash: ownerHash, LeaseExpiresAt: &leaseExpiresAt, AttemptCount: 1,
			CreatedAt: now, UpdatedAt: now,
		}
		created := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&row)
		if created.Error != nil {
			return created.Error
		}
		if created.RowsAffected == 0 {
			if err := tx.Where("user_id = ? AND request_key = ?", userID, requestKey).First(&existing).Error; err != nil {
				return err
			}
			if existing.Code != code {
				return ErrRequestConflict
			}
			result, err = claimAIResearchLeaseTx(tx, existing, ownerToken, ownerHash, now)
			return err
		}
		consumption, err := s.consumeAIResearchTx(ctx, tx, userID, deviceID, requestKey, code, now, true)
		if err != nil {
			return err
		}
		result = researchJobView(row)
		result.Created = true
		result.OwnerToken = ownerToken
		result.Consumption = consumption
		return nil
	})
	return result, err
}

// LeaseAIResearchJob reclaims only an expired, never-attached reservation.
// Quota and DeviceCheck state were committed with the original row, so a
// reclaim rotates only the short-lived capability and never spends again.
func (s *SubscriptionService) LeaseAIResearchJob(
	ctx context.Context,
	userID uuid.UUID,
	requestKey, code string,
) (AIResearchJob, error) {
	requestKey, code, err := normalizeRequest(requestKey, strings.ToUpper(code))
	if err != nil {
		return AIResearchJob{}, err
	}
	if s == nil || s.db == nil {
		return AIResearchJob{}, ErrBillingDisabled
	}
	ownerToken, ownerHash, err := randomCapability()
	if err != nil {
		return AIResearchJob{}, err
	}
	now := s.now()
	var result AIResearchJob
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row models.MarketResearchJob
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ? AND request_key = ?", userID, requestKey).First(&row).Error; err != nil {
			return err
		}
		if row.Code != code {
			return ErrRequestConflict
		}
		result, err = claimAIResearchLeaseTx(tx, row, ownerToken, ownerHash, now)
		return err
	})
	return result, err
}

func claimAIResearchLeaseTx(
	tx *gorm.DB,
	row models.MarketResearchJob,
	ownerToken, ownerHash string,
	now time.Time,
) (AIResearchJob, error) {
	if row.Status != ResearchJobReserved {
		return researchJobView(row), nil
	}
	if row.LeaseExpiresAt != nil && row.LeaseExpiresAt.After(now) {
		return researchJobView(row), nil
	}
	leaseExpiresAt := now.Add(researchJobLeaseDuration)
	updated := tx.Model(&models.MarketResearchJob{}).
		Where("id = ? AND status = ? AND (lease_expires_at IS NULL OR lease_expires_at <= ?)", row.ID, ResearchJobReserved, now).
		Updates(map[string]any{
			"owner_token_hash": ownerHash, "lease_expires_at": leaseExpiresAt,
			"attempt_count": gorm.Expr("attempt_count + 1"), "updated_at": now,
		})
	if updated.Error != nil {
		return AIResearchJob{}, updated.Error
	}
	if updated.RowsAffected != 1 {
		if err := tx.Where("id = ?", row.ID).First(&row).Error; err != nil {
			return AIResearchJob{}, err
		}
		return researchJobView(row), nil
	}
	row.OwnerTokenHash = ownerHash
	row.LeaseExpiresAt = &leaseExpiresAt
	row.AttemptCount++
	row.UpdatedAt = now
	view := researchJobView(row)
	view.Created = true
	view.OwnerToken = ownerToken
	return view, nil
}

func researchJobView(row models.MarketResearchJob) AIResearchJob {
	view := AIResearchJob{
		RequestKey: row.RequestKey, Code: row.Code, Status: row.Status,
		ProviderResponseID: row.ProviderResponseID,
	}
	if json.Valid([]byte(row.ResultPayload)) {
		view.ResultPayload = json.RawMessage(row.ResultPayload)
	}
	return view
}

func (s *SubscriptionService) AIResearchJob(ctx context.Context, userID uuid.UUID, requestKey, code string) (AIResearchJob, error) {
	requestKey, code, err := normalizeRequest(requestKey, strings.ToUpper(code))
	if err != nil {
		return AIResearchJob{}, err
	}
	var row models.MarketResearchJob
	if err := s.db.WithContext(ctx).Where("user_id = ? AND request_key = ?", userID, requestKey).First(&row).Error; err != nil {
		return AIResearchJob{}, err
	}
	if row.Code != code {
		return AIResearchJob{}, ErrRequestConflict
	}
	return researchJobView(row), nil
}

func (s *SubscriptionService) AttachAIResearchProvider(
	ctx context.Context,
	userID uuid.UUID,
	requestKey, ownerToken, providerResponseID string,
) error {
	providerResponseID = strings.TrimSpace(providerResponseID)
	if providerResponseID == "" || len(providerResponseID) > 160 {
		return ErrRequestConflict
	}
	result := s.db.WithContext(ctx).Model(&models.MarketResearchJob{}).
		Where("user_id = ? AND request_key = ? AND status = ? AND owner_token_hash = ? AND lease_expires_at > ?",
			userID, requestKey, ResearchJobReserved, capabilityHash(ownerToken), s.now()).
		Updates(map[string]any{
			"status": ResearchJobPending, "provider_response_id": providerResponseID,
			"owner_token_hash": "", "lease_expires_at": nil, "updated_at": s.now(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrRequestConflict
	}
	return nil
}

func (s *SubscriptionService) CompleteAIResearchJob(
	ctx context.Context,
	userID uuid.UUID,
	requestKey, ownerToken, providerResponseID string,
	payload []byte,
) error {
	if !json.Valid(payload) {
		return ErrRequestConflict
	}
	now := s.now()
	query := s.db.WithContext(ctx).Model(&models.MarketResearchJob{}).
		Where("user_id = ? AND request_key = ?", userID, requestKey)
	if strings.TrimSpace(providerResponseID) != "" {
		query = query.Where("status = ? AND provider_response_id = ?", ResearchJobPending, strings.TrimSpace(providerResponseID))
	} else {
		query = query.Where("status = ? AND owner_token_hash = ? AND lease_expires_at > ?", ResearchJobReserved, capabilityHash(ownerToken), now)
	}
	result := query.Updates(map[string]any{
		"status": ResearchJobCompleted, "result_payload": string(payload),
		"owner_token_hash": "", "lease_expires_at": nil,
		"completed_at": now, "updated_at": now,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 1 {
		return nil
	}
	// A concurrent poll may already have persisted the same immutable result.
	var row models.MarketResearchJob
	if err := s.db.WithContext(ctx).Where("user_id = ? AND request_key = ?", userID, requestKey).First(&row).Error; err == nil &&
		row.Status == ResearchJobCompleted && row.ResultPayload == string(payload) {
		return nil
	}
	return ErrRequestConflict
}

// ReleaseAIResearchJob refunds only a never-started reservation and only when
// the caller presents the capability returned to the creator. Pending provider
// work is never refunded because provider cost may already have been incurred.
func (s *SubscriptionService) ReleaseAIResearchJob(ctx context.Context, userID uuid.UUID, requestKey, ownerToken, failureCode string) error {
	if s == nil || s.db == nil || strings.TrimSpace(ownerToken) == "" {
		return ErrRequestConflict
	}
	_ = failureCode // reserved for structured audit logging at the caller
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		deleted := tx.
			Where("user_id = ? AND request_key = ? AND status = ? AND owner_token_hash = ?", userID, requestKey, ResearchJobReserved, capabilityHash(ownerToken)).
			Delete(&models.MarketResearchJob{})
		if deleted.Error != nil {
			return deleted.Error
		}
		if deleted.RowsAffected != 1 {
			return ErrRequestConflict
		}
		if err := tx.Where("user_id = ? AND feature = ? AND request_key = ?", userID, FeatureAIResearch, requestKey).
			Delete(&models.FeatureUsage{}).Error; err != nil {
			return err
		}
		return tx.Where("user_id = ? AND request_key = ?", userID, requestKey).
			Delete(&models.FreeResearchClaim{}).Error
	})
}

// MarkAIResearchReservationFailed closes a provider-dispatched reservation
// without refunding quota. It is used when the provider responded but its
// result was unusable; refunding here would let one allowance create repeated
// upstream spend.
func (s *SubscriptionService) MarkAIResearchReservationFailed(
	ctx context.Context,
	userID uuid.UUID,
	requestKey, ownerToken, failureCode string,
) error {
	result := s.db.WithContext(ctx).Model(&models.MarketResearchJob{}).
		Where("user_id = ? AND request_key = ? AND status = ? AND owner_token_hash = ?",
			userID, requestKey, ResearchJobReserved, capabilityHash(ownerToken)).
		Updates(map[string]any{
			"status": ResearchJobFailed, "failure_code": failureCode,
			"owner_token_hash": "", "lease_expires_at": nil, "updated_at": s.now(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrRequestConflict
	}
	return nil
}

func (s *SubscriptionService) MarkAIResearchProviderFailed(ctx context.Context, userID uuid.UUID, requestKey, providerResponseID, failureCode string) error {
	// This terminal marker intentionally does not refund: once the upstream
	// provider has accepted a job, cost was incurred and a replay must not mint a
	// second allowance.
	result := s.db.WithContext(ctx).Model(&models.MarketResearchJob{}).
		Where("user_id = ? AND request_key = ? AND status = ? AND provider_response_id = ?", userID, requestKey, ResearchJobPending, providerResponseID).
		Updates(map[string]any{
			"status": ResearchJobFailed, "failure_code": failureCode,
			"owner_token_hash": "", "lease_expires_at": nil, "updated_at": s.now(),
		})
	return result.Error
}
