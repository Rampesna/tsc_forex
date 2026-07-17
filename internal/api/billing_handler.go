package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/tsc-forex/tsc_forex/internal/service"
)

type BillingHandler struct {
	service *service.SubscriptionService
}

func NewBillingHandler(subscription *service.SubscriptionService) *BillingHandler {
	return &BillingHandler{service: subscription}
}

func (h *BillingHandler) Plans(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"purchase_channel": "ios_app_store",
		"plans":            service.SubscriptionPlans(),
		"free_trial": gin.H{
			"channel": "ios_app_attest_devicecheck", "ready": h.service.FreeTrialReady(),
		},
	})
}

func (h *BillingHandler) Status(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
		return
	}
	status, err := h.service.Status(c.Request.Context(), uid, c.Query("device_id"))
	if err != nil {
		writeSubscriptionError(c, err, "")
		return
	}
	c.JSON(http.StatusOK, status)
}

func (h *BillingHandler) SyncApple(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 128<<10)
	var request struct {
		SignedTransaction string `json:"signed_transaction" binding:"required"`
		DeviceID          string `json:"device_id"`
		RecoveryToken     string `json:"subscription_recovery_token"`
	}
	if err := c.ShouldBindJSON(&request); err != nil || strings.TrimSpace(request.SignedTransaction) == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "imzalı App Store işlemi gerekli"})
		return
	}
	status, err := h.service.SyncAppleTransaction(c.Request.Context(), uid, request.SignedTransaction, request.RecoveryToken)
	if err != nil {
		writeSubscriptionError(c, err, "")
		return
	}
	// Include device-aware free-trial state for a consistent response shape.
	if request.DeviceID != "" {
		if refreshed, statusErr := h.service.Status(c.Request.Context(), uid, request.DeviceID); statusErr == nil {
			status = refreshed
		}
	}
	c.JSON(http.StatusOK, status)
}

// AppleNotifications is the public App Store Server Notifications V2 URL.
// Authenticity comes from the nested Apple JWS certificate/signature chain;
// an arbitrary JSON POST cannot mutate an entitlement.
func (h *BillingHandler) AppleNotifications(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 512<<10)
	var request struct {
		SignedPayload string `json:"signedPayload" binding:"required"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "signedPayload gerekli"})
		return
	}
	if err := h.service.ProcessAppleNotification(c.Request.Context(), request.SignedPayload); err != nil {
		writeSubscriptionError(c, err, "")
		return
	}
	c.Status(http.StatusOK)
}

func (h *BillingHandler) AppAttestChallenge(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 16<<10)
	var request struct {
		DeviceID  string `json:"device_id" binding:"required"`
		KeyID     string `json:"key_id"`
		Purpose   string `json:"purpose" binding:"required"`
		RequestID string `json:"request_id" binding:"required"`
		Code      string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&request); err != nil || request.Purpose != "free_ai" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "geçerli App Attest isteği gerekli"})
		return
	}
	challenge, err := h.service.IssueAppAttestChallenge(
		c.Request.Context(), uid, request.DeviceID, request.KeyID, request.RequestID, request.Code,
	)
	if err != nil {
		writeSubscriptionError(c, err, service.FeatureAIResearch)
		return
	}
	c.JSON(http.StatusOK, challenge)
}

func (h *BillingHandler) RegisterAppAttest(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 128<<10)
	var request struct {
		DeviceID          string `json:"device_id" binding:"required"`
		KeyID             string `json:"key_id" binding:"required"`
		ChallengeID       string `json:"challenge_id" binding:"required"`
		AttestationObject string `json:"attestation_object" binding:"required"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "geçerli App Attest kaydı gerekli"})
		return
	}
	challengeID, parseErr := uuid.Parse(strings.TrimSpace(request.ChallengeID))
	attestation, decodeErr := service.DecodeAppAttestObject(request.AttestationObject)
	if parseErr != nil || decodeErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "App Attest verisi geçersiz", "code": "invalid_device_attestation"})
		return
	}
	if err := h.service.RegisterAppAttestKey(c.Request.Context(), uid, request.DeviceID, request.KeyID, challengeID, attestation); err != nil {
		writeSubscriptionError(c, err, service.FeatureAIResearch)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"registered": true, "key_id": request.KeyID})
}

func (h *BillingHandler) ResearchHistory(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	items, err := h.service.ResearchHistory(c.Request.Context(), uid, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "araştırma arşivi yüklenemedi"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func writeSubscriptionError(c *gin.Context, err error, feature string) {
	var quota *service.QuotaError
	switch {
	case errors.As(err, &quota):
		c.JSON(http.StatusPaymentRequired, gin.H{
			"error": quota.Error(), "code": quota.Code, "feature": quota.Feature,
			"limit": quota.Limit, "used": quota.Used,
		})
	case errors.Is(err, service.ErrFreeResearchUnavailable), errors.Is(err, service.ErrSubscriptionRequired):
		c.JSON(http.StatusPaymentRequired, gin.H{
			"error": "Ücretsiz araştırma hakkı kullanılmış; devam etmek için bir paket gerekli.",
			"code":  "subscription_required", "feature": coalesceFeature(feature, service.FeatureAIResearch),
		})
	case errors.Is(err, service.ErrInvalidDeviceID):
		c.JSON(http.StatusBadRequest, gin.H{"error": "güvenli cihaz kimliği gerekli", "code": "device_id_required", "feature": feature})
	case errors.Is(err, service.ErrDeviceAttestationNeeded):
		c.JSON(http.StatusForbidden, gin.H{"error": "Ücretsiz hak için doğrulanmış iOS uygulaması gerekli", "code": "device_attestation_required", "feature": feature})
	case errors.Is(err, service.ErrDeviceAttestationUnavailable):
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Cihaz doğrulama hizmeti hazır değil", "code": "device_attestation_unavailable", "feature": feature})
	case errors.Is(err, service.ErrInvalidDeviceAttestation):
		c.JSON(http.StatusForbidden, gin.H{"error": "Cihaz doğrulaması geçersiz veya süresi dolmuş", "code": "invalid_device_attestation", "feature": feature})
	case errors.Is(err, service.ErrInvalidDeviceCheckToken):
		c.JSON(http.StatusForbidden, gin.H{"error": "Apple DeviceCheck doğrulaması geçersiz", "code": "invalid_device_check", "feature": feature})
	case errors.Is(err, service.ErrDeviceCheckUnavailable):
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Apple DeviceCheck hizmeti hazır değil", "code": "device_check_unavailable", "feature": feature})
	case errors.Is(err, service.ErrOperationInProgress):
		c.JSON(http.StatusConflict, gin.H{"error": "İşlem halen tamamlanıyor", "code": "operation_in_progress", "feature": feature})
	case errors.Is(err, service.ErrRequestConflict):
		c.JSON(http.StatusConflict, gin.H{"error": "istek kimliği farklı bir işlem için kullanılmış", "code": "request_conflict", "feature": feature})
	case errors.Is(err, service.ErrTransactionAccount):
		c.JSON(http.StatusConflict, gin.H{"error": "Bu App Store aboneliği başka bir hesaba bağlı.", "code": "transaction_account_conflict"})
	case errors.Is(err, service.ErrInvalidAppleTransaction):
		c.JSON(http.StatusBadRequest, gin.H{"error": "App Store işlemi doğrulanamadı", "code": "invalid_apple_transaction"})
	case errors.Is(err, service.ErrBillingDisabled):
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "abonelik hizmeti geçici olarak kullanılamıyor", "code": "billing_unavailable"})
	default:
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "abonelik işlemi tamamlanamadı"})
	}
}

func coalesceFeature(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}
