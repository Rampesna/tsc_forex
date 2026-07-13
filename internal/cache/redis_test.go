package cache_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"

	"github.com/tsc-forex/tsc_forex/internal/cache"
	"github.com/tsc-forex/tsc_forex/internal/config"
	"github.com/tsc-forex/tsc_forex/internal/domain"
)

func newTestCache(t *testing.T) (*cache.RedisCache, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	rc, err := cache.NewRedisCache(
		context.Background(),
		config.RedisConfig{
			Addr:          mr.Addr(),
			PoolSize:      4,
			DialTimeout:   time.Second,
			ReadTimeout:   time.Second,
			WriteTimeout:  time.Second,
			PubSubChannel: "gold:test",
		},
		config.CacheConfig{
			LatestPriceTTL: time.Minute,
			LatestPriceKey: "gold:latest",
		},
		nil, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = rc.Close(); mr.Close() })
	return rc, mr
}

func TestSetGetLatest(t *testing.T) {
	rc, _ := newTestCache(t)
	ctx := context.Background()

	if _, err := rc.GetLatest(ctx); err != cache.ErrCacheMiss {
		t.Fatalf("expected cache miss, got %v", err)
	}

	g := domain.GramGold{Symbol: domain.SymbolGramGoldTRY, Price: 2500, XAUUSD: 2300, USDTRY: 34, Provider: "mock", Timestamp: time.Now().UTC()}
	if err := rc.SetLatest(ctx, g); err != nil {
		t.Fatal(err)
	}
	got, err := rc.GetLatest(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got.Price != 2500 || got.Symbol != domain.SymbolGramGoldTRY {
		t.Fatalf("got %+v", got)
	}
}

func TestPublishSubscribeSkipsNothing(t *testing.T) {
	rc, _ := newTestCache(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sub, cleanup, err := rc.Subscribe(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	g := domain.GramGold{Symbol: domain.SymbolGramGoldTRY, Price: 2600, Timestamp: time.Now().UTC()}
	if err := rc.Publish(ctx, "instance-A", g); err != nil {
		t.Fatal(err)
	}

	select {
	case pm := <-sub:
		if pm.Instance != "instance-A" || pm.Gold.Price != 2600 {
			t.Fatalf("got %+v", pm)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for pub/sub message")
	}
}

func TestPing(t *testing.T) {
	rc, _ := newTestCache(t)
	if err := rc.Ping(context.Background()); err != nil {
		t.Fatal(err)
	}
}
