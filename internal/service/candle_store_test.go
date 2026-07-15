package service

import (
	"context"
	"testing"
	"time"

	"github.com/tsc-forex/tsc_forex/internal/domain"
	"github.com/tsc-forex/tsc_forex/internal/models"
	"github.com/tsc-forex/tsc_forex/pkg/eventbus"
)

func feed(bus *eventbus.Bus, code string, price float64, ts time.Time) {
	bus.Publish(eventbus.TopicInstrumentUpdated, domain.InstrumentPrice{
		Code: code, Price: price, Timestamp: ts,
	})
}

func TestCandleStoreAggregatesLiveCandle(t *testing.T) {
	bus := eventbus.New()
	defer bus.Close()
	s := NewCandleStore(bus, nil, nil) // no DB: in-memory only

	base := time.Date(2026, 7, 15, 12, 0, 5, 0, time.UTC) // inside one 1m bucket
	feed(bus, "USDTRY", 34.0, base)
	feed(bus, "USDTRY", 34.6, base.Add(10*time.Second)) // high
	feed(bus, "USDTRY", 33.8, base.Add(20*time.Second)) // low
	feed(bus, "USDTRY", 34.2, base.Add(30*time.Second)) // close

	got, err := s.Candles(context.Background(), "USDTRY", "1m", time.Time{}, time.Time{}, 100)
	if err != nil {
		t.Fatalf("Candles: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("want 1 candle, got %d", len(got))
	}
	c := got[0]
	if c.Open != 34.0 || c.High != 34.6 || c.Low != 33.8 || c.Close != 34.2 {
		t.Fatalf("OHLC wrong: %+v", c)
	}
	if c.Count != 4 {
		t.Fatalf("count = %d want 4", c.Count)
	}
}

func TestCandleStoreRollsBuckets(t *testing.T) {
	bus := eventbus.New()
	defer bus.Close()
	s := NewCandleStore(bus, nil, nil)

	t0 := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	feed(bus, "BTC", 100, t0)
	feed(bus, "BTC", 110, t0.Add(30*time.Second))
	// Next minute → new candle.
	feed(bus, "BTC", 120, t0.Add(70*time.Second))
	feed(bus, "BTC", 90, t0.Add(80*time.Second))

	got, _ := s.Candles(context.Background(), "BTC", "1m", time.Time{}, time.Time{}, 100)
	// Without a DB the closed candle isn't persisted, but the live (second) bucket
	// must reflect the latest minute.
	if len(got) == 0 {
		t.Fatal("no candles")
	}
	last := got[len(got)-1]
	if last.Open != 120 || last.High != 120 || last.Low != 90 || last.Close != 90 {
		t.Fatalf("live candle wrong: %+v", last)
	}
}

func TestRebucketDerivesCoarser(t *testing.T) {
	// Three consecutive 1d candles re-bucketed into one 1w candle.
	base := time.Date(2026, 7, 13, 0, 0, 0, 0, time.UTC)
	in := []models.OHLC{
		{Bucket: base, Open: 10, High: 12, Low: 9, Close: 11, Count: 1},
		{Bucket: base.Add(24 * time.Hour), Open: 11, High: 15, Low: 10, Close: 14, Count: 1},
		{Bucket: base.Add(48 * time.Hour), Open: 14, High: 16, Low: 8, Close: 9, Count: 1},
	}
	out := rebucket(in, 86400*7)
	if len(out) != 1 {
		t.Fatalf("want 1 weekly candle, got %d", len(out))
	}
	w := out[0]
	if w.Open != 10 || w.High != 16 || w.Low != 8 || w.Close != 9 || w.Count != 3 {
		t.Fatalf("weekly OHLC wrong: %+v", w)
	}
}

func TestSparklineFromMemory(t *testing.T) {
	bus := eventbus.New()
	defer bus.Close()
	s := NewCandleStore(bus, nil, nil)
	now := time.Now().UTC()
	for i := 0; i < 5; i++ {
		feed(bus, "EURTRY", float64(40+i), now.Add(time.Duration(i)*time.Second))
	}
	sp := s.Sparkline("EURTRY", 3)
	if len(sp) != 3 {
		t.Fatalf("want 3 points, got %d", len(sp))
	}
	if sp[2] != 44 {
		t.Fatalf("last spark = %v want 44", sp[2])
	}
}
