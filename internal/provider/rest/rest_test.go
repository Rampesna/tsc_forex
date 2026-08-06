package rest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tsc-forex/tsc_forex/internal/config"
	"github.com/tsc-forex/tsc_forex/internal/domain"
)

func TestFetchDerivesTRYRates(t *testing.T) {
	fx := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"result":"success","rates":{"TRY":34.5,"EUR":0.92,"GBP":0.78,"JPY":150.0}}`))
	}))
	defer fx.Close()
	gold := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"XAUUSD":2350.5}`))
	}))
	defer gold.Close()

	p := New(config.RESTProviderConfig{FXBaseURL: fx.URL, GoldURL: gold.URL})
	ticks, err := p.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]float64{}
	for _, tk := range ticks {
		got[tk.Symbol] = tk.Price
	}
	if got[domain.SymbolUSDTRY] != 34.5 {
		t.Fatalf("USDTRY = %v", got[domain.SymbolUSDTRY])
	}
	// EURTRY = 34.5 / 0.92 ≈ 37.5
	if v := got[domain.SymbolEURTRY]; v < 37 || v > 38 {
		t.Fatalf("EURTRY = %v", v)
	}
	if got[domain.SymbolXAUUSD] != 2350.5 {
		t.Fatalf("XAUUSD = %v", got[domain.SymbolXAUUSD])
	}
}

func TestFetchFallsBackToBinanceGold(t *testing.T) {
	fx := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"result":"success","rates":{"TRY":34.5,"EUR":0.92,"GBP":0.78,"JPY":150.0}}`))
	}))
	defer fx.Close()
	// Broken gold URL forces the Binance PAXG fallback.
	gold := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer gold.Close()
	binance := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"symbol":"PAXGUSDT","price":"4250.25"}`))
	}))
	defer binance.Close()

	p := New(config.RESTProviderConfig{
		FXBaseURL:     fx.URL,
		GoldURL:       gold.URL,
		CryptoBaseURL: binance.URL,
	})
	ticks, err := p.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]float64{}
	for _, tk := range ticks {
		got[tk.Symbol] = tk.Price
	}
	if got[domain.SymbolXAUUSD] != 4250.25 {
		t.Fatalf("XAUUSD = %v, want 4250.25 from binance fallback", got[domain.SymbolXAUUSD])
	}
}

func TestFetchErrorOnBadStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	p := New(config.RESTProviderConfig{FXBaseURL: srv.URL})
	if _, err := p.Fetch(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}
