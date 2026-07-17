package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
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

const appAttestPurposeFreeAI = "free_ai"

type AppAttestChallengeView struct {
	ChallengeID         uuid.UUID `json:"challenge_id"`
	Challenge           string    `json:"challenge"`
	ClientData          string    `json:"client_data"`
	ExpiresAt           time.Time `json:"expires_at"`
	NeedsKeyAttestation bool      `json:"needs_key_attestation"`
}

type AppAttestProof struct {
	KeyID            string
	ChallengeID      uuid.UUID
	Assertion        []byte
	DeviceCheckToken string
}

type appAttestClientData struct {
	Challenge string `json:"challenge"`
	UserID    string `json:"user_id"`
	DeviceID  string `json:"device_id"`
	RequestID string `json:"request_id"`
	Code      string `json:"code"`
	Purpose   string `json:"purpose"`
}

func (s *SubscriptionService) appAttestReady() bool {
	return s != nil && s.db != nil && s.cfg.AppAttestEnabled && s.appAttest != nil && s.appAttest.Ready()
}

func (s *SubscriptionService) freeTrialReady() bool {
	return s.appAttestReady() && s.deviceCheck != nil && s.deviceCheck.Ready()
}

// IssueAppAttestChallenge returns one random, short-lived challenge already
// bound to the authenticated account, local installation and exact AI request.
// ClientData is base64url-encoded canonical JSON; iOS hashes the decoded bytes
// before calling attestKey or generateAssertion.
func (s *SubscriptionService) IssueAppAttestChallenge(
	ctx context.Context,
	userID uuid.UUID,
	deviceID, keyID, requestKey, scope string,
) (AppAttestChallengeView, error) {
	if !s.appAttestReady() {
		return AppAttestChallengeView{}, ErrDeviceAttestationUnavailable
	}
	requestKey, scope, err := normalizeRequest(requestKey, strings.ToUpper(scope))
	if err != nil {
		return AppAttestChallengeView{}, err
	}
	deviceID = strings.TrimSpace(deviceID)
	deviceHash, err := deviceDigest(deviceID)
	if err != nil {
		return AppAttestChallengeView{}, err
	}
	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		return AppAttestChallengeView{}, err
	}
	nonceText := base64.RawURLEncoding.EncodeToString(nonce)
	clientData, err := json.Marshal(appAttestClientData{
		Challenge: nonceText, UserID: userID.String(), DeviceID: deviceID,
		RequestID: requestKey, Code: scope, Purpose: appAttestPurposeFreeAI,
	})
	if err != nil {
		return AppAttestChallengeView{}, err
	}
	digest := sha256.Sum256(clientData)
	ttl := s.cfg.AppAttestChallengeTTL
	if ttl <= 0 || ttl > 15*time.Minute {
		ttl = 5 * time.Minute
	}
	now := s.now()
	challenge := models.AppAttestChallenge{
		UserID: userID, DeviceHash: deviceHash, RequestKey: requestKey, Scope: scope,
		Purpose: appAttestPurposeFreeAI, NonceHash: hex.EncodeToString(digest[:]),
		ClientData: clientData, ExpiresAt: now.Add(ttl), CreatedAt: now,
	}
	if err := s.db.WithContext(ctx).Create(&challenge).Error; err != nil {
		return AppAttestChallengeView{}, err
	}
	needsAttestation := true
	if keyID = strings.TrimSpace(keyID); keyID != "" {
		var key models.AppAttestKey
		if err := s.db.WithContext(ctx).Where("key_id = ? AND device_hash = ?", keyID, deviceHash).First(&key).Error; err == nil {
			needsAttestation = false
		}
	}
	return AppAttestChallengeView{
		ChallengeID: challenge.ID, Challenge: nonceText,
		ClientData: base64.RawURLEncoding.EncodeToString(clientData),
		ExpiresAt:  challenge.ExpiresAt, NeedsKeyAttestation: needsAttestation,
	}, nil
}

