package binance

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tsc-forex/tsc_forex/internal/config"
	"github.com/tsc-forex/tsc_forex/internal/domain"
)

func TestFetchDerivesLiveLegs(t *testing.T) {
	binanceSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"symbol":"PAXGUSDT","price":"4050.00"},{"symbol":"USDTTRY","price":"47.00"}]`))
	}))
	defer binanceSrv.Close()

	fxSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"result":"success","rates":{"EUR":0.9,"GBP":0.8,"JPY":160.0}}`))
	}))
	defer fxSrv.Close()

	orig := binanceBatchURL
	binanceBatchURL = binanceSrv.URL
	defer func() { binanceBatchURL = orig }()

	p := New(config.RESTProviderConfig{FXBaseURL: fxSrv.URL})
	if err := p.refreshFX(context.Background()); err != nil {
		t.Fatalf("refreshFX: %v", err)
	}

	ticks, err := p.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	got := make(map[string]float64, len(ticks))
	for _, tk := range ticks {
		got[tk.Symbol] = tk.Price
	}

	assertClose(t, "XAUUSD", got[domain.SymbolXAUUSD], 4050.0)
	assertClose(t, "USDTRY", got[domain.SymbolUSDTRY], 47.0)
	assertClose(t, "EURTRY", got[domain.SymbolEURTRY], 47.0/0.9)
	assertClose(t, "GBPTRY", got[domain.SymbolGBPTRY], 47.0/0.8)
	assertClose(t, "JPYTRY", got[domain.SymbolJPYTRY], 47.0/160.0)
}

func TestFetchErrorsOnBadStatus(t *testing.T) {
	binanceSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer binanceSrv.Close()

	orig := binanceBatchURL
	binanceBatchURL = binanceSrv.URL
	defer func() { binanceBatchURL = orig }()

	p := New(config.RESTProviderConfig{FXBaseURL: "http://127.0.0.1:0"})
	if _, err := p.Fetch(context.Background()); err == nil {
		t.Fatal("expected error on HTTP 429")
	}
}

func assertClose(t *testing.T, name string, got, want float64) {
	t.Helper()
	diff := got - want
	if diff < 0 {
		diff = -diff
	}
	if diff > 1e-6 {
		t.Errorf("%s = %v, want %v", name, got, want)
	}
}
