package api

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/tsc-forex/tsc_forex/internal/config"
	"github.com/tsc-forex/tsc_forex/internal/domain"
	"github.com/tsc-forex/tsc_forex/internal/models"
	"github.com/tsc-forex/tsc_forex/internal/service"
)

const marketInsightDisclaimer = "Bu içerik yatırım tavsiyesi değildir; eğitim ve karar desteği amaçlıdır."

// MarketPriceReader supplies the latest trusted server-side market snapshot.
type MarketPriceReader interface {
	Get(code string) (domain.InstrumentPrice, bool)
}

// MarketCandleReader supplies durable daily history for multi-timeframe and
// cross-asset context. Web search complements this data; it never replaces it.
type MarketCandleReader interface {
	Candles(ctx context.Context, code, interval string, from, to time.Time, limit int) ([]models.OHLC, error)
}

// MarketAIHandler generates short, structured market briefs without exposing
// provider credentials to browser or mobile clients.
type MarketAIHandler struct {
	cfg     config.AIConfig
	prices  MarketPriceReader
	candles MarketCandleReader
	client  *http.Client
	billing *service.SubscriptionService
	mu      sync.Mutex
	cache   map[string]cachedMarketInsight
}

type cachedMarketInsight struct {
	value     MarketInsightResponse
	expiresAt time.Time
}

type marketInsightRequest struct {
	Code                   string `json:"code" binding:"required"`
	JobID                  string `json:"job_id,omitempty"`
	DeviceID               string `json:"device_id,omitempty"`
	RequestID              string `json:"request_id,omitempty"`
	AttestationKeyID       string `json:"attestation_key_id,omitempty"`
	AttestationChallengeID string `json:"attestation_challenge_id,omitempty"`
	AttestationAssertion   string `json:"attestation_assertion,omitempty"`
	DeviceCheckToken       string `json:"device_check_token,omitempty"`
}

type marketInsightPendingResponse struct {
	Status    string `json:"status"`
	JobID     string `json:"job_id"`
	Code      string `json:"code"`
	RequestID string `json:"request_id,omitempty"`
}

type marketInsightJobClaims struct {
	ResponseID string `json:"response_id"`
	Code       string `json:"code"`
	UserID     string `json:"user_id"`
	RequestID  string `json:"request_id"`
	ExpiresAt  int64  `json:"expires_at"`
}

// MarketInsightResponse is intentionally scenario-oriented, not a buy/sell
// signal. The client can render it without parsing untrusted markdown.
type MarketInsightResponse struct {
	Code               string              `json:"code"`
	Title              string              `json:"title"`
	Stance             string              `json:"stance"`
	StanceLabel        string              `json:"stance_label"`
	Summary            string              `json:"summary"`
	Factors            []string            `json:"factors"`
	RiskNote           string              `json:"risk_note"`
	Confidence         string              `json:"confidence"`
	TimeHorizons       []MarketHorizonView `json:"time_horizons"`
	MacroDrivers       []string            `json:"macro_drivers"`
	GeopoliticalRisks  []string            `json:"geopolitical_risks"`
	CrossAssetAnalysis []string            `json:"cross_asset_analysis"`
	Scenarios          []MarketScenario    `json:"scenarios"`
	WhatIWouldDo       MarketWhatIWouldDo  `json:"what_i_would_do"`
	DataContext        []MarketDataPoint   `json:"data_context"`
	Sources            []MarketSource      `json:"sources"`
	ResearchMode       string              `json:"research_mode"`
	GeneratedAt        string              `json:"generated_at"`
	Disclaimer         string              `json:"disclaimer"`
}

type MarketHorizonView struct {
	Horizon   string `json:"horizon"`
	Outlook   string `json:"outlook"`
	Rationale string `json:"rationale"`
}

// MarketWhatIWouldDo is a first-person, non-advisory closing playbook.
// It must stay educational ("ben olsam…") and never issue buy/sell orders.
type MarketWhatIWouldDo struct {
	Intro      string `json:"intro"`
	ShortTerm  string `json:"short_term"`
	MediumTerm string `json:"medium_term"`
	LongTerm   string `json:"long_term"`
}

type MarketScenario struct {
	Name        string `json:"name"`
	Likelihood  string `json:"likelihood"`
	Trigger     string `json:"trigger"`
	Implication string `json:"implication"`
}

