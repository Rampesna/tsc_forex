package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/tsc-forex/tsc_forex/internal/api"
	"github.com/tsc-forex/tsc_forex/internal/config"
	"github.com/tsc-forex/tsc_forex/internal/domain"
	"github.com/tsc-forex/tsc_forex/internal/middleware"
	"github.com/tsc-forex/tsc_forex/internal/models"
	"github.com/tsc-forex/tsc_forex/internal/service"
)

type fixedPriceReader struct {
	price domain.InstrumentPrice
}

func (r fixedPriceReader) Get(code string) (domain.InstrumentPrice, bool) {
	return r.price, code == r.price.Code
}

type fixedCandleReader struct{}

func (fixedCandleReader) Candles(_ context.Context, _ string, _ string, _, _ time.Time, _ int) ([]models.OHLC, error) {
	now := time.Now().UTC()
	return []models.OHLC{
		{Bucket: now.AddDate(-1, 0, -1), Close: 35},
		{Bucket: now.AddDate(0, -3, -1), Close: 42},
		{Bucket: now.AddDate(0, -1, -1), Close: 45},
		{Bucket: now.AddDate(0, 0, -8), Close: 46},
	}, nil
}

func TestMarketInsightUsesServerSideStructuredResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var received map[string]any
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("authorization = %q", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatal(err)
		}
		modelOutput, _ := json.Marshal(map[string]any{
			"title": "Dengeli görünüm", "stance": "neutral", "stance_label": "Dengeli",
			"summary":   "Çok zaman dilimli veri ve güncel araştırma dengeli bir görünüm veriyor.",
			"factors":   []string{"Haftalık eğilim", "Aylık eğilim", "Döviz aktarımı", "Makro görünüm"},
			"risk_note": "Jeopolitik haber akışı oynaklık yaratabilir.", "confidence": "medium",
			"time_horizons": []map[string]string{
				{"horizon": "1 hafta", "outlook": "Dengeli", "rationale": "Fiyat sıkışıyor."},
				{"horizon": "1-3 ay", "outlook": "Temkinli", "rationale": "Faiz patikası belirleyici."},
				{"horizon": "6-12 ay", "outlook": "Belirsiz", "rationale": "Makro rejim değişebilir."},
			},
			"macro_drivers":        []string{"Faiz", "Enflasyon", "Dolar"},
			"geopolitical_risks":   []string{"Bölgesel gerilim", "Ticaret sürtüşmesi"},
			"cross_asset_analysis": []string{"USD/TRY etkisi", "Ons altın etkisi", "Risk iştahı"},
			"scenarios": []map[string]string{
				{"name": "Boğa", "likelihood": "medium", "trigger": "Risk artışı", "implication": "Yukarı baskı"},
				{"name": "Ana", "likelihood": "high", "trigger": "Yatay makro", "implication": "Dalgalı seyir"},
				{"name": "Ayı", "likelihood": "low", "trigger": "Reel faiz artışı", "implication": "Aşağı baskı"},
			},
		})
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"output": []any{
			map[string]any{"type": "web_search_call", "action": map[string]any{"sources": []any{
				map[string]any{"type": "url", "title": "Central bank", "url": "https://example.com/source"},
			}}},
			map[string]any{"type": "message", "content": []any{map[string]any{
				"type": "output_text", "text": string(modelOutput),
				"annotations": []any{map[string]any{"type": "url_citation", "title": "Central bank", "url": "https://example.com/source"}},
			}}},
		}})
	}))
	defer provider.Close()

	reader := fixedPriceReader{price: domain.InstrumentPrice{
		Code: "USDTRY", DisplayName: "Amerikan Doları", Kind: "currency",
		Price: 47.25, PriceTRY: 47.25, Bid: 47.24, Ask: 47.26,
		Change: 0.12, ChangePct: 0.25, Currency: "TRY", Provider: "test",
		Timestamp: time.Now().UTC(),
	}}
	handler := api.NewMarketAIHandler(config.AIConfig{
		Enabled: true, APIKey: "test-key", Model: "test-model",
		BaseURL: provider.URL, Timeout: time.Second, MaxOutputTokens: 2600,
		ReasoningEffort: "high", WebSearch: true, SearchContext: "high", MaxToolCalls: 4,
	}, reader, fixedCandleReader{})
	router := gin.New()
	router.POST("/insight", func(c *gin.Context) {
		c.Set(middleware.ClaimsKey, &service.Claims{UserID: uuid.NewString()})
		handler.MarketInsight(c)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/insight", strings.NewReader(`{"code":"usdtry"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
	}
	var response api.MarketInsightResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Code != "USDTRY" || response.Stance != "neutral" || response.Disclaimer == "" {
		t.Fatalf("unexpected response: %+v", response)
	}
	if response.Model != "test-model" || response.ResearchMode != "web+market_data" || len(response.Sources) != 1 || len(response.DataContext) != 1 {
		t.Fatalf("missing research metadata: %+v", response)
	}
	if received["model"] != "test-model" || received["store"] != false {
		t.Fatalf("unexpected OpenAI payload: %+v", received)
	}
	if received["background"] != true {
		t.Fatalf("background mode missing: %+v", received)
	}
	if tools, _ := received["tools"].([]any); len(tools) != 1 {
		t.Fatalf("web search tool missing: %+v", received)
	}
	if input, _ := received["input"].(string); !strings.Contains(input, "change_1w_pct") {
		t.Fatalf("multi-timeframe context missing: %q", input)
	}
	if safetyID, _ := received["safety_identifier"].(string); !strings.HasPrefix(safetyID, "tsc_") {
		t.Fatalf("missing hashed safety identifier: %q", safetyID)
	}
}

