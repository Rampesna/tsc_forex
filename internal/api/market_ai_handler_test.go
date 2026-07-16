package api_test

import (
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
	"github.com/tsc-forex/tsc_forex/internal/service"
)

type fixedPriceReader struct {
	price domain.InstrumentPrice
}

func (r fixedPriceReader) Get(code string) (domain.InstrumentPrice, bool) {
	return r.price, code == r.price.Code
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
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"output":[{"type":"message","content":[{"type":"output_text","text":"{\"title\":\"Dengeli görünüm\",\"stance\":\"neutral\",\"stance_label\":\"Dengeli\",\"summary\":\"Gün içi hareket sınırlı kalırken fiyat yön arıyor.\",\"factors\":[\"Değişim sınırlı\",\"Spread dar\",\"Canlı fiyat güncel\"],\"risk_note\":\"Kısa vadeli oynaklığa dikkat.\"}"}]}]}`))
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
		BaseURL: provider.URL, Timeout: time.Second, MaxOutputTokens: 300,
	}, reader)
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
	if received["model"] != "test-model" || received["store"] != false {
		t.Fatalf("unexpected OpenAI payload: %+v", received)
	}
	if safetyID, _ := received["safety_identifier"].(string); !strings.HasPrefix(safetyID, "tsc_") {
		t.Fatalf("missing hashed safety identifier: %q", safetyID)
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