type MarketDataPoint struct {
	Code        string   `json:"code"`
	DisplayName string   `json:"display_name"`
	Price       float64  `json:"price"`
	Currency    string   `json:"currency"`
	Change1D    *float64 `json:"change_1d_pct,omitempty"`
	Change1W    *float64 `json:"change_1w_pct,omitempty"`
	Change1M    *float64 `json:"change_1m_pct,omitempty"`
	Change3M    *float64 `json:"change_3m_pct,omitempty"`
	Change1Y    *float64 `json:"change_1y_pct,omitempty"`
	Timestamp   string   `json:"timestamp"`
}

type MarketSource struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

type openAIResponsesPayload struct {
	Model           string                `json:"model"`
	Instructions    string                `json:"instructions"`
	Input           string                `json:"input"`
	MaxOutputTokens int                   `json:"max_output_tokens"`
	MaxToolCalls    int                   `json:"max_tool_calls,omitempty"`
	SafetyID        string                `json:"safety_identifier,omitempty"`
	Store           bool                  `json:"store"`
	Background      bool                  `json:"background,omitempty"`
	ToolChoice      string                `json:"tool_choice,omitempty"`
	Tools           []openAIWebSearchTool `json:"tools,omitempty"`
	Include         []string              `json:"include,omitempty"`
	Reasoning       openAIReasoningSpec   `json:"reasoning"`
	Text            openAITextSpec        `json:"text"`
}

type openAIReasoningSpec struct {
	Effort string `json:"effort"`
}

type openAIWebSearchTool struct {
	Type              string `json:"type"`
	SearchContextSize string `json:"search_context_size,omitempty"`
}

type openAITextSpec struct {
	Format openAIFormatSpec `json:"format"`
}

type openAIFormatSpec struct {
	Type   string         `json:"type"`
	Name   string         `json:"name"`
	Strict bool           `json:"strict"`
	Schema map[string]any `json:"schema"`
}

type openAIResponsesEnvelope struct {
	ID                string `json:"id"`
	Status            string `json:"status"`
	IncompleteDetails struct {
		Reason string `json:"reason"`
	} `json:"incomplete_details"`
	Output []struct {
		Type   string `json:"type"`
		Action struct {
			Sources []struct {
				Type  string `json:"type"`
				Title string `json:"title"`
				URL   string `json:"url"`
			} `json:"sources"`
		} `json:"action"`
		Content []struct {
			Type        string `json:"type"`
			Text        string `json:"text"`
			Annotations []struct {
				Type  string `json:"type"`
				Title string `json:"title"`
				URL   string `json:"url"`
			} `json:"annotations"`
		} `json:"content"`
	} `json:"output"`
}

func NewMarketAIHandler(cfg config.AIConfig, prices MarketPriceReader, candleReaders ...MarketCandleReader) *MarketAIHandler {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 55 * time.Second
	}
	var candles MarketCandleReader
	if len(candleReaders) > 0 {
		candles = candleReaders[0]
	}
	return &MarketAIHandler{
		cfg: cfg, prices: prices, candles: candles,
		client: &http.Client{Timeout: timeout},
		cache:  make(map[string]cachedMarketInsight),
	}
}

func (h *MarketAIHandler) WithSubscriptions(subscription *service.SubscriptionService) *MarketAIHandler {
	h.billing = subscription
	return h
}

