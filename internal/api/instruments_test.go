package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/tsc-forex/tsc_forex/internal/api"
	"github.com/tsc-forex/tsc_forex/internal/calculator"
	"github.com/tsc-forex/tsc_forex/internal/domain"
	"github.com/tsc-forex/tsc_forex/pkg/eventbus"
)

func TestInstrumentCatalogEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := api.NewInstrumentHandler(nil, nil, nil, "", nil)
	r := gin.New()
	r.GET("/instruments", h.Catalog)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/instruments", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	var body struct {
		Count int `json:"count"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body.Count != len(domain.Catalog) {
		t.Fatalf("count = %d want %d", body.Count, len(domain.Catalog))
	}
}

func TestInstrumentPricesEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	bus := eventbus.New()
	defer bus.Close()
	engine := calculator.NewEngine(bus, nil)
	engine.Attach()

	// Feed enough legs so gold + USDTRY compute.
	bus.Publish(eventbus.TopicTickReceived, domain.Tick{Symbol: domain.SymbolXAUUSD, Price: 2350})
	bus.Publish(eventbus.TopicTickReceived, domain.Tick{Symbol: domain.SymbolUSDTRY, Price: 34.5})

	h := api.NewInstrumentHandler(engine, nil, nil, "", nil)
	r := gin.New()
	r.GET("/instruments/prices", h.Prices)
	r.GET("/instruments/:code", h.Get)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/instruments/prices", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("prices status = %d", w.Code)
	}

	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest(http.MethodGet, "/instruments/GRAM_GOLD", nil))
	if w2.Code != http.StatusOK {
		t.Fatalf("get status = %d", w2.Code)
	}

	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, httptest.NewRequest(http.MethodGet, "/instruments/NOPE", nil))
	if w3.Code != http.StatusNotFound {
		t.Fatalf("unknown code status = %d", w3.Code)
	}
}
