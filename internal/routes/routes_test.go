package routes_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/tsc-forex/tsc_forex/internal/api"
	"github.com/tsc-forex/tsc_forex/internal/calculator"
	"github.com/tsc-forex/tsc_forex/internal/config"
	"github.com/tsc-forex/tsc_forex/internal/domain"
	"github.com/tsc-forex/tsc_forex/internal/metrics"
	"github.com/tsc-forex/tsc_forex/internal/provider"
	"github.com/tsc-forex/tsc_forex/internal/routes"
	ws "github.com/tsc-forex/tsc_forex/internal/websocket"
)

func buildEngine(t *testing.T) (http.Handler, *calculator.Calculator) {
	return buildEngineWithReadiness(t, func(context.Context) error { return nil })
}

func buildEngineWithReadiness(t *testing.T, ready func(context.Context) error) (http.Handler, *calculator.Calculator) {
	t.Helper()

	cfg := &config.Config{}
	cfg.Server.Mode = "test"
	cfg.WebSocket = config.WebSocketConfig{
		Path: "/ws", WriteWait: time.Second, PongWait: time.Second,
		PingPeriod: time.Second, MaxMessageSize: 4096, SendBufferSize: 8,
	}
	cfg.CORS.AllowedOrigins = []string{"*"}
	cfg.Metrics = config.MetricsConfig{Enabled: true, Path: "/metrics"}
	cfg.RateLimit = config.RateLimitConfig{Enabled: true, Requests: 1000, WindowSec: 60}

	pmCfg := config.ProviderConfig{
		Primary: "mock", ReconnectInitial: 10 * time.Millisecond,
		ReconnectMax: 20 * time.Millisecond, HealthInterval: time.Second,
		StaleThreshold: time.Hour,
		Mock:           config.MockProviderConfig{TickInterval: 10 * time.Millisecond},
	}
	pm, err := provider.NewManager(pmCfg, provider.DefaultFactory(pmCfg))
	if err != nil {
		t.Fatal(err)
	}

	calc := calculator.New()
	ticks := make(chan domain.Tick, 4)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go calc.Run(ctx, ticks)
	ticks <- domain.Tick{Symbol: domain.SymbolXAUUSD, Price: 2350, Timestamp: time.Now().UTC()}
	ticks <- domain.Tick{Symbol: domain.SymbolUSDTRY, Price: 34.5, Timestamp: time.Now().UTC()}
	waitReady(t, calc)

	hub := ws.NewHub(cfg.WebSocket, nil)
	go hub.Run()
	t.Cleanup(hub.Close)

	m := metrics.New("test")

	engine := routes.NewEngine(routes.Deps{
		Config:  cfg,
		Hub:     hub,
		WS:      ws.NewHandler(hub, nil),
		Metrics: m,
		Gold:    api.NewGoldHandler(calc, nil, nil, nil),
		System: api.NewSystemHandler(api.SystemDeps{
			Provider: pm, Calc: calc, ClientsFn: hub.ClientCount,
		}),
		Ready: ready,
	})
	return engine, calc
}

func waitReady(t *testing.T, calc *calculator.Calculator) {
	deadline := time.After(time.Second)
	for {
		if _, ok := calc.Latest(); ok {
			return
		}
		select {
		case <-deadline:
			t.Fatal("calc not ready")
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}
}

func TestEngineEndpoints(t *testing.T) {
	engine, _ := buildEngine(t)

	cases := []struct {
		path string
		code int
	}{
		{"/api/v1/health", http.StatusOK},
		{"/api/v1/version", http.StatusOK},
		{"/api/v1/providers", http.StatusOK},
		{"/api/v1/gold/current", http.StatusOK},
		{"/metrics", http.StatusOK},
		{"/livez", http.StatusOK},
		{"/readyz", http.StatusOK},
		{"/docs", http.StatusOK},
		{"/docs/openapi.yaml", http.StatusOK},
	}
	for _, tc := range cases {
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, tc.path, nil))
		if w.Code != tc.code {
			t.Fatalf("%s → %d, want %d", tc.path, w.Code, tc.code)
		}
	}
}

func TestReadinessProbeFailsClosed(t *testing.T) {
	engine, _ := buildEngineWithReadiness(t, func(context.Context) error {
		return errors.New("sensitive database detail")
	})
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("readyz status=%d body=%s", w.Code, w.Body.String())
	}
	if w.Header().Get("Retry-After") != "5" {
		t.Fatalf("retry-after=%q", w.Header().Get("Retry-After"))
	}
	if strings.Contains(w.Body.String(), "sensitive database detail") {
		t.Fatalf("readiness leaked internal error: %s", w.Body.String())
	}
}

func TestEngineGoldCurrentBody(t *testing.T) {
	engine, _ := buildEngine(t)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/gold/current", nil))

	var resp api.GoldResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Symbol != domain.SymbolGramGoldTRY || resp.Price <= 0 {
		t.Fatalf("resp = %+v", resp)
	}
}

func TestEngineHistoryUnavailable(t *testing.T) {
	engine, _ := buildEngine(t)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/gold/history", nil))
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("history → %d", w.Code)
	}
}