// MarketInsight godoc
// @Summary  Generate an AI-assisted market brief for one instrument
// @Tags     ai
// @Accept   json
// @Produce  json
// @Security BearerAuth
// @Router   /me/ai/market-insight [post]
func (h *MarketAIHandler) MarketInsight(c *gin.Context) {
	uid, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
		return
	}
	if !h.cfg.Enabled || strings.TrimSpace(h.cfg.APIKey) == "" {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "AI özelliği henüz yapılandırılmadı"})
		return
	}
	var req marketInsightRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "geçerli bir varlık kodu gerekli"})
		return
	}
	req.Code = strings.ToUpper(strings.TrimSpace(req.Code))
	if !validCode(req.Code) || h.prices == nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "geçersiz varlık kodu"})
		return
	}
	price, found := h.prices.Get(req.Code)
	if !found {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "varlık için canlı fiyat henüz hazır değil"})
		return
	}
	initialRequest := strings.TrimSpace(req.JobID) == ""
	requestID := strings.TrimSpace(req.RequestID)
	var claims marketInsightJobClaims
	if !initialRequest {
		var verifyErr error
		claims, verifyErr = h.verifyJobToken(req.JobID, uid.String(), req.Code)
		if verifyErr != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "geçersiz veya süresi dolmuş araştırma işi"})
			return
		}
		requestID = claims.RequestID
	} else if requestID == "" {
		requestID = uuid.NewString()
	}

	ownerToken := ""
	if h.billing != nil {
		if initialRequest {
			proof, proofErr := marketAppAttestProof(req)
			if proofErr != nil {
				writeSubscriptionError(c, proofErr, service.FeatureAIResearch)
				return
			}
			job, beginErr := h.billing.BeginAIResearchJob(c.Request.Context(), uid, req.DeviceID, requestID, req.Code, proof)
			if beginErr != nil {
				writeSubscriptionError(c, beginErr, service.FeatureAIResearch)
				return
			}
			if !job.Created {
				if h.writeExistingResearchJob(c, uid, job) {
					return
				}
			}
			ownerToken = job.OwnerToken
			if cached, cachedFound := h.cached(req.Code); cachedFound {
				payload, marshalErr := json.Marshal(cached)
				if marshalErr != nil || h.billing.CompleteAIResearchJob(c.Request.Context(), uid, requestID, ownerToken, "", payload) != nil {
					_ = h.billing.ReleaseAIResearchJob(c.Request.Context(), uid, requestID, ownerToken, "cache_persist_failed")
					c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "araştırma sonucu kaydedilemedi"})
					return
				}
				h.archiveResearch(c.Request.Context(), uid, requestID, req.Code, cached)
				c.JSON(http.StatusOK, cached)
				return
			}
		} else {
			job, jobErr := h.billing.AIResearchJob(c.Request.Context(), uid, requestID, req.Code)
			if jobErr != nil {
				c.JSON(http.StatusBadRequest, ErrorResponse{Error: "araştırma işi bulunamadı"})
				return
			}
			if job.Status == service.ResearchJobCompleted {
				h.writeResearchResult(c, job.ResultPayload)
				return
			}
			if job.Status == service.ResearchJobFailed {
				c.JSON(http.StatusBadGateway, ErrorResponse{Error: "Derin piyasa araştırması tamamlanamadı; yeni bir istek başlatın"})
				return
			}
			if job.Status == service.ResearchJobReserved {
				leased, leaseErr := h.billing.LeaseAIResearchJob(c.Request.Context(), uid, requestID, req.Code)
				if leaseErr != nil {
					writeSubscriptionError(c, leaseErr, service.FeatureAIResearch)
					return
				}
				if !leased.Created {
					c.JSON(http.StatusAccepted, marketInsightPendingResponse{Status: "queued", JobID: req.JobID, Code: req.Code, RequestID: requestID})
					return
				}
				// The original creator died before attaching a provider response.
				// This request now owns the rotated durable lease and restarts with
				// the same provider idempotency key and no additional quota spend.
				ownerToken = leased.OwnerToken
				initialRequest = true
				req.JobID = ""
			} else if job.ProviderResponseID == "" {
				c.JSON(http.StatusAccepted, marketInsightPendingResponse{Status: "queued", JobID: req.JobID, Code: req.Code, RequestID: requestID})
				return
			} else {
				claims.ResponseID = job.ProviderResponseID
			}
		}
	} else if initialRequest {
		if cached, cachedFound := h.cached(req.Code); cachedFound {
			c.JSON(http.StatusOK, cached)
			return
		}
	}

	dataContext := h.researchContext(c.Request.Context(), price)
	digest := sha256.Sum256([]byte(uid.String()))
	safetyID := "market_research_" + hex.EncodeToString(digest[:12])
	var envelope openAIResponsesEnvelope
	var err error
	if initialRequest {
		envelope, err = h.startBackground(
			c.Request.Context(), price, dataContext, safetyID,
			providerIdempotencyKey(uid, requestID, req.Code),
		)
	} else {
		envelope, err = h.retrieveBackground(c.Request.Context(), claims.ResponseID)
	}
	if err != nil {
		// The request may have reached the provider even when its response was
		// lost. Keep the durable reservation; the same request can reclaim its
		// lease and reuse the provider idempotency key without another quota use.
		c.JSON(http.StatusBadGateway, ErrorResponse{Error: "Derin piyasa araştırması şu anda tamamlanamadı"})
		return
	}
	if envelope.Status == "queued" || envelope.Status == "in_progress" {
		jobID := req.JobID
		if jobID == "" {
			if h.billing != nil {
				if err := h.billing.AttachAIResearchProvider(c.Request.Context(), uid, requestID, ownerToken, envelope.ID); err != nil {
					c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "araştırma işi kaydedilemedi"})
					return
				}
			}
			jobID, err = h.signJobToken(envelope.ID, req.Code, uid.String(), requestID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "araştırma işi oluşturulamadı"})
				return
			}
		}
		c.JSON(http.StatusAccepted, marketInsightPendingResponse{Status: envelope.Status, JobID: jobID, Code: req.Code, RequestID: requestID})
		return
	}
	result, err := h.finalizeResponse(envelope, price, dataContext)
	if err != nil {
		if h.billing != nil {
			if initialRequest && ownerToken != "" {
				_ = h.billing.MarkAIResearchReservationFailed(c.Request.Context(), uid, requestID, ownerToken, "provider_invalid_result")
			} else if claims.ResponseID != "" {
				_ = h.billing.MarkAIResearchProviderFailed(c.Request.Context(), uid, requestID, claims.ResponseID, "provider_invalid_result")
			}
		}
		c.JSON(http.StatusBadGateway, ErrorResponse{Error: "Derin piyasa araştırması tamamlanamadı; yeniden deneyin"})
		return
	}
	h.store(req.Code, result)
	if h.billing != nil {
		payload, marshalErr := json.Marshal(result)
		providerID := claims.ResponseID
		if initialRequest {
			providerID = ""
		}
		if marshalErr != nil || h.billing.CompleteAIResearchJob(c.Request.Context(), uid, requestID, ownerToken, providerID, payload) != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "araştırma sonucu kaydedilemedi"})
			return
		}
	}
	h.archiveResearch(c.Request.Context(), uid, requestID, req.Code, result)
	c.JSON(http.StatusOK, result)
}

