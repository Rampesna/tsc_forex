package calculator_test

import (
	"context"
	"math"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/tsc-forex/tsc_forex/internal/calculator"
	"github.com/tsc-forex/tsc_forex/internal/domain"
	"github.com/tsc-forex/tsc_forex/pkg/eventbus"
)

func TestCalculatorEmitsWhenBothLegsReady(t *testing.T) {
	t.Parallel()

	bus := eventbus.New()
	defer bus.Close()

	got := make(chan domain.GramGold, 4)
	bus.Subscribe(eventbus.TopicGramGoldUpdated, func(ev eventbus.Event) {
		g, ok := ev.Payload.(domain.GramGold)
		if ok {
			got <- g
		}
	})

	calc := calculator.New(
		calculator.WithBus(bus),
		calculator.WithTroyOunceToGram(domain.TroyOunceToGram),
		calculator.WithLogger(zap.NewNop()),
		calculator.WithLogger(nil), // nil ignored
		calculator.WithEpsilon(-1), // negative ignored
	)

	ticks := make(chan domain.Tick, 8)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go calc.Run(ctx, ticks)

	now := time.Now().UTC()
	ticks <- domain.Tick{Symbol: domain.SymbolXAUUSD, Bid: 3100, Ask: 3102, Price: 3101, Timestamp: now, Provider: "mock"}
	select {
	case <-got:
		t.Fatal("must not emit before USDTRY")
	case <-time.After(30 * time.Millisecond):
	}

	ticks <- domain.Tick{Symbol: domain.SymbolUSDTRY, Bid: 34, Ask: 34.1, Price: 34.05, Timestamp: now, Provider: "mock"}

	select {
	case g := <-got:
		want := (3101.0 / domain.TroyOunceToGram) * 34.05
		if math.Abs(g.Price-want) > 1e-9 {
			t.Fatalf("price = %v, want %v", g.Price, want)
		}
		if g.Symbol != domain.SymbolGramGoldTRY {
			t.Fatalf("symbol = %s", g.Symbol)
		}
		if g.XAUUSD != 3101 || g.USDTRY != 34.05 {
			t.Fatalf("legs xau=%v usd=%v", g.XAUUSD, g.USDTRY)
		}
		if g.Spread <= 0 {
			t.Fatalf("expected positive spread, got %v", g.Spread)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for gram gold")
	}

	latest, ok := calc.Latest()
	if !ok || latest.Price <= 0 {
		t.Fatalf("Latest = %+v ok=%v", latest, ok)
	}
	if calc.CalculationCount() < 1 || calc.PublishCount() < 1 {
		t.Fatalf("counts calc=%d pub=%d", calc.CalculationCount(), calc.PublishCount())
	}
}

func TestCalculatorSkipsUnchangedPrice(t *testing.T) {
	t.Parallel()

	bus := eventbus.New()
	defer bus.Close()

	var pubs int
	bus.Subscribe(eventbus.TopicGramGoldUpdated, func(eventbus.Event) { pubs++ })

	calc := calculator.New(calculator.WithBus(bus), calculator.WithEpsilon(1e-6))
	ticks := make(chan domain.Tick, 8)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go calc.Run(ctx, ticks)

	now := time.Now().UTC()
	ticks <- domain.Tick{Symbol: domain.SymbolXAUUSD, Price: 2000, Timestamp: now, Provider: "m"}
	ticks <- domain.Tick{Symbol: domain.SymbolUSDTRY, Price: 30, Timestamp: now, Provider: "m"}
	ticks <- domain.Tick{Symbol: domain.SymbolXAUUSD, Price: 2000, Timestamp: now, Provider: "m"} // same
	ticks <- domain.Tick{Symbol: domain.SymbolUSDTRY, Price: 30, Timestamp: now, Provider: "m"}

	deadline := time.After(200 * time.Millisecond)
	for calc.CalculationCount() < 3 {
		select {
		case <-deadline:
			t.Fatalf("calcs=%d pubs=%d", calc.CalculationCount(), calc.PublishCount())
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}
	if calc.PublishCount() != 1 {
		t.Fatalf("publishes = %d, want 1 (unchanged skipped)", calc.PublishCount())
	}
	if pubs != 1 {
		t.Fatalf("bus pubs = %d", pubs)
	}
}

func TestCalculatorUpdatesOnLegChange(t *testing.T) {
	t.Parallel()

	bus := eventbus.New()
	defer bus.Close()

	got := make(chan domain.GramGold, 4)
	bus.Subscribe(eventbus.TopicGramGoldUpdated, func(ev eventbus.Event) {
		got <- ev.Payload.(domain.GramGold)
	})

	calc := calculator.New(calculator.WithBus(bus))
	ticks := make(chan domain.Tick, 8)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go calc.Run(ctx, ticks)

	now := time.Now().UTC()
	ticks <- domain.Tick{Symbol: domain.SymbolXAUUSD, Price: 2000, Timestamp: now}
	ticks <- domain.Tick{Symbol: domain.SymbolUSDTRY, Price: 30, Timestamp: now}
	g1 := <-got

	ticks <- domain.Tick{Symbol: domain.SymbolUSDTRY, Price: 31, Timestamp: now}
	g2 := <-got

	if g2.Price <= g1.Price {
		t.Fatalf("expected price increase: %v → %v", g1.Price, g2.Price)
	}
	want := (2000.0 / domain.TroyOunceToGram) * 31
	if math.Abs(g2.Price-want) > 1e-9 {
		t.Fatalf("price = %v want %v", g2.Price, want)
	}
}

func TestComputeMatchesDomainFormula(t *testing.T) {
	t.Parallel()
	xau, usd := 2350.0, 34.5
	want := domain.CalculateGramGoldTRY(xau, usd)
	calc := calculator.New()
	ticks := make(chan domain.Tick, 2)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go calc.Run(ctx, ticks)
	ticks <- domain.Tick{Symbol: domain.SymbolXAUUSD, Price: xau, Timestamp: time.Now().UTC()}
	ticks <- domain.Tick{Symbol: domain.SymbolUSDTRY, Price: usd, Timestamp: time.Now().UTC()}

	deadline := time.After(time.Second)
	for {
		if g, ok := calc.Latest(); ok {
			if math.Abs(g.Price-want) > 1e-9 {
				t.Fatalf("got %v want %v", g.Price, want)
			}
			return
		}
		select {
		case <-deadline:
			t.Fatal("timeout")
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}
}
