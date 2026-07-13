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
	"github.com/tsc-forex/tsc_forex/pkg/eventbus"
)

func testProviderCfg() config.ProviderConfig {
	return config.ProviderConfig{
		Primary:          "mock",
		Fallbacks:        nil,
		ReconnectInitial: 20 * time.Millisecond,
		ReconnectMax:     50 * time.Millisecond,
		HealthInterval:   100 * time.Millisecond,
		StaleThreshold:   time.Hour,
		Mock: config.MockProviderConfig{
			Enabled:       true,
			TickInterval:  15 * time.Millisecond,
			XAUUSDBase:    2100,
			USDTRYBase:    32,
			JitterPercent: 0.01,
		},
	}
}

func TestManagerDeliversTicksAndBusEvents(t *testing.T) {
	cfg := testProviderCfg()
	bus := eventbus.New()
	defer bus.Close()

	var busTicks atomic.Int64
	bus.Subscribe(eventbus.TopicTickReceived, func(ev eventbus.Event) {
		if _, ok := ev.Payload.(domain.Tick); ok {
			busTicks.Add(1)
		}
	})

	mgr, err := provider.NewManager(
		cfg,
		provider.DefaultFactory(cfg),
		provider.WithBus(bus),
	)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := mgr.Run(ctx); err != nil {
		t.Fatal(err)
	}

	got := 0
	deadline := time.After(3 * time.Second)
	for got < 4 {
		select {
		case tick, ok := <-mgr.PriceChannel():
			if !ok {
				t.Fatal("manager channel closed")
			}
			if tick.Symbol != domain.SymbolXAUUSD && tick.Symbol != domain.SymbolUSDTRY {
				t.Fatalf("unexpected symbol %q", tick.Symbol)
			}
			got++
		case <-deadline:
			t.Fatalf("timeout, got %d ticks", got)
		}
	}

	if mgr.ActiveName() != "mock" {
		t.Fatalf("active = %q", mgr.ActiveName())
	}
	if busTicks.Load() == 0 {
		t.Fatal("expected bus tick events")
	}

	cancel()
	_ = mgr.Close()
}

func TestManagerFailoverOnStreamEnd(t *testing.T) {
	cfg := testProviderCfg()
	cfg.Primary = "primary"
	cfg.Fallbacks = []string{"secondary"}
	cfg.ReconnectInitial = 5 * time.Millisecond
	cfg.ReconnectMax = 10 * time.Millisecond
	cfg.HealthInterval = 200 * time.Millisecond // stableAfter; short sessions stay in retry budget

	var primaryBuilds atomic.Int64
	factory := provider.Factory(func(name string) (provider.MarketProvider, error) {
		switch name {
		case "primary":
			primaryBuilds.Add(1)
			p := mock.New(config.MockProviderConfig{TickInterval: 5 * time.Millisecond})
			p.FailAfter = 2 // ~10ms session << HealthInterval → counts toward failover
			return namedProvider{MarketProvider: p, name: "primary"}, nil
		case "secondary":
			p := mock.New(config.MockProviderConfig{
				TickInterval: 5 * time.Millisecond,
				XAUUSDBase:   2200,
				USDTRYBase:   33,
			})
			return namedProvider{MarketProvider: p, name: "secondary"}, nil
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
			t.Fatalf("timeout waiting for failover; active=%q switches=%d reconnects=%d primary_builds=%d",
				mgr.ActiveName(), mgr.SwitchCount(), mgr.ReconnectCount(), primaryBuilds.Load())
		case <-mgr.PriceChannel():
			if mgr.ActiveName() == "secondary" {
				cancel()
				_ = mgr.Close()
				if mgr.SwitchCount() == 0 {
					t.Fatal("expected at least one provider switch")
				}
				return
			}
		}
	}
}

func TestDefaultFactoryUnknown(t *testing.T) {
	t.Parallel()
	f := provider.DefaultFactory(config.ProviderConfig{})
	if _, err := f("nope"); err == nil {
		t.Fatal("expected error")
	}
}

// namedProvider overrides Name() so failover chain names match factory keys.
type namedProvider struct {
	provider.MarketProvider
	name string
}

func (n namedProvider) Name() string { return n.name }