func marketAppAttestProof(req marketInsightRequest) (service.AppAttestProof, error) {
	if strings.TrimSpace(req.AttestationChallengeID) == "" && strings.TrimSpace(req.AttestationAssertion) == "" && strings.TrimSpace(req.AttestationKeyID) == "" {
		return service.AppAttestProof{}, nil
	}
	challengeID, err := uuid.Parse(strings.TrimSpace(req.AttestationChallengeID))
	if err != nil {
		return service.AppAttestProof{}, service.ErrInvalidDeviceAttestation
	}
	assertion, err := service.DecodeAppAttestObject(req.AttestationAssertion)
	if err != nil {
		return service.AppAttestProof{}, err
	}
	return service.AppAttestProof{
		KeyID: strings.TrimSpace(req.AttestationKeyID), ChallengeID: challengeID,
		Assertion: assertion, DeviceCheckToken: strings.TrimSpace(req.DeviceCheckToken),
	}, nil
}

func (h *MarketAIHandler) writeExistingResearchJob(c *gin.Context, userID uuid.UUID, job service.AIResearchJob) bool {
	switch job.Status {
	case service.ResearchJobCompleted:
		h.writeResearchResult(c, job.ResultPayload)
		return true
	case service.ResearchJobFailed:
		c.JSON(http.StatusBadGateway, ErrorResponse{Error: "Derin piyasa araştırması tamamlanamadı; yeni bir istek kimliği kullanın"})
		return true
	case service.ResearchJobReserved, service.ResearchJobPending:
		jobID, err := h.signJobToken(job.ProviderResponseID, job.Code, userID.String(), job.RequestKey)
		if err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "araştırma işi yüklenemedi"})
			return true
		}
		status := "queued"
		if job.Status == service.ResearchJobPending {
			status = "in_progress"
		}
		c.JSON(http.StatusAccepted, marketInsightPendingResponse{Status: status, JobID: jobID, Code: job.Code, RequestID: job.RequestKey})
		return true
	default:
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "araştırma işi geçersiz durumda"})
		return true
	}
}

func (h *MarketAIHandler) writeResearchResult(c *gin.Context, payload json.RawMessage) {
	var result MarketInsightResponse
	if !json.Valid(payload) || json.Unmarshal(payload, &result) != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "araştırma sonucu okunamadı"})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *MarketAIHandler) archiveResearch(ctx context.Context, userID uuid.UUID, requestID, code string, result MarketInsightResponse) {
	if h.billing == nil || requestID == "" {
		return
	}
	payload, err := json.Marshal(result)
	if err == nil {
		_ = h.billing.ArchiveResearch(ctx, userID, requestID, code, payload)
	}
}

func (h *MarketAIHandler) generate(ctx context.Context, p domain.InstrumentPrice, dataContext []MarketDataPoint, safetyID string) (MarketInsightResponse, error) {
	payload, baseURL, err := h.buildPayload(p, dataContext, safetyID, false)
	if err != nil {
		return MarketInsightResponse{}, err
	}
	envelope, err := h.createResponse(ctx, baseURL, payload, "")
	if err != nil {
		return MarketInsightResponse{}, err
	}
	return h.finalizeResponse(envelope, p, dataContext)
}