// RegisterAppAttestKey verifies Apple's certificate chain, nonce, App ID,
// AAGUID, key identifier and signed app metadata before binding the key to the
// installation. A key or device can only be registered once.
func (s *SubscriptionService) RegisterAppAttestKey(
	ctx context.Context,
	userID uuid.UUID,
	deviceID, keyID string,
	challengeID uuid.UUID,
	attestationObject []byte,
) error {
	if !s.appAttestReady() {
		return ErrDeviceAttestationUnavailable
	}
	deviceHash, err := deviceDigest(deviceID)
	if err != nil {
		return err
	}
	keyID = strings.TrimSpace(keyID)
	if keyID == "" || challengeID == uuid.Nil || len(attestationObject) == 0 {
		return ErrInvalidDeviceAttestation
	}
	now := s.now()
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var challenge models.AppAttestChallenge
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", challengeID).First(&challenge).Error; err != nil {
			return ErrInvalidDeviceAttestation
		}
		if challenge.UserID != userID || challenge.DeviceHash != deviceHash || challenge.Purpose != appAttestPurposeFreeAI ||
			challenge.UsedAt != nil || !challenge.ExpiresAt.After(now) {
			return ErrInvalidDeviceAttestation
		}
		clientHash := sha256.Sum256(challenge.ClientData)
		if hex.EncodeToString(clientHash[:]) != challenge.NonceHash {
			return ErrInvalidDeviceAttestation
		}
		verified, err := s.appAttest.VerifyAttestation(attestationObject, clientHash[:], keyID)
		if err != nil {
			return ErrInvalidDeviceAttestation
		}
		key := models.AppAttestKey{
			KeyID: keyID, DeviceHash: deviceHash, RegisteredUserID: &userID,
			PublicKeyDER: verified.PublicKeyDER, Receipt: verified.Receipt,
			Environment: verified.Environment, Counter: 0, CreatedAt: now, UpdatedAt: now,
		}
		created := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&key)
		if created.Error != nil || created.RowsAffected != 1 {
			return ErrInvalidDeviceAttestation
		}
		return tx.Model(&challenge).Updates(map[string]any{"used_at": now, "client_data": []byte{}}).Error
	})
}

func (s *SubscriptionService) verifyAppAttestAssertionTx(
	ctx context.Context,
	tx *gorm.DB,
	userID uuid.UUID,
	deviceID, requestKey, scope string,
	proof AppAttestProof,
	now time.Time,
) error {
	if !s.appAttestReady() {
		return ErrDeviceAttestationUnavailable
	}
	deviceHash, err := deviceDigest(deviceID)
	if err != nil {
		return err
	}
	if proof.ChallengeID == uuid.Nil || strings.TrimSpace(proof.KeyID) == "" || len(proof.Assertion) == 0 {
		return ErrDeviceAttestationNeeded
	}
	var challenge models.AppAttestChallenge
	if err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", proof.ChallengeID).First(&challenge).Error; err != nil {
		return ErrInvalidDeviceAttestation
	}
	if challenge.UserID != userID || challenge.DeviceHash != deviceHash || challenge.RequestKey != requestKey ||
		challenge.Scope != scope || challenge.Purpose != appAttestPurposeFreeAI || challenge.UsedAt != nil || !challenge.ExpiresAt.After(now) {
		return ErrInvalidDeviceAttestation
	}
	clientHash := sha256.Sum256(challenge.ClientData)
	if hex.EncodeToString(clientHash[:]) != challenge.NonceHash {
		return ErrInvalidDeviceAttestation
	}
	var key models.AppAttestKey
	if err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Where("key_id = ?", strings.TrimSpace(proof.KeyID)).First(&key).Error; err != nil || key.DeviceHash != deviceHash {
		return ErrInvalidDeviceAttestation
	}
	newCounter, err := s.appAttest.VerifyAssertion(proof.Assertion, challenge.ClientData, key.PublicKeyDER, key.Counter)
	if err != nil || newCounter <= key.Counter {
		return ErrInvalidDeviceAttestation
	}
	if err := tx.Model(&models.AppAttestKey{}).Where("key_id = ? AND counter = ?", key.KeyID, key.Counter).
		Updates(map[string]any{"counter": newCounter, "updated_at": now}).Error; err != nil {
		return err
	}
	result := tx.Model(&challenge).Where("used_at IS NULL").Updates(map[string]any{"used_at": now, "client_data": []byte{}})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrInvalidDeviceAttestation
	}
	return nil
}

func appAttestDecode(value string, maxBytes int) ([]byte, error) {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > (maxBytes*4/3)+8 {
		return nil, ErrInvalidDeviceAttestation
	}
	for _, encoding := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.RawURLEncoding} {
		decoded, err := encoding.DecodeString(value)
		if err == nil && len(decoded) > 0 && len(decoded) <= maxBytes {
			return decoded, nil
		}
	}
	return nil, ErrInvalidDeviceAttestation
}

func DecodeAppAttestObject(value string) ([]byte, error) {
	return appAttestDecode(value, maxAppAttestObjectBytes)
}

func isAppAttestError(err error) bool {
	return errors.Is(err, ErrDeviceAttestationNeeded) || errors.Is(err, ErrDeviceAttestationUnavailable) || errors.Is(err, ErrInvalidDeviceAttestation)
}
