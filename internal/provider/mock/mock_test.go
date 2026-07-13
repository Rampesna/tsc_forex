package mock_test

import (
	"context"
	"testing"
	"time"

	"github.com/tsc-forex/tsc_forex/internal/config"
	"github.com/tsc-forex/tsc_forex/internal/domain"
	"github.com/tsc-forex/tsc_forex/internal/provider/mock"
)

func TestMockEmitsSubscribedTicks(t *testing.T) {
	t.Parallel()

	p := mock.New(config.MockProviderConfig{
		TickInterval:  20 * time.Millisecond,
		XAUUSDBase:    2000,
		USDTRYBase:    30,
		JitterPercent: 0.01,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := p.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	if err := p.Subscribe(domain.SymbolXAUUSD); err != nil {
		t.Fatal(err)
	}
	if err := p.Subscribe(domain.SymbolUSDTRY); err != nil {
		t.Fatal(err)
	}

	seen := map[string]int{}
	deadline := time.After(2 * time.Second)
	for len(seen) < 2 {
		select {
		case tick, ok := <-p.PriceChannel():
			if !ok {
				t.Fatal("channel closed early")
			}
			if tick.Provider != "mock" {
				t.Fatalf("provider = %q", tick.Provider)
			}
			if tick.Price <= 0 || tick.Bid <= 0 || tick.Ask <= 0 {
				t.Fatalf("invalid tick: %+v", tick)
			}
			seen[tick.Symbol]++
		case <-deadline:
			t.Fatalf("timeout waiting for both symbols, seen=%v", seen)
		}
	}

	_ = p.Close()
	h := p.Health()
	if h.Status != domain.ProviderUnhealthy && h.Name != "mock" {
		// After close, status should be unhealthy.
		t.Fatalf("health after close: %+v", h)
	}
}

func TestMockFailAfterClosesStream(t *testing.T) {
	t.Parallel()

	p := mock.New(config.MockProviderConfig{TickInterval: 10 * time.Millisecond})
	p.FailAfter = 3

	ctx := context.Background()
	if err := p.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	_ = p.Subscribe(domain.SymbolXAUUSD)

	count := 0
	for range p.PriceChannel() {
		count++
	}
	if count != 3 {
		t.Fatalf("ticks = %d, want 3", count)
	}
}
