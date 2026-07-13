package provider

import (
	"context"
	"testing"

	"github.com/tsc-forex/tsc_forex/internal/config"
)

// TestStubProvider covers the placeholder adapter used for future real feeds.
func TestStubProvider(t *testing.T) {
	t.Parallel()
	p := newStub("twelvedata", config.APIProviderConfig{Enabled: false})
	if p.Name() != "twelvedata" {
		t.Fatalf("name = %q", p.Name())
	}
	if err := p.Connect(context.Background()); err != ErrNotConfigured {
		t.Fatalf("connect err = %v", err)
	}
	if err := p.Subscribe("XAUUSD"); err != ErrNotConfigured {
		t.Fatalf("subscribe err = %v", err)
	}
	ch := p.PriceChannel()
	if _, ok := <-ch; ok {
		t.Fatal("expected closed channel")
	}
	h := p.Health()
	if h.Status != "unhealthy" {
		t.Fatalf("status = %q", h.Status)
	}
	if err := p.Close(); err != nil {
		t.Fatal(err)
	}
}

// TestDefaultFactoryAllNames ensures every configured provider name builds.
func TestDefaultFactoryAllNames(t *testing.T) {
	t.Parallel()
	f := DefaultFactory(config.ProviderConfig{})
	for _, name := range []string{"mock", "twelvedata", "finnhub", "polygon", "alphavantage"} {
		if _, err := f(name); err != nil {
			t.Fatalf("factory(%q) err = %v", name, err)
		}
	}
}

// TestManagerOptions covers WithLogger/WithSymbols/Backoff.Attempt.
func TestManagerOptions(t *testing.T) {
	t.Parallel()
	cfg := config.ProviderConfig{
		Primary: "mock", ReconnectInitial: 1, ReconnectMax: 2,
		HealthInterval: 1, StaleThreshold: 1,
	}
	mgr, err := NewManager(cfg, DefaultFactory(cfg),
		WithLogger(nil), // nil ignored
		WithSymbols("XAUUSD"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(mgr.symbols) != 1 || mgr.symbols[0] != "XAUUSD" {
		t.Fatalf("symbols = %v", mgr.symbols)
	}

	b := Backoff{Initial: 1, Max: 2}
	if b.Attempt() != 0 {
		t.Fatal("attempt should start at 0")
	}
	b.Next()
	if b.Attempt() != 1 {
		t.Fatal("attempt should be 1")
	}

	if _, err := NewManager(cfg, nil); err == nil {
		t.Fatal("expected error for nil factory")
	}
}