func (h *MarketAIHandler) startBackground(
	ctx context.Context,
	p domain.InstrumentPrice,
	dataContext []MarketDataPoint,
	safetyID, idempotencyKey string,
) (openAIResponsesEnvelope, error) {
	payload, baseURL, err := h.buildPayload(p, dataContext, safetyID, true)
	if err != nil {
		return openAIResponsesEnvelope{}, err
	}
	return h.createResponse(ctx, baseURL, payload, idempotencyKey)
}

func (h *MarketAIHandler) buildPayload(p domain.InstrumentPrice, dataContext []MarketDataPoint, safetyID string, background bool) (openAIResponsesPayload, string, error) {
	model := h.model()
	maxTokens := h.cfg.MaxOutputTokens
	if maxTokens <= 0 {
		maxTokens = 8000
	}
	baseURL := strings.TrimRight(strings.TrimSpace(h.cfg.BaseURL), "/")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	reasoningEffort := strings.TrimSpace(h.cfg.ReasoningEffort)
	if reasoningEffort == "" {
		reasoningEffort = "high"
	}
	searchContext := strings.TrimSpace(h.cfg.SearchContext)
	if searchContext == "" {
		searchContext = "high"
	}
	maxToolCalls := h.cfg.MaxToolCalls
	if maxToolCalls <= 0 {
		maxToolCalls = 6
	}
	// The report needs enough searches to cover macro, cross-asset and
	// geopolitical evidence, but an operator typo must not create unbounded
	// provider spend for a single quota unit.
	if maxToolCalls > 8 {
		maxToolCalls = 8
	}

	contextJSON, err := json.Marshal(dataContext)
	if err != nil {
		return openAIResponsesPayload{}, "", err
	}
	prompt := fmt.Sprintf(
		"Araştırma zamanı: %s\nSeçili varlık: %s (%s), tür=%s\nSon fiyat: %.8f %s; TRY karşılığı: %.4f; alış: %.8f; satış: %.8f; günlük değişim: %.4f%%\nSunucu tarafından doğrulanan çok-zaman-dilimli ve çapraz-varlık verisi (JSON):\n%s\n\nBu veriyi güncel web araştırmasıyla birleştir. Bir hafta, bir ay, üç ay ve bir yıllık eğilimleri ayrı değerlendir. Fiyat aktarım mekanizmasını; ilgili döviz sepetlerini, dolar endeksini, tahvil/faiz ve reel getiri dinamiklerini, emtia/kripto/hisse ilişkilerini, merkez bankası kararlarını, enflasyonu ve yaklaşan veri takvimini varlıkla ilgisi ölçüsünde incele. Güncel jeopolitik çatışma, yaptırım ve ülkeler arası gerilimlerde yalnızca fiyatı etkileyebilecek kanıtlı gelişmeleri dahil et. Resmî ve birincil kaynaklara öncelik ver, en az dört güncel kaynak kullan, olay tarihi ile yayın tarihini ayır ve araştırmayı en fazla %d araç çağrısında tamamla.\n\nSon olarak what_i_would_do alanını doldur: intro 'Yatırım tavsiyesi olmamakla beraber, ben olsam…' diye başlasın; short_term (1-4 hafta), medium_term (1-3 ay) ve long_term (6-12 ay) için gerçekçi, birinci şahıs, eyleme dönük ama emir vermeyen kısa özet yaz (ör. beklerdim, izlerdim, kademeli biriktirirdim, riskimi azaltırdım, nakit ağırlığını korurdum).",
		time.Now().UTC().Format(time.RFC3339), p.DisplayName, p.Code, p.Kind, p.Price, p.Currency, p.PriceTRY, p.Bid, p.Ask, p.ChangePct, string(contextJSON), maxToolCalls,
	)
	payload := openAIResponsesPayload{
		Model: model,
		Instructions: "Türkçe yazan, kanıta dayalı kıdemli bir makro ve çapraz-varlık araştırma analistisin. Sunucunun sağladığı fiyat ve tarihsel değişim verilerini sayısal gerçek olarak kabul et; web'i güncel makroekonomi, merkez bankaları, emtia aktarımı, veri takvimi ve jeopolitik gelişmeler için kullan. Resmî kurumları, merkez bankalarını, borsaları ve güçlü finans haber kaynaklarını önceliklendir. Güncelliği ve olay tarihini kontrol et. Bir iddiayı kaynakla destekleyemiyorsan çıkar. Korelasyonu nedensellik gibi sunma. Kesin fiyat tahmini, al/sat emri, hedef fiyat veya garanti verme. Boğa, dengeli ve ayı senaryolarını tetik ve sonuçlarıyla ayır. time_horizons analitik görünüm kalsın; what_i_would_do ise raporun kişisel kapanış özeti olsun ve yatırım tavsiyesi olmadığını açıkça söylesin. Sonuç ayrıntılı ama tekrarsız olsun. Son kullanıcı metninde sağlayıcı, model, altyapı veya iç ürün/marka adı anma.",
		Input: prompt, MaxOutputTokens: maxTokens, MaxToolCalls: maxToolCalls,
		SafetyID: safetyID, Store: false, Background: background,
		Reasoning: openAIReasoningSpec{Effort: reasoningEffort},
		Text: openAITextSpec{Format: openAIFormatSpec{
			Type: "json_schema", Name: "market_research", Strict: true,
			Schema: map[string]any{
				"type": "object", "additionalProperties": false,
				"properties": map[string]any{
					"title":        map[string]any{"type": "string"},
					"stance":       map[string]any{"type": "string", "enum": []string{"positive", "neutral", "cautious"}},
					"stance_label": map[string]any{"type": "string"},
					"summary":      map[string]any{"type": "string"},
					"factors":      map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "minItems": 4, "maxItems": 6},
					"risk_note":    map[string]any{"type": "string"},
					"confidence":   map[string]any{"type": "string", "enum": []string{"low", "medium", "high"}},
					"time_horizons": map[string]any{
						"type": "array", "minItems": 3, "maxItems": 3,
						"items": map[string]any{
							"type": "object", "additionalProperties": false,
							"properties": map[string]any{
								"horizon":   map[string]any{"type": "string"},
								"outlook":   map[string]any{"type": "string"},
								"rationale": map[string]any{"type": "string"},
							},
							"required": []string{"horizon", "outlook", "rationale"},
						},
					},
					"macro_drivers":        map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "minItems": 3, "maxItems": 6},
					"geopolitical_risks":   map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "minItems": 2, "maxItems": 5},
					"cross_asset_analysis": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "minItems": 3, "maxItems": 6},
					"scenarios": map[string]any{
						"type": "array", "minItems": 3, "maxItems": 3,
						"items": map[string]any{
							"type": "object", "additionalProperties": false,
							"properties": map[string]any{
								"name":        map[string]any{"type": "string"},
								"likelihood":  map[string]any{"type": "string", "enum": []string{"low", "medium", "high"}},
								"trigger":     map[string]any{"type": "string"},
								"implication": map[string]any{"type": "string"},
							},
							"required": []string{"name", "likelihood", "trigger", "implication"},
						},
					},
					"what_i_would_do": map[string]any{
						"type": "object", "additionalProperties": false,
						"properties": map[string]any{
							"intro":       map[string]any{"type": "string"},
							"short_term":  map[string]any{"type": "string"},
							"medium_term": map[string]any{"type": "string"},
							"long_term":   map[string]any{"type": "string"},
						},
						"required": []string{"intro", "short_term", "medium_term", "long_term"},
					},
				},
				"required": []string{"title", "stance", "stance_label", "summary", "factors", "risk_note", "confidence", "time_horizons", "macro_drivers", "geopolitical_risks", "cross_asset_analysis", "scenarios", "what_i_would_do"},
			},
		}},
	}
	if h.cfg.WebSearch {
		payload.ToolChoice = "auto"
		payload.Tools = []openAIWebSearchTool{{Type: "web_search", SearchContextSize: searchContext}}
		payload.Include = []string{"web_search_call.action.sources"}
	}
	return payload, baseURL, nil
}

