package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/tsc-forex/tsc_forex/internal/api"
	"github.com/tsc-forex/tsc_forex/internal/calculator"
	"github.com/tsc-forex/tsc_forex/internal/domain"
)

func TestGoldCurrentFromCalculator(t *testing.T) {
	gin.SetMode(gin.TestMode)

	calc := calculator.New()
	ticks := make(chan domain.Tick, 4)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go calc.Run(ctx, ticks)

	ticks <- domain.Tick{Symbol: domain.SymbolXAUUSD, Price: 2350, Timestamp: time.Now().UTC()}
	ticks <- domain.Tick{Symbol: domain.SymbolUSDTRY, Price: 34.5, Timestamp: time.Now().UTC()}

	deadline := time.After(time.Second)
	for {
		if _, ok := calc.Latest(); ok {
			break
		}
		select {
		case <-deadline:
			t.Fatal("calc not ready")
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}

	h := api.NewGoldHandler(calc, nil, nil, nil)
	r := gin.New()
	r.GET("/gold/current", h.Current)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/gold/current", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	var resp api.GoldResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	want := domain.CalculateGramGoldTRY(2350, 34.5)
	if resp.Price != want {
		t.Fatalf("price = %v want %v", resp.Price, want)
	}
	if resp.Source != "calculator" {
		t.Fatalf("source = %s", resp.Source)
	}
}

func TestGoldCurrentUnavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	calc := calculator.New() // never fed
	h := api.NewGoldHandler(calc, nil, nil, nil)
	r := gin.New()
	r.GET("/gold/current", h.Current)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/gold/current", nil))
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestHistoryWithoutRepo(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := api.NewGoldHandler(calculator.New(), nil, nil, nil)
	r := gin.New()
	r.GET("/gold/history", h.History)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/gold/history", nil))
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestVersionEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	sys := api.NewSystemHandler(api.SystemDeps{})
	r := gin.New()
	r.GET("/version", sys.Version)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/version", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["version"] == "" {
		t.Fatal("missing version")
	}
}
