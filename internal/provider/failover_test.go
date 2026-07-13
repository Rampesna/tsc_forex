package provider_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tsc-forex/tsc_forex/internal/config"
	"github.com/tsc-forex/tsc_forex/internal/domain"
	"github.com/tsc-forex/tsc_forex/internal/provider"
	"github.com/tsc-forex/tsc_forex/internal/provider/mock"
)

// TestManagerFactoryErrorFailover exercises the factory-error branch: the
// primary name is unknown to the factory, so the manager advances to a working
// fallback and starts streaming.
func TestManagerFactoryErrorFailover(t *testing.T) {
	cfg := config.ProviderConfig{
		Primary:          "broken",
		Fallbacks:        []string{"good"},
		ReconnectInitial: 5 * time.Millisecond,
		ReconnectMax:     10 * time.Millisecond,
		HealthInterval:   time.Second,
		StaleThreshold:   time.Hour,
	}

	var factoryCalls atomic.Int64
	factory := provider.Factory(func(name string) (provider.MarketProvider, error) {
		factoryCalls.Add(1)
		switch name {
		case "good":
			p := mock.New(config.MockProviderConfig{TickInterval: 5 * time.Millisecond})
			return namedProvider{MarketProvider: p, name: "good"}, nil
		default:
			return nil, provider.ErrNotConfigured
		}
	})

	mgr, err := provider.NewManager(cfg, factory)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := mgr.Run(ctx); err != nil {
		t.Fatal(err)
	}

	deadline := time.After(3 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatalf("no failover to good; active=%q", mgr.ActiveName())
		case tick := <-mgr.PriceChannel():
			if tick.Symbol == domain.SymbolXAUUSD || tick.Symbol == domain.SymbolUSDTRY {
				if mgr.ActiveName() == "good" {
					cancel()
					_ = mgr.Close()
					return
				}
			}
		}
	}
}

// TestManagerActivateFailureRetries exercises the activate-failure branch where
// Connect returns an error repeatedly, triggering reconnect then failover.
func TestManagerActivateFailureRetries(t *testing.T) {
	cfg := config.ProviderConfig{
		Primary:          "always_fail",
		Fallbacks:        []string{"good"},
		ReconnectInitial: 2 * time.Millisecond,
		ReconnectMax:     5 * time.Millisecond,
		HealthInterval:   time.Second,
		StaleThreshold:   time.Hour,
	}
	factory := provider.Factory(func(name string) (provider.MarketProvider, error) {
		switch name {
		case "always_fail":
			return failingProvider{}, nil
		default:
			p := mock.New(config.MockProviderConfig{TickInterval: 5 * time.Millisecond})
			return namedProvider{MarketProvider: p, name: "good"}, nil
		}
	})

	mgr, err := provider.NewManager(cfg, factory)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_ = mgr.Run(ctx)

	deadline := time.After(3 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatalf("no recovery; active=%q reconnects=%d", mgr.ActiveName(), mgr.ReconnectCount())
		case <-mgr.PriceChannel():
			if mgr.ActiveName() == "good" {
				if mgr.ReconnectCount() == 0 {
					t.Fatal("expected reconnect attempts")
				}
				cancel()
				_ = mgr.Close()
				return
			}
		}
	}
}

// failingProvider always fails to connect.
type failingProvider struct{}

func (failingProvider) Name() string                  { return "always_fail" }
func (failingProvider) Connect(context.Context) error { return provider.ErrNotConfigured }
func (failingProvider) Subscribe(string) error        { return provider.ErrNotConfigured }
func (failingProvider) PriceChannel() <-chan domain.Tick {
	ch := make(chan domain.Tick)
	close(ch)
	return ch
}
func (failingProvider) Health() domain.ProviderHealth {
	return domain.ProviderHealth{Name: "always_fail", Status: domain.ProviderUnhealthy}
}
func (failingProvider) Close() error { return nil }
