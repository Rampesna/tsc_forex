package presence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/tsc-forex/tsc_forex/internal/config"
)

const (
	indexKey  = "tsc:presence:index"
	entryBase = "tsc:presence:connection:"
)

// Entry is one live authenticated WebSocket connection. Entries are short
// lived and refreshed by protocol pongs, so crashed pods disappear naturally.
type Entry struct {
	ConnectionID string    `json:"connection_id"`
	UserID       string    `json:"user_id"`
	Platform     string    `json:"platform"`
	UserAgent    string    `json:"user_agent"`
	RemoteAddr   string    `json:"remote_addr"`
	ConnectedAt  time.Time `json:"connected_at"`
	LastSeenAt   time.Time `json:"last_seen_at"`
}

type Store struct {
	client *redis.Client
	ttl    time.Duration
}

func New(ctx context.Context, cfg config.RedisConfig, ttl time.Duration) (*Store, error) {
	if ttl < 30*time.Second {
		ttl = 90 * time.Second
	}
	client := redis.NewClient(&redis.Options{
		Addr: cfg.Addr, Password: cfg.Password, DB: cfg.DB,
		DialTimeout: cfg.DialTimeout, ReadTimeout: cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout, PoolSize: max(4, cfg.PoolSize/4),
	})
	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("presence redis: %w", err)
	}
	return &Store{client: client, ttl: ttl}, nil
}

func (s *Store) Connected(ctx context.Context, entry Entry) error {
	if s == nil || s.client == nil || entry.ConnectionID == "" || entry.UserID == "" {
		return nil
	}
	now := time.Now().UTC()
	if entry.ConnectedAt.IsZero() {
		entry.ConnectedAt = now
	}
	entry.LastSeenAt = now
	raw, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	_, err = s.client.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.Set(ctx, entryBase+entry.ConnectionID, raw, s.ttl)
		pipe.ZAdd(ctx, indexKey, redis.Z{Score: float64(now.Unix()), Member: entry.ConnectionID})
		pipe.Expire(ctx, indexKey, 24*time.Hour)
		return nil
	})
	return err
}

func (s *Store) Heartbeat(ctx context.Context, connectionID string) error {
	if s == nil || s.client == nil || connectionID == "" {
		return nil
	}
	now := time.Now().UTC()
	key := entryBase + connectionID
	raw, err := s.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil
		}
		return err
	}
	var entry Entry
	if err := json.Unmarshal(raw, &entry); err != nil {
		return err
	}
	entry.LastSeenAt = now
	raw, err = json.Marshal(entry)
	if err != nil {
		return err
	}
	_, err = s.client.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.Set(ctx, key, raw, s.ttl)
		pipe.ZAdd(ctx, indexKey, redis.Z{Score: float64(now.Unix()), Member: connectionID})
		return nil
	})
	return err
}

func (s *Store) Disconnected(ctx context.Context, connectionID string) error {
	if s == nil || s.client == nil || connectionID == "" {
		return nil
	}
	_, err := s.client.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.Del(ctx, entryBase+connectionID)
		pipe.ZRem(ctx, indexKey, connectionID)
		return nil
	})
	return err
}

func (s *Store) Online(ctx context.Context) ([]Entry, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	cutoff := time.Now().Add(-s.ttl).Unix()
	_ = s.client.ZRemRangeByScore(ctx, indexKey, "-inf", fmt.Sprintf("%d", cutoff)).Err()
	ids, err := s.client.ZRangeByScore(ctx, indexKey, &redis.ZRangeBy{
		Min: fmt.Sprintf("%d", cutoff), Max: "+inf",
	}).Result()
	if err != nil || len(ids) == 0 {
		return nil, err
	}
	keys := make([]string, 0, len(ids))
	for _, id := range ids {
		keys = append(keys, entryBase+id)
	}
	values, err := s.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, err
	}
	out := make([]Entry, 0, len(values))
	stale := make([]any, 0)
	for i, value := range values {
		text, ok := value.(string)
		if !ok || strings.TrimSpace(text) == "" {
			stale = append(stale, ids[i])
			continue
		}
		var entry Entry
		if json.Unmarshal([]byte(text), &entry) == nil && entry.UserID != "" {
			out = append(out, entry)
		}
	}
	if len(stale) > 0 {
		_ = s.client.ZRem(ctx, indexKey, stale...).Err()
	}
	return out, nil
}

func (s *Store) Close() error {
	if s == nil || s.client == nil {
		return nil
	}
	return s.client.Close()
}
