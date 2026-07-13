package cache

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/tsc-forex/tsc_forex/internal/config"
	"github.com/tsc-forex/tsc_forex/internal/domain"
	"github.com/tsc-forex/tsc_forex/internal/metrics"
)

// ErrCacheMiss indicates the latest price is not present in Redis.
var ErrCacheMiss = errors.New("cache miss")

// RedisCache caches the latest gram-gold price and provides pub/sub fan-out
// so multiple backend instances stay synchronized.
type RedisCache struct {
	client   *redis.Client
	cfg      config.RedisConfig
	cacheCfg config.CacheConfig
	log      *zap.Logger
	metrics  *metrics.Metrics
}

// NewRedisCache connects to Redis and verifies connectivity with PING.
func NewRedisCache(ctx context.Context, cfg config.RedisConfig, cacheCfg config.CacheConfig, log *zap.Logger, m *metrics.Metrics) (*RedisCache, error) {
	if log == nil {
		log = zap.NewNop()
	}
	client := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	})

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		_ = client.Close()
		return nil, err
	}

	return &RedisCache{
		client:   client,
		cfg:      cfg,
		cacheCfg: cacheCfg,
		log:      log,
		metrics:  m,
	}, nil
}

func (c *RedisCache) observe(start time.Time, err error) {
	if c.metrics != nil {
		c.metrics.RedisLatency.Observe(time.Since(start).Seconds())
		if err != nil && !errors.Is(err, redis.Nil) {
			c.metrics.RedisErrors.Inc()
		}
	}
}

// SetLatest stores the latest price with configured TTL.
func (c *RedisCache) SetLatest(ctx context.Context, g domain.GramGold) error {
	start := time.Now()
	data, err := json.Marshal(g)
	if err != nil {
		return err
	}
	err = c.client.Set(ctx, c.cacheCfg.LatestPriceKey, data, c.cacheCfg.LatestPriceTTL).Err()
	c.observe(start, err)
	return err
}

// GetLatest reads the latest cached price.
func (c *RedisCache) GetLatest(ctx context.Context) (domain.GramGold, error) {
	start := time.Now()
	data, err := c.client.Get(ctx, c.cacheCfg.LatestPriceKey).Bytes()
	c.observe(start, err)
	if errors.Is(err, redis.Nil) {
		return domain.GramGold{}, ErrCacheMiss
	}
	if err != nil {
		return domain.GramGold{}, err
	}
	var g domain.GramGold
	if err := json.Unmarshal(data, &g); err != nil {
		return domain.GramGold{}, err
	}
	return g, nil
}

// PriceMessage wraps a gram-gold update with the originating instance ID
// so subscribers can skip messages they published themselves.
type PriceMessage struct {
	Instance string          `json:"instance"`
	Gold     domain.GramGold `json:"gold"`
}

// Publish broadcasts a price update (tagged with instanceID) to the channel.
func (c *RedisCache) Publish(ctx context.Context, instanceID string, g domain.GramGold) error {
	start := time.Now()
	data, err := json.Marshal(PriceMessage{Instance: instanceID, Gold: g})
	if err != nil {
		return err
	}
	err = c.client.Publish(ctx, c.cfg.PubSubChannel, data).Err()
	c.observe(start, err)
	return err
}

// Subscribe returns a channel of price messages from all instances.
// The returned cleanup function must be called to release the subscription.
func (c *RedisCache) Subscribe(ctx context.Context) (<-chan PriceMessage, func(), error) {
	sub := c.client.Subscribe(ctx, c.cfg.PubSubChannel)
	if _, err := sub.Receive(ctx); err != nil {
		_ = sub.Close()
		return nil, nil, err
	}

	out := make(chan PriceMessage, 256)
	go func() {
		defer close(out)
		ch := sub.Channel()
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}
				var pm PriceMessage
				if err := json.Unmarshal([]byte(msg.Payload), &pm); err != nil {
					c.log.Warn("redis sub decode failed", zap.Error(err))
					continue
				}
				select {
				case out <- pm:
				default:
					// Drop rather than block the subscriber goroutine.
				}
			}
		}
	}()

	cleanup := func() { _ = sub.Close() }
	return out, cleanup, nil
}

// Ping checks connectivity (used by health checks).
func (c *RedisCache) Ping(ctx context.Context) error {
	start := time.Now()
	err := c.client.Ping(ctx).Err()
	c.observe(start, err)
	return err
}

// Close releases the Redis client.
func (c *RedisCache) Close() error {
	return c.client.Close()
}
