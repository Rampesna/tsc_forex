package calculator_test

import (
	"testing"
	"time"

	"github.com/tsc-forex/tsc_forex/internal/calculator"
	"github.com/tsc-forex/tsc_forex/internal/domain"
	"github.com/tsc-forex/tsc_forex/pkg/eventbus"
)

func TestEngineComputesCatalog(t *testing.T) {
	bus := eventbus.New()
	defer bus.Close()
	e := calculator.NewEngine(bus, nil)
	e.Attach()

	var updates int
	bus.Subscribe(eventbus.TopicInstrumentUpdated, func(ev eventbus.Event) {
		if _, ok := ev.Payload.(domain.InstrumentPrice); ok {
			updates++
		}
	})

	now := time.Now().UTC()
	bus.Publish(eventbus.TopicTickReceived, domain.Tick{Symbol: domain.SymbolXAUUSD, Price: 2350, Bid: 2349, Ask: 2351, Timestamp: now, Provider: "test"})
	// Feed a price for every world currency so the full catalog computes.
	for i, c := range domain.Currencies {
		bus.Publish(eventbus.TopicTickReceived, domain.Tick{
			Symbol: c.ISO + "TRY", Price: 34.5 + float64(i), Timestamp: now, Provider: "test",
		})
	}
	// Feed crypto + BIST equity legs (direct TRY values) so they compute too.
	for i, c := range domain.Cryptos {
		bus.Publish(eventbus.TopicTickReceived, domain.Tick{
			Symbol: c.Symbol, Price: 1000 + float64(i), Timestamp: now, Provider: "test",
		})
	}
	for i, s := range domain.AllStocks() {
		bus.Publish(eventbus.TopicTickReceived, domain.Tick{
			Symbol: s.Symbol, Price: 50 + float64(i), Timestamp: now, Provider: "test",
		})
	}
	// USDTRY must be exactly 34.5 for the assertion below.
	bus.Publish(eventbus.TopicTickReceived, domain.Tick{Symbol: domain.SymbolUSDTRY, Price: 34.5, Bid: 34.4, Ask: 34.6, Timestamp: now, Provider: "test"})

	snap := e.Snapshot()
	if len(snap) != len(domain.Catalog) {
		t.Fatalf("snapshot size = %d, want %d", len(snap), len(domain.Catalog))
	}
	if updates == 0 {
		t.Fatal("no updates published")
	}

	// Gram gold should be present and positive.
	gram, ok := e.Get("GRAM_GOLD")
	if !ok || gram.Price <= 0 {
		t.Fatalf("gram gold not computed: %+v", gram)
	}
	usd, ok := e.Get("USDTRY")
	if !ok || usd.Price != 34.5 {
		t.Fatalf("usdtry = %+v", usd)
	}
}

func TestEngineChangeDetection(t *testing.T) {
	bus := eventbus.New()
	defer bus.Close()
	e := calculator.NewEngine(bus, nil)
	e.Attach()

	now := time.Now().UTC()
	bus.Publish(eventbus.TopicTickReceived, domain.Tick{Symbol: domain.SymbolUSDTRY, Price: 34.5, Timestamp: now})
	before := e.UpdateCount()
	// Same price again → no new update for USDTRY.
	bus.Publish(eventbus.TopicTickReceived, domain.Tick{Symbol: domain.SymbolUSDTRY, Price: 34.5, Timestamp: now})
	if e.UpdateCount() != before {
		t.Fatalf("expected no new update, got %d -> %d", before, e.UpdateCount())
	}
}