func (h *MarketAIHandler) createResponse(ctx context.Context, baseURL string, payload openAIResponsesPayload, idempotencyKey string) (openAIResponsesEnvelope, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return openAIResponsesEnvelope{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/responses", bytes.NewReader(body))
	if err != nil {
		return openAIResponsesEnvelope{}, err
	}
	req.Header.Set("Authorization", "Bearer "+h.cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(idempotencyKey) != "" {
		req.Header.Set("Idempotency-Key", idempotencyKey)
	}
	res, err := h.client.Do(req)
	if err != nil {
		return openAIResponsesEnvelope{}, err
	}
	defer res.Body.Close()
	limited := io.LimitReader(res.Body, 1<<20)
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, limited)
		return openAIResponsesEnvelope{}, fmt.Errorf("openai status %d", res.StatusCode)
	}
	var envelope openAIResponsesEnvelope
	if err := json.NewDecoder(limited).Decode(&envelope); err != nil {
		return openAIResponsesEnvelope{}, err
	}
	return envelope, nil
}

func (h *MarketAIHandler) retrieveBackground(ctx context.Context, responseID string) (openAIResponsesEnvelope, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(h.cfg.BaseURL), "/")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/responses/"+url.PathEscape(responseID), nil)
	if err != nil {
		return openAIResponsesEnvelope{}, err
	}
	req.Header.Set("Authorization", "Bearer "+h.cfg.APIKey)
	res, err := h.client.Do(req)
	if err != nil {
		return openAIResponsesEnvelope{}, err
	}
	defer res.Body.Close()
	limited := io.LimitReader(res.Body, 1<<20)
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, limited)
		return openAIResponsesEnvelope{}, fmt.Errorf("openai retrieve status %d", res.StatusCode)
	}
	var envelope openAIResponsesEnvelope
	if err := json.NewDecoder(limited).Decode(&envelope); err != nil {
		return openAIResponsesEnvelope{}, err
	}
	return envelope, nil
}

