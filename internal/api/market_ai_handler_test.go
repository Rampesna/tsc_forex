package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"

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
	var providerIdempotency string
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("authorization = %q", got)
		}
		providerIdempotency = r.Header.Get("Idempotency-Key")
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
			"what_i_would_do": map[string]string{
				"intro":       "Yatırım tavsiyesi olmamakla beraber, ben olsam şöyle hareket ederdim.",
				"short_term":  "Kısa vadede bekler, veri akışını izlerdim.",
				"medium_term": "Orta vadede kademeli ve küçük adımlarla ilerlerdim.",
				"long_term":   "Uzun vadede temkinli birikim planımı korurdum.",
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
	if response.WhatIWouldDo.ShortTerm == "" || response.WhatIWouldDo.LongTerm == "" {
		t.Fatalf("missing what_i_would_do: %+v", response.WhatIWouldDo)
	}
	if response.ResearchMode != "web+market_data" || len(response.Sources) != 1 || len(response.DataContext) != 1 {
		t.Fatalf("missing research metadata: %+v", response)
	}
	var public map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &public); err != nil {
		t.Fatal(err)
	}
	if _, exposed := public["model"]; exposed {
		t.Fatalf("provider model leaked in public response: %s", w.Body.String())
	}
	if received["model"] != "test-model" || received["store"] != false {
		t.Fatalf("unexpected OpenAI payload: %+v", received)
	}
	if received["background"] != true {
		t.Fatalf("background mode missing: %+v", received)
	}
	if !strings.HasPrefix(providerIdempotency, "market-research-") || len(providerIdempotency) != len("market-research-")+64 {
		t.Fatalf("missing privacy-preserving provider idempotency key: %q", providerIdempotency)
	}
	if tools, _ := received["tools"].([]any); len(tools) != 1 {
		t.Fatalf("web search tool missing: %+v", received)
	}
	if input, _ := received["input"].(string); !strings.Contains(input, "change_1w_pct") {
		t.Fatalf("multi-timeframe context missing: %q", input)
	}
	if safetyID, _ := received["safety_identifier"].(string); !strings.HasPrefix(safetyID, "market_research_") {
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
		"what_i_would_do": map[string]string{
			"intro":       "Yatırım tavsiyesi olmamakla beraber, ben olsam şöyle hareket ederdim.",
			"short_term":  "Kısa vadede beklerdim.",
			"medium_term": "Orta vadede temkinli ilerlerdim.",
			"long_term":   "Uzun vadede planımı korurdum.",
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

func TestMarketInsightConcurrentSameRequestLaunchesOneProviderJob(t *testing.T) {
	gin.SetMode(gin.TestMode)
	started := make(chan struct{})
	release := make(chan struct{})
	var providerStarts atomic.Int32
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected provider method %s", r.Method)
		}
		if providerStarts.Add(1) == 1 {
			close(started)
		}
		<-release
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "provider-job-one", "status": "in_progress"})
	}))
	defer provider.Close()

	db, err := gorm.Open(sqlite.Open("file:api-ai-idempotency-"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&models.SubscriptionEntitlement{}, &models.AppleTransaction{}, &models.FeatureUsage{},
		&models.FreeResearchClaim{}, &models.MarketResearchJob{}, &models.AppAttestKey{},
		&models.AppAttestChallenge{}, &models.DeviceCheckRedemption{},
	); err != nil {
		t.Fatal(err)
	}
	userID := uuid.New()
	if err := db.Create(&models.SubscriptionEntitlement{
		UserID: userID, ProductID: "talhacan.tscforex.plus.monthly", PlanCode: "plus",
		OriginalTransactionID: "api-ai-original", LatestTransactionID: "api-ai-latest",
		Environment: "Production", Status: "active", PeriodStart: time.Now().UTC().Add(-time.Hour),
		ExpiresAt: time.Now().UTC().AddDate(0, 1, 0),
	}).Error; err != nil {
		t.Fatal(err)
	}
	billing := service.NewSubscriptionService(db, config.BillingConfig{
		Enabled: true, BundleID: "talhacan.tscforex",
	}, nil)
	reader := fixedPriceReader{price: domain.InstrumentPrice{
		Code: "USDTRY", DisplayName: "Amerikan Doları", Kind: "currency", Price: 47.25,
		Currency: "TRY", Timestamp: time.Now().UTC(),
	}}
	handler := api.NewMarketAIHandler(config.AIConfig{
		Enabled: true, APIKey: "test-key", Model: "test-model", BaseURL: provider.URL,
		Timeout: time.Second, MaxOutputTokens: 8000,
	}, reader).WithSubscriptions(billing)
	router := gin.New()
	router.POST("/insight", func(c *gin.Context) {
		c.Set(middleware.ClaimsKey, &service.Claims{UserID: userID.String()})
		handler.MarketInsight(c)
	})

	requestBody := `{"code":"USDTRY","request_id":"same-durable-request"}`
	first := httptest.NewRecorder()
	firstDone := make(chan struct{})
	go func() {
		defer close(firstDone)
		req := httptest.NewRequest(http.MethodPost, "/insight", strings.NewReader(requestBody))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(first, req)
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("provider start timeout")
	}

	second := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/insight", strings.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(second, req)
	if second.Code != http.StatusAccepted {
		t.Fatalf("duplicate status=%d body=%s", second.Code, second.Body.String())
	}
	close(release)
	<-firstDone
	if first.Code != http.StatusAccepted {
		t.Fatalf("creator status=%d body=%s", first.Code, first.Body.String())
	}
	if providerStarts.Load() != 1 {
		t.Fatalf("provider starts=%d want 1", providerStarts.Load())
	}
	var jobs, usage int64
	_ = db.Model(&models.MarketResearchJob{}).Count(&jobs).Error
	_ = db.Model(&models.FeatureUsage{}).Count(&usage).Error
	if jobs != 1 || usage != 1 {
		t.Fatalf("jobs=%d usage=%d", jobs, usage)
	}
}

