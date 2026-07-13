package benchmark

import (
	"context"
	"testing"
	"time"

	"github.com/tsc-forex/tsc_forex/internal/calculator"
	"github.com/tsc-forex/tsc_forex/internal/domain"
	"github.com/tsc-forex/tsc_forex/pkg/eventbus"
)

// BenchmarkCalculatorHotPath measures tick → gram-gold recalculation throughput.
func BenchmarkCalculatorHotPath(b *testing.B) {
	bus := eventbus.New()
	defer bus.Close()
	bus.Subscribe(eventbus.TopicGramGoldUpdated, func(eventbus.Event) {})

	calc := calculator.New(calculator.WithBus(bus), calculator.WithEpsilon(0))
	ticks := make(chan domain.Tick, 1024)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go calc.Run(ctx, ticks)

	now := time.Now().UTC()
	ticks <- domain.Tick{Symbol: domain.SymbolUSDTRY, Price: 34.5, Timestamp: now}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ticks <- domain.Tick{Symbol: domain.SymbolXAUUSD, Price: 2350 + float64(i%100), Timestamp: now}
	}
	b.StopTimer()
	// Drain remaining.
	for len(ticks) > 0 {
		time.Sleep(time.Millisecond)
	}
}

// BenchmarkEventBusPublish measures in-process bus fan-out overhead.
func BenchmarkEventBusPublish(b *testing.B) {
	bus := eventbus.New()
	defer bus.Close()
	for i := 0; i < 4; i++ {
		bus.Subscribe(eventbus.TopicTickReceived, func(eventbus.Event) {})
	}
	tick := domain.Tick{Symbol: domain.SymbolXAUUSD, Price: 2350, Timestamp: time.Now()}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bus.Publish(eventbus.TopicTickReceived, tick)
	}
}

// BenchmarkGramGoldFormula measures the pure formula cost.
func BenchmarkGramGoldFormula(b *testing.B) {
	b.ReportAllocs()
	var sink float64
	for i := 0; i < b.N; i++ {
		sink = domain.CalculateGramGoldTRY(2350.5, 34.52)
	}
	_ = sink
}
