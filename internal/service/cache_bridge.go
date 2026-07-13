package service

import (
	"context"

	"go.uber.org/zap"

	"github.com/tsc-forex/tsc_forex/internal/cache"
	"github.com/tsc-forex/tsc_forex/internal/domain"
	"github.com/tsc-forex/tsc_forex/pkg/eventbus"
)

// CacheBridge writes the latest price to Redis and publishes updates so that
// all backend instances stay synchronized (multi-instance fan-out).
type CacheBridge struct {
	cache      *cache.RedisCache
	log        *zap.Logger
	ctx        context.Context
	instanceID string
}

// NewCacheBridge subscribes to gram-gold updates for cache set + publish.
func NewCacheBridge(ctx context.Context, bus *eventbus.Bus, c *cache.RedisCache, instanceID string, log *zap.Logger) *CacheBridge {
	if log == nil {
		log = zap.NewNop()
	}
	b := &CacheBridge{cache: c, log: log, ctx: ctx, instanceID: instanceID}
	bus.Subscribe(eventbus.TopicGramGoldUpdated, b.onUpdate)
	return b
}

func (b *CacheBridge) onUpdate(ev eventbus.Event) {
	g, ok := ev.Payload.(domain.GramGold)
	if !ok {
		return
	}
	if err := b.cache.SetLatest(b.ctx, g); err != nil {
		b.log.Warn("cache set latest failed", zap.Error(err))
	}
	if err := b.cache.Publish(b.ctx, b.instanceID, g); err != nil {
		b.log.Warn("cache publish failed", zap.Error(err))
	}
}
