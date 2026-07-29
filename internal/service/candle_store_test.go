package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/tsc-forex/tsc_forex/internal/domain"
	"github.com/tsc-forex/tsc_forex/internal/models"
	"github.com/tsc-forex/tsc_forex/internal/repository"
	"github.com/tsc-forex/tsc_forex/pkg/eventbus"
)

type noopCandleRepository struct{}

var _ repository.CandleRepository = noopCandleRepository{}

func (noopCandleRepository) UpsertBatch(context.Context, []models.Candle) error { return nil }
func (noopCandleRepository) Range(context.Context, string, string, time.Time, time.Time, int) ([]models.OHLC, error) {
	return nil, nil
}
func (noopCandleRepository) FirstOpenSince(context.Context, string, time.Time) (map[string]float64, error) {
	return nil, nil
}
func (noopCandleRepository) SeedFromTicks(context.Context, string, int, time.Time, time.Time, []string) (int64, error) {
	return 0, nil
}
func (noopCandleRepository) MissingHistoryCodes(context.Context, string, time.Time, []string) ([]string, error) {
	return nil, nil
}
func (noopCandleRepository) TryLockBootstrap(context.Context) (bool, func(), error) {
	return false, func() {}, nil
}
func (noopCandleRepository) Prune(context.Context, string, time.Time) (int64, error) {
	return 0, nil
}

type recordingCandleRepository struct {
	noopCandleRepository
	mu      sync.Mutex
	batches [][]models.Candle
	fail    bool
}

func (r *recordingCandleRepository) UpsertBatch(_ context.Context, batch []models.Candle) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.batches = append(r.batches, append([]models.Candle(nil), batch...))
	if r.fail {
		return errors.New("transient write failure")
	}
	return nil
}

func (r *recordingCandleRepository) snapshot() [][]models.Candle {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([][]models.Candle, len(r.batches))
	for i := range r.batches {
		out[i] = append([]models.Candle(nil), r.batches[i]...)
	}
	return out
}

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

func TestRebucketCalendarWeekStartsMonday(t *testing.T) {
	friday := time.Date(2026, 7, 17, 0, 0, 0, 0, time.UTC)
	monday := time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)
	in := []models.OHLC{
		{Bucket: friday, Open: 10, High: 12, Low: 9, Close: 11, Count: 1},
		{Bucket: monday, Open: 20, High: 23, Low: 19, Close: 22, Count: 1},
	}
	out := rebucketCalendar(in, "week")
	if len(out) != 2 {
		t.Fatalf("want 2 calendar weeks, got %d", len(out))
	}
	wantFirst := time.Date(2026, 7, 13, 0, 0, 0, 0, time.UTC)
	if !out[0].Bucket.Equal(wantFirst) || !out[1].Bucket.Equal(monday) {
		t.Fatalf("week buckets not Monday-aligned: %+v", out)
	}
}

func TestRebucketCalendarMonthUsesFirstDay(t *testing.T) {
	in := []models.OHLC{
		{Bucket: time.Date(2026, 1, 30, 0, 0, 0, 0, time.UTC), Open: 10, High: 12, Low: 9, Close: 11, Count: 1},
		{Bucket: time.Date(2026, 2, 2, 0, 0, 0, 0, time.UTC), Open: 11, High: 15, Low: 10, Close: 14, Count: 1},
	}
	out := rebucketCalendar(in, "month")
	if len(out) != 2 {
		t.Fatalf("want 2 calendar months, got %d", len(out))
	}
	if out[0].Bucket.Day() != 1 || out[0].Bucket.Month() != time.January {
		t.Fatalf("first month bucket = %s", out[0].Bucket)
	}
	if out[1].Bucket.Day() != 1 || out[1].Bucket.Month() != time.February {
		t.Fatalf("second month bucket = %s", out[1].Bucket)
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

func TestCandleStoreWorkersStopBeforeDatabaseClose(t *testing.T) {
	t.Parallel()

	s := NewCandleStore(nil, noopCandleRepository{}, nil)
	ctx, cancel := context.WithCancel(context.Background())
	s.Bootstrap(ctx)
	s.Start(ctx)
	cancel()

	waitCtx, waitCancel := context.WithTimeout(context.Background(), time.Second)
	defer waitCancel()
	if err := s.Wait(waitCtx); err != nil {
		t.Fatalf("wait: %v", err)
	}
}

func TestCandleFlushUsesDeterministicLockOrder(t *testing.T) {
	t.Parallel()

	repo := &recordingCandleRepository{}
	s := NewCandleStore(nil, repo, nil)
	s.current["ZED|5m"] = &candleAgg{
		bucket: time.Date(2026, 7, 27, 10, 5, 0, 0, time.UTC),
		open:   3, high: 3, low: 3, close: 3, count: 1,
	}
	s.current["ALPHA|5m"] = &candleAgg{
		bucket: time.Date(2026, 7, 27, 10, 5, 0, 0, time.UTC),
		open:   1, high: 1, low: 1, close: 1, count: 1,
	}
	s.pending = []models.Candle{{
		Code: "ALPHA", Resolution: "1m",
		Bucket: time.Date(2026, 7, 27, 10, 4, 0, 0, time.UTC),
	}}

	s.flush(context.Background())
	batches := repo.snapshot()
	if len(batches) != 1 || len(batches[0]) != 3 {
		t.Fatalf("batches = %#v", batches)
	}
	got := batches[0]
	if got[0].Code != "ALPHA" || got[0].Resolution != "1m" ||
		got[1].Code != "ALPHA" || got[1].Resolution != "5m" ||
		got[2].Code != "ZED" {
		t.Fatalf("non-deterministic order: %#v", got)
	}
}

func TestCandleFlushFollowerDoesNotWriteOrAccumulatePending(t *testing.T) {
	t.Parallel()

	repo := &recordingCandleRepository{}
	s := NewCandleStore(nil, repo, nil)
	s.SetWriteGate(func() bool { return false })
	s.pending = []models.Candle{{
		Code: "USDTRY", Resolution: "1m",
		Bucket: time.Date(2026, 7, 27, 10, 4, 0, 0, time.UTC),
	}}

	s.flush(context.Background())

	if batches := repo.snapshot(); len(batches) != 0 {
		t.Fatalf("follower persisted %d batches, want 0", len(batches))
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.pending) != 0 {
		t.Fatalf("follower retained %d closed candles, want 0", len(s.pending))
	}
}

func TestCandleFlushRequeuesClosedCandlesAfterFailure(t *testing.T) {
	t.Parallel()

	repo := &recordingCandleRepository{fail: true}
	s := NewCandleStore(nil, repo, nil)
	closed := models.Candle{
		Code: "USDTRY", Resolution: "1m",
		Bucket: time.Date(2026, 7, 27, 10, 4, 0, 0, time.UTC),
	}
	s.pending = []models.Candle{closed}

	s.flush(context.Background())

	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.pending) != 1 || s.pending[0].Code != closed.Code ||
		!s.pending[0].Bucket.Equal(closed.Bucket) {
		t.Fatalf("closed candle was not requeued: %#v", s.pending)
	}
}
