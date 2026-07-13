package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"

	"github.com/tsc-forex/tsc_forex/internal/cache"
	"github.com/tsc-forex/tsc_forex/internal/config"
	"github.com/tsc-forex/tsc_forex/internal/domain"
	"github.com/tsc-forex/tsc_forex/internal/service"
	"github.com/tsc-forex/tsc_forex/pkg/eventbus"
)

func TestCacheBridgeSetsAndPublishes(t *testing.T) {
	t.Parallel()

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()

	rc, err := cache.NewRedisCache(
		context.Background(),
		config.RedisConfig{Addr: mr.Addr(), PoolSize: 2, PubSubChannel: "gold:test"},
		config.CacheConfig{LatestPriceTTL: time.Minute, LatestPriceKey: "gold:latest"},
		nil, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer rc.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bus := eventbus.New()
	defer bus.Close()
	_ = service.NewCacheBridge(ctx, bus, rc, "instance-X", nil)

	g := domain.GramGold{Symbol: domain.SymbolGramGoldTRY, Price: 2700, Timestamp: time.Now().UTC()}
	bus.Publish(eventbus.TopicGramGoldUpdated, g)

	deadline := time.After(2 * time.Second)
	for {
		got, err := rc.GetLatest(ctx)
		if err == nil && got.Price == 2700 {
			return
		}
		select {
		case <-deadline:
			t.Fatal("latest not cached")
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}
}