func TestMarketInsightReclaimsStaleReservationWithSameProviderIdempotencyKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var providerStarts atomic.Int32
	var providerKey string
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		providerStarts.Add(1)
		providerKey = r.Header.Get("Idempotency-Key")
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "reclaimed-provider-job", "status": "in_progress"})
	}))
	defer provider.Close()

	db, err := gorm.Open(sqlite.Open("file:api-ai-stale-lease-"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&models.SubscriptionEntitlement{}, &models.AppleTransaction{}, &models.FeatureUsage{},
		&models.FreeResearchClaim{}, &models.MarketResearchJob{}, &models.AppAttestKey{},
		&models.AppAttestChallenge{}, &models.DeviceCheckRedemption{},
	); err != nil {
		t.Fatal(err)
	}
	userID := uuid.New()
	if err := db.Create(&models.SubscriptionEntitlement{
		UserID: userID, ProductID: "talhacan.tscforex.plus.monthly", PlanCode: "plus",
		OriginalTransactionID: "api-ai-stale-original", LatestTransactionID: "api-ai-stale-latest",
		Environment: "Production", Status: "active", PeriodStart: time.Now().UTC().Add(-time.Hour),
		ExpiresAt: time.Now().UTC().AddDate(0, 1, 0),
	}).Error; err != nil {
		t.Fatal(err)
	}
	billing := service.NewSubscriptionService(db, config.BillingConfig{Enabled: true, BundleID: "talhacan.tscforex"}, nil)
	requestID := "same-request-after-worker-crash"
	if job, err := billing.BeginAIResearchJob(context.Background(), userID, "paid-device", requestID, "USDTRY", service.AppAttestProof{}); err != nil || !job.Created {
		t.Fatalf("seed reservation=%+v err=%v", job, err)
	}
	if err := db.Model(&models.MarketResearchJob{}).
		Where("user_id = ? AND request_key = ?", userID, requestID).
		Update("lease_expires_at", time.Now().UTC().Add(-time.Second)).Error; err != nil {
		t.Fatal(err)
	}

	reader := fixedPriceReader{price: domain.InstrumentPrice{
		Code: "USDTRY", DisplayName: "Amerikan Doları", Kind: "currency", Price: 47.25,
		Currency: "TRY", Timestamp: time.Now().UTC(),
	}}
	handler := api.NewMarketAIHandler(config.AIConfig{
		Enabled: true, APIKey: "test-key", Model: "test-model", BaseURL: provider.URL,
		Timeout: time.Second, MaxOutputTokens: 8000,
	}, reader).WithSubscriptions(billing)
	router := gin.New()
	router.POST("/insight", func(c *gin.Context) {
		c.Set(middleware.ClaimsKey, &service.Claims{UserID: userID.String()})
		handler.MarketInsight(c)
	})

	w := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]string{"code": "USDTRY", "request_id": requestID})
	req := httptest.NewRequest(http.MethodPost, "/insight", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusAccepted {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if providerStarts.Load() != 1 || !strings.HasPrefix(providerKey, "market-research-") {
		t.Fatalf("provider starts=%d key=%q", providerStarts.Load(), providerKey)
	}
	var pending map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &pending); err != nil {
		t.Fatal(err)
	}
	if _, exposed := pending["model"]; exposed {
		t.Fatalf("provider model leaked in pending response: %s", w.Body.String())
	}
	var usage int64
	if err := db.Model(&models.FeatureUsage{}).
		Where("user_id = ? AND feature = ? AND request_key = ?", userID, service.FeatureAIResearch, requestID).
		Count(&usage).Error; err != nil || usage != 1 {
		t.Fatalf("usage=%d err=%v", usage, err)
	}
	var job models.MarketResearchJob
	if err := db.Where("user_id = ? AND request_key = ?", userID, requestID).First(&job).Error; err != nil ||
		job.AttemptCount != 2 || job.Status != service.ResearchJobPending || job.ProviderResponseID != "reclaimed-provider-job" {
		t.Fatalf("job=%+v err=%v", job, err)
	}
}
