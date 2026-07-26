package api

import (
	"testing"
	"time"

	"github.com/tsc-forex/tsc_forex/internal/domain"
	"github.com/tsc-forex/tsc_forex/internal/models"
)

func TestLatestTradingSessionSkipsSparseWeekendCandles(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Istanbul")
	if err != nil {
		t.Fatal(err)
	}
	candles := []models.OHLC{
		{Bucket: time.Date(2026, 7, 24, 13, 30, 0, 0, time.UTC), Close: 100, Count: 20},
		{Bucket: time.Date(2026, 7, 24, 13, 35, 0, 0, time.UTC), Close: 101, Count: 20},
		{Bucket: time.Date(2026, 7, 24, 13, 40, 0, 0, time.UTC), Close: 102, Count: 20},
		// Repeated last-close heartbeats written while the exchange is closed.
		{Bucket: time.Date(2026, 7, 25, 1, 10, 0, 0, time.UTC), Close: 102, Count: 1},
		{Bucket: time.Date(2026, 7, 26, 8, 10, 0, 0, time.UTC), Close: 102, Count: 1},
	}

	got := latestTradingSession(candles, loc, domain.KindStock)
	if len(got) != 3 {
		t.Fatalf("session candle count = %d, want 3", len(got))
	}
	for _, candle := range got {
		if candle.Bucket.Weekday() != time.Friday {
			t.Fatalf("returned non-Friday candle: %s", candle.Bucket)
		}
	}
}

func TestLatestTradingSessionUsesNewestRenderableDay(t *testing.T) {
	candles := []models.OHLC{
		// Deliberately unordered to keep the selector independent of repository
		// ordering.
		{Bucket: time.Date(2026, 7, 24, 10, 5, 0, 0, time.UTC), Close: 102},
		{Bucket: time.Date(2026, 7, 23, 10, 0, 0, 0, time.UTC), Close: 100},
		{Bucket: time.Date(2026, 7, 23, 10, 5, 0, 0, time.UTC), Close: 101},
		{Bucket: time.Date(2026, 7, 24, 10, 0, 0, 0, time.UTC), Close: 101},
	}

	got := latestTradingSession(candles, time.UTC, domain.KindStock)
	if len(got) != 2 || got[0].Bucket.Day() != 24 || got[1].Bucket.Day() != 24 {
		t.Fatalf("got %#v, want newest day only", got)
	}
}

func TestLatestTradingSessionRequiresDistinctBuckets(t *testing.T) {
	duplicate := time.Date(2026, 7, 26, 8, 10, 0, 0, time.UTC)
	candles := []models.OHLC{
		{Bucket: time.Date(2026, 7, 24, 10, 0, 0, 0, time.UTC), Close: 100},
		{Bucket: time.Date(2026, 7, 24, 10, 5, 0, 0, time.UTC), Close: 101},
		{Bucket: duplicate},
		{Bucket: duplicate},
	}

	got := latestTradingSession(candles, time.UTC, domain.KindStock)
	if len(got) != 2 || got[0].Bucket.Day() != 24 {
		t.Fatalf("got %#v, want the prior session with distinct buckets", got)
	}
}

func TestLatestTradingSessionRejectsRepeatedCloseHeartbeats(t *testing.T) {
	candles := []models.OHLC{
		{Bucket: time.Date(2026, 7, 24, 19, 55, 0, 0, time.UTC), Close: 332.50, Count: 20},
		{Bucket: time.Date(2026, 7, 24, 20, 0, 0, 0, time.UTC), Close: 333.02, Count: 20},
		{Bucket: time.Date(2026, 7, 26, 8, 10, 0, 0, time.UTC), Close: 333.02, Count: 1},
		{Bucket: time.Date(2026, 7, 26, 9, 30, 0, 0, time.UTC), Close: 333.02, Count: 1},
		{Bucket: time.Date(2026, 7, 26, 10, 50, 0, 0, time.UTC), Close: 333.02, Count: 1},
	}

	got := latestTradingSession(candles, time.UTC, domain.KindStock)
	if len(got) != 2 || got[0].Bucket.Weekday() != time.Friday {
		t.Fatalf("got %#v, want the last active Friday session", got)
	}
	if sessionHasTradingActivity(candles[2:]) {
		t.Fatal("repeated one-tick close heartbeats must not count as market activity")
	}
}

func TestTradingSessionFallbackDoesNotApplyToCrypto(t *testing.T) {
	from := time.Now().Add(-24 * time.Hour)
	sparse := []models.OHLC{{Bucket: time.Now()}}

	if shouldUseLatestTradingSession(
		domain.Instrument{Kind: domain.KindCrypto}, "5m", from, time.Time{}, sparse, true,
	) {
		t.Fatal("crypto must keep its continuous 24-hour window")
	}
	if !shouldUseLatestTradingSession(
		domain.Instrument{Kind: domain.KindStock}, "5m", from, time.Time{}, sparse, false,
	) {
		t.Fatal("closed stock market should use the latest trading session")
	}
	if !shouldUseLatestTradingSession(
		domain.Instrument{Kind: domain.KindGold},
		"5m",
		from,
		time.Time{},
		[]models.OHLC{{}, {}},
		true,
	) {
		t.Fatal("explicit latest-session requests must be normalized even with sparse heartbeats")
	}
}