func (h *MarketAIHandler) finalizeResponse(envelope openAIResponsesEnvelope, p domain.InstrumentPrice, dataContext []MarketDataPoint) (MarketInsightResponse, error) {
	if envelope.Status != "" && envelope.Status != "completed" {
		return MarketInsightResponse{}, fmt.Errorf("openai response status %s: %s", envelope.Status, envelope.IncompleteDetails.Reason)
	}
	output := ""
	for _, item := range envelope.Output {
		for _, content := range item.Content {
			if content.Type == "output_text" && content.Text != "" {
				output = content.Text
				break
			}
		}
	}
	if output == "" {
		return MarketInsightResponse{}, errors.New("missing output_text")
	}
	var insight MarketInsightResponse
	if err := json.Unmarshal([]byte(output), &insight); err != nil {
		return MarketInsightResponse{}, err
	}
	if insight.Stance != "positive" && insight.Stance != "neutral" && insight.Stance != "cautious" {
		return MarketInsightResponse{}, errors.New("invalid stance")
	}
	if strings.TrimSpace(insight.WhatIWouldDo.Intro) == "" ||
		strings.TrimSpace(insight.WhatIWouldDo.ShortTerm) == "" ||
		strings.TrimSpace(insight.WhatIWouldDo.MediumTerm) == "" ||
		strings.TrimSpace(insight.WhatIWouldDo.LongTerm) == "" {
		return MarketInsightResponse{}, errors.New("missing what_i_would_do")
	}
	insight.Code = p.Code
	insight.DataContext = dataContext
	insight.Sources = extractMarketSources(envelope)
	insight.ResearchMode = "market_data"
	if len(insight.Sources) > 0 {
		insight.ResearchMode = "web+market_data"
	}
	insight.GeneratedAt = time.Now().UTC().Format(time.RFC3339)
	insight.Disclaimer = marketInsightDisclaimer
	return insight, nil
}

func (h *MarketAIHandler) model() string {
	model := strings.TrimSpace(h.cfg.Model)
	if model == "" {
		return "gpt-5.6-sol"
	}
	return model
}

func providerIdempotencyKey(userID uuid.UUID, requestID, code string) string {
	digest := sha256.Sum256([]byte(userID.String() + ":" + strings.TrimSpace(requestID) + ":" + strings.ToUpper(strings.TrimSpace(code))))
	return "market-research-" + hex.EncodeToString(digest[:])
}

func (h *MarketAIHandler) signJobToken(responseID, code, userID, requestID string) (string, error) {
	claimsJSON, err := json.Marshal(marketInsightJobClaims{
		ResponseID: responseID, Code: code, UserID: userID, RequestID: requestID, ExpiresAt: time.Now().Add(15 * time.Minute).Unix(),
	})
	if err != nil {
		return "", err
	}
	payload := base64.RawURLEncoding.EncodeToString(claimsJSON)
	mac := hmac.New(sha256.New, []byte(h.cfg.APIKey))
	_, _ = mac.Write([]byte(payload))
	return payload + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

func (h *MarketAIHandler) verifyJobToken(token, userID, code string) (marketInsightJobClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return marketInsightJobClaims{}, errors.New("invalid job token")
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return marketInsightJobClaims{}, err
	}
	mac := hmac.New(sha256.New, []byte(h.cfg.APIKey))
	_, _ = mac.Write([]byte(parts[0]))
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return marketInsightJobClaims{}, errors.New("invalid job signature")
	}
	claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return marketInsightJobClaims{}, err
	}
	var claims marketInsightJobClaims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return marketInsightJobClaims{}, err
	}
	if claims.UserID != userID || claims.Code != code || claims.RequestID == "" || time.Now().Unix() > claims.ExpiresAt {
		return marketInsightJobClaims{}, errors.New("invalid job claims")
	}
	return claims, nil
}