func TestMarketInsightPollsSignedBackgroundJob(t *testing.T) {
	gin.SetMode(gin.TestMode)
	modelOutput, _ := json.Marshal(map[string]any{
		"title": "Kaynaklı görünüm", "stance": "neutral", "stance_label": "Dengeli",
		"summary": "Araştırma tamamlandı.", "factors": []string{"A", "B", "C", "D"},
		"risk_note": "Belirsizlik sürüyor.", "confidence": "medium",
		"time_horizons": []map[string]string{
			{"horizon": "1 hafta", "outlook": "Dengeli", "rationale": "A"},
			{"horizon": "1-3 ay", "outlook": "Dengeli", "rationale": "B"},
			{"horizon": "6-12 ay", "outlook": "Dengeli", "rationale": "C"},
		},
		"macro_drivers": []string{"A", "B", "C"}, "geopolitical_risks": []string{"A", "B"},
		"cross_asset_analysis": []string{"A", "B", "C"},
		"scenarios": []map[string]string{
			{"name": "Boğa", "likelihood": "low", "trigger": "A", "implication": "A"},
			{"name": "Ana", "likelihood": "high", "trigger": "B", "implication": "B"},
			{"name": "Ayı", "likelihood": "low", "trigger": "C", "implication": "C"},
		},
	})
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "resp_test", "status": "in_progress"})
			return
		}
		if r.Method != http.MethodGet || r.URL.Path != "/responses/resp_test" {
			t.Fatalf("unexpected poll request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "resp_test", "status": "completed",
			"output": []any{map[string]any{
				"type": "message",
				"content": []any{map[string]any{
					"type": "output_text", "text": string(modelOutput),
				}},
			}},
		})
	}))
	defer provider.Close()

	reader := fixedPriceReader{price: domain.InstrumentPrice{
		Code: "USDTRY", DisplayName: "Amerikan Doları", Kind: "currency", Price: 47.25,
		Currency: "TRY", Timestamp: time.Now().UTC(),
	}}
	handler := api.NewMarketAIHandler(config.AIConfig{
		Enabled: true, APIKey: "test-key", Model: "test-model", BaseURL: provider.URL,
		Timeout: time.Second, MaxOutputTokens: 8000, ReasoningEffort: "high", WebSearch: true,
	}, reader)
	userID := uuid.NewString()
	router := gin.New()
	router.POST("/insight", func(c *gin.Context) {
		c.Set(middleware.ClaimsKey, &service.Claims{UserID: userID})
		handler.MarketInsight(c)
	})

	first := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/insight", strings.NewReader(`{"code":"USDTRY"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(first, req)
	if first.Code != http.StatusAccepted {
		t.Fatalf("start status = %d body = %s", first.Code, first.Body.String())
	}
	var pending struct {
		JobID string `json:"job_id"`
	}
	if err := json.Unmarshal(first.Body.Bytes(), &pending); err != nil || pending.JobID == "" {
		t.Fatalf("missing signed job: %v body=%s", err, first.Body.String())
	}

	second := httptest.NewRecorder()
	pollBody, _ := json.Marshal(map[string]string{"code": "USDTRY", "job_id": pending.JobID})
	req = httptest.NewRequest(http.MethodPost, "/insight", strings.NewReader(string(pollBody)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(second, req)
	if second.Code != http.StatusOK {
		t.Fatalf("poll status = %d body = %s", second.Code, second.Body.String())
	}
	var insight api.MarketInsightResponse
	if err := json.Unmarshal(second.Body.Bytes(), &insight); err != nil || insight.Title != "Kaynaklı görünüm" {
		t.Fatalf("unexpected completed insight: %v %+v", err, insight)
	}
}

func TestMarketInsightRequiresConfiguredServerKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := api.NewMarketAIHandler(config.AIConfig{}, fixedPriceReader{})
	router := gin.New()
	router.POST("/insight", func(c *gin.Context) {
		c.Set(middleware.ClaimsKey, &service.Claims{UserID: uuid.NewString()})
		handler.MarketInsight(c)
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/insight", strings.NewReader(`{"code":"USDTRY"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
	}
}
