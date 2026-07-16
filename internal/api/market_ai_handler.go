package api

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/tsc-forex/tsc_forex/internal/config"
	"github.com/tsc-forex/tsc_forex/internal/domain"
)

const marketInsightDisclaimer = "Bu içerik yatırım tavsiyesi değildir; eğitim ve karar desteği amaçlıdır."

// MarketPriceReader supplies the latest trusted server-side market snapshot.
type MarketPriceReader interface {
	Get(code string) (domain.InstrumentPrice, bool)
}

// MarketAIHandler generates short, structured market briefs without exposing
// provider credentials to browser or mobile clients.
type MarketAIHandler struct {
	cfg    config.AIConfig
	prices MarketPriceReader
	client *http.Client
	mu     sync.Mutex
	cache  map[string]cachedMarketInsight
}

type cachedMarketInsight struct {
	value     MarketInsightResponse
	expiresAt time.Time
}

type marketInsightRequest struct {
	Code string `json:"code" binding:"required"`
}

// MarketInsightResponse is intentionally scenario-oriented, not a buy/sell
// signal. The client can render it without parsing untrusted markdown.
type MarketInsightResponse struct {
	Code        string   `json:"code"`
	Title       string   `json:"title"`
	Stance      string   `json:"stance"`
	StanceLabel string   `json:"stance_label"`
	Summary     string   `json:"summary"`
	Factors     []string `json:"factors"`
	RiskNote    string   `json:"risk_note"`
	GeneratedAt string   `json:"generated_at"`
	Disclaimer  string   `json:"disclaimer"`
}

type openAIResponsesPayload struct {
	Model           string         `json:"model"`
	Instructions    string         `json:"instructions"`
	Input           string         `json:"input"`
	MaxOutputTokens int            `json:"max_output_tokens"`
	SafetyID        string         `json:"safety_identifier,omitempty"`
	Store           bool           `json:"store"`
	Text            openAITextSpec `json:"text"`
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
	Output []struct {
		Type    string `json:"type"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"output"`
}

func NewMarketAIHandler(cfg config.AIConfig, prices MarketPriceReader) *MarketAIHandler {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 8 * time.Second
	}
	return &MarketAIHandler{
		cfg: cfg, prices: prices,
		client: &http.Client{Timeout: timeout},
		cache:  make(map[string]cachedMarketInsight),
	}
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
	if cached, found := h.cached(req.Code); found {
		c.JSON(http.StatusOK, cached)
		return
	}

	digest := sha256.Sum256([]byte(uid.String()))
	result, err := h.generate(c.Request.Context(), price, "tsc_"+hex.EncodeToString(digest[:12]))
	if err != nil {
		c.JSON(http.StatusBadGateway, ErrorResponse{Error: "AI yorumu şu anda alınamadı"})
		return
	}
	h.store(req.Code, result)
	c.JSON(http.StatusOK, result)
}

func (h *MarketAIHandler) generate(ctx context.Context, p domain.InstrumentPrice, safetyID string) (MarketInsightResponse, error) {
	model := strings.TrimSpace(h.cfg.Model)
	if model == "" {
		model = "gpt-5.6-luna"
	}
	maxTokens := h.cfg.MaxOutputTokens
	if maxTokens <= 0 {
		maxTokens = 500
	}
	baseURL := strings.TrimRight(strings.TrimSpace(h.cfg.BaseURL), "/")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	prompt := fmt.Sprintf(
		"Varlık: %s (%s)\nTür: %s\nSon fiyat: %.8f %s\nTRY karşılığı: %.4f\nGünlük değişim: %.4f%%\nGünlük mutlak değişim: %.8f\nAlış: %.8f\nSatış: %.8f\nVeri sağlayıcı: %s\nVeri zamanı: %s",
		p.DisplayName, p.Code, p.Kind, p.Price, p.Currency, p.PriceTRY, p.ChangePct, p.Change, p.Bid, p.Ask, p.Provider, p.Timestamp.UTC().Format(time.RFC3339),
	)
	payload := openAIResponsesPayload{
		Model:        model,
		Instructions: "Türkçe yazan bir piyasa araştırma asistanısın. Yalnızca verilen canlı fiyat verisini kullan; haber, temel veri veya gelecek fiyatı uydurma. Al/sat emri verme ve kesinlik iddia etme. Kısa, sakin ve profesyonel bir senaryo özeti üret. Yön değerlendirmesi yalnızca positive, neutral veya cautious olabilir. factors alanındaki her madde en fazla 8 kelime olsun.",
		Input:        prompt, MaxOutputTokens: maxTokens, SafetyID: safetyID, Store: false,
		Text: openAITextSpec{Format: openAIFormatSpec{
			Type: "json_schema", Name: "tsc_market_insight", Strict: true,
			Schema: map[string]any{
				"type": "object", "additionalProperties": false,
				"properties": map[string]any{
					"title":        map[string]any{"type": "string"},
					"stance":       map[string]any{"type": "string", "enum": []string{"positive", "neutral", "cautious"}},
					"stance_label": map[string]any{"type": "string"},
					"summary":      map[string]any{"type": "string"},
					"factors":      map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "minItems": 3, "maxItems": 3},
					"risk_note":    map[string]any{"type": "string"},
				},
				"required": []string{"title", "stance", "stance_label", "summary", "factors", "risk_note"},
			},
		}},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return MarketInsightResponse{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/responses", bytes.NewReader(body))
	if err != nil {
		return MarketInsightResponse{}, err
	}
	req.Header.Set("Authorization", "Bearer "+h.cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")
	res, err := h.client.Do(req)
	if err != nil {
		return MarketInsightResponse{}, err
	}
	defer res.Body.Close()
	limited := io.LimitReader(res.Body, 1<<20)
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, limited)
		return MarketInsightResponse{}, fmt.Errorf("openai status %d", res.StatusCode)
	}
	var envelope openAIResponsesEnvelope
	if err := json.NewDecoder(limited).Decode(&envelope); err != nil {
		return MarketInsightResponse{}, err
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
	insight.Code = p.Code
	insight.GeneratedAt = time.Now().UTC().Format(time.RFC3339)
	insight.Disclaimer = marketInsightDisclaimer
	return insight, nil
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
	h.cache[code] = cachedMarketInsight{value: value, expiresAt: time.Now().Add(2 * time.Minute)}
}