func (h *MarketAIHandler) researchContext(ctx context.Context, primary domain.InstrumentPrice) []MarketDataPoint {
	now := time.Now().UTC()
	codes := relatedMarketCodes(primary)
	points := make([]MarketDataPoint, 0, len(codes))
	for _, code := range codes {
		price, ok := h.prices.Get(code)
		if !ok {
			continue
		}
		dayChange := price.ChangePct
		point := MarketDataPoint{
			Code: price.Code, DisplayName: price.DisplayName, Price: price.Price,
			Currency: price.Currency, Change1D: &dayChange,
			Timestamp: price.Timestamp.UTC().Format(time.RFC3339),
		}
		if h.candles != nil {
			candles, err := h.candles.Candles(ctx, code, "1d", now.AddDate(-1, 0, -7), now.Add(24*time.Hour), 400)
			if err == nil {
				point.Change1W = changeSince(candles, now.AddDate(0, 0, -7), price.Price)
				point.Change1M = changeSince(candles, now.AddDate(0, -1, 0), price.Price)
				point.Change3M = changeSince(candles, now.AddDate(0, -3, 0), price.Price)
				point.Change1Y = changeSince(candles, now.AddDate(-1, 0, 0), price.Price)
			}
		}
		points = append(points, point)
	}
	return points
}

func relatedMarketCodes(primary domain.InstrumentPrice) []string {
	candidates := []string{primary.Code}
	switch primary.Kind {
	case string(domain.KindGold):
		candidates = append(candidates, "ONS_GOLD", "USDTRY", "EURTRY", "GRAM_GOLD")
	case string(domain.KindCurrency):
		candidates = append(candidates, "USDTRY", "EURTRY", "ONS_GOLD", "GRAM_GOLD")
	case string(domain.KindCrypto):
		candidates = append(candidates, "BTC", "ETH", "USDTRY", "ONS_GOLD")
	case string(domain.KindStock):
		candidates = append(candidates, "USDTRY", "ONS_GOLD", "BTC")
	default:
		candidates = append(candidates, "USDTRY", "ONS_GOLD")
	}
	seen := make(map[string]bool, len(candidates))
	out := make([]string, 0, len(candidates))
	for _, code := range candidates {
		if code != "" && !seen[code] {
			seen[code] = true
			out = append(out, code)
		}
	}
	return out
}

func changeSince(candles []models.OHLC, target time.Time, latest float64) *float64 {
	var baseline float64
	for _, candle := range candles {
		if candle.Bucket.After(target) {
			break
		}
		if candle.Close > 0 {
			baseline = candle.Close
		}
	}
	if baseline <= 0 || latest <= 0 {
		return nil
	}
	change := (latest - baseline) / baseline * 100
	return &change
}

func extractMarketSources(envelope openAIResponsesEnvelope) []MarketSource {
	seen := make(map[string]bool)
	sources := make([]MarketSource, 0, 12)
	add := func(title, rawURL string) {
		rawURL = strings.TrimSpace(rawURL)
		parsed, err := url.Parse(rawURL)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || seen[rawURL] || len(sources) >= 12 {
			return
		}
		seen[rawURL] = true
		title = strings.TrimSpace(title)
		if title == "" {
			title = parsed.Hostname()
		}
		sources = append(sources, MarketSource{Title: title, URL: rawURL})
	}
	for _, item := range envelope.Output {
		for _, content := range item.Content {
			for _, annotation := range content.Annotations {
				if annotation.Type == "url_citation" {
					add(annotation.Title, annotation.URL)
				}
			}
		}
	}
	for _, item := range envelope.Output {
		for _, source := range item.Action.Sources {
			add(source.Title, source.URL)
		}
	}
	return sources
}

func (h *MarketAIHandler) cached(code string) (MarketInsightResponse, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	item, ok := h.cache[code]
	if !ok || time.Now().After(item.expiresAt) {
		delete(h.cache, code)
		return MarketInsightResponse{}, false
	}
	return item.value, true
}

func (h *MarketAIHandler) store(code string, value MarketInsightResponse) {
	h.mu.Lock()
	defer h.mu.Unlock()
	// Research is materially more expensive than a quote summary. Fifteen
	// minutes keeps sources fresh while preventing duplicate searches per pod.
	h.cache[code] = cachedMarketInsight{value: value, expiresAt: time.Now().Add(15 * time.Minute)}
}
