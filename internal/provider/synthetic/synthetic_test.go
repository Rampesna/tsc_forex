package synthetic

import (
	"context"
	"testing"
	"time"

	"github.com/tsc-forex/tsc_forex/internal/config"
	"github.com/tsc-forex/tsc_forex/internal/domain"
)

func TestSyntheticEmitsAllLegs(t *testing.T) {
	p := New(config.SyntheticConfig{TickInterval: 10 * time.Millisecond})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := p.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = p.Close() }()

	seen := map[string]bool{}
	deadline := time.After(2 * time.Second)
	for len(seen) < len(domain.BaseSymbols()) {
		select {
		case tick := <-p.PriceChannel():
			if tick.Price <= 0 {
				t.Fatalf("non-positive price for %s", tick.Symbol)
			}
			if tick.Bid <= 0 || tick.Ask <= tick.Bid {
				t.Fatalf("bad bid/ask for %s: %v/%v", tick.Symbol, tick.Bid, tick.Ask)
			}
			seen[tick.Symbol] = true
		case <-deadline:
			t.Fatalf("timeout; only saw %v", seen)
		}
	}
	for _, s := range domain.BaseSymbols() {
		if !seen[s] {
			t.Fatalf("missing leg %s", s)
		}
	}
	if p.Health().Name != "synthetic" {
		t.Fatal("bad health name")
	}
}
