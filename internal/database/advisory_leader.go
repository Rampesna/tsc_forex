package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// PersistenceLeaderLockKey is intentionally distinct from the schema-migration
// and candle-bootstrap locks. Exactly one API replica may own this session lock
// and perform cluster-wide market persistence / alert evaluation at a time.
const PersistenceLeaderLockKey int64 = 2026073001

const (
	leaderRetryInterval = 2 * time.Second
	leaderProbeInterval = 5 * time.Second
)

// AdvisoryLeader provides a lightweight, PostgreSQL-backed leader election for
// work that must run once per Kubernetes deployment even though the HTTP API is
// horizontally replicated. The advisory lock lives on a dedicated connection;
// PostgreSQL releases it automatically if the process, pod, connection, or
// database disappears. Followers keep retrying, so failover needs no restart.
type AdvisoryLeader struct {
	db  *sql.DB
	key int64
	log *zap.Logger

	leader atomic.Bool
	start  sync.Once
	wg     sync.WaitGroup
}

// NewAdvisoryLeader builds a leader elector from the same pool used by GORM.
func NewAdvisoryLeader(db *gorm.DB, key int64, log *zap.Logger) (*AdvisoryLeader, error) {
	if db == nil {
		return nil, errors.New("leader election requires a database")
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("leader election database handle: %w", err)
	}
	if log == nil {
		log = zap.NewNop()
	}
	return &AdvisoryLeader{db: sqlDB, key: key, log: log}, nil
}

// Start begins acquisition and automatic failover. Calling Start more than once
// is harmless.
func (l *AdvisoryLeader) Start(ctx context.Context) {
	l.start.Do(func() {
		l.wg.Add(1)
		go func() {
			defer l.wg.Done()
			l.run(ctx)
		}()
	})
}

// IsLeader reports whether this replica currently owns the persistence lock.
func (l *AdvisoryLeader) IsLeader() bool {
	return l != nil && l.leader.Load()
}

func (l *AdvisoryLeader) run(ctx context.Context) {
	for ctx.Err() == nil {
		conn, err := l.db.Conn(ctx)
		if err != nil {
			l.log.Warn("persistence leadership connection failed", zap.Error(err))
			if !waitLeaderRetry(ctx) {
				return
			}
			continue
		}

		var acquired bool
		err = conn.QueryRowContext(ctx, "SELECT pg_try_advisory_lock($1)", l.key).Scan(&acquired)
		if err != nil {
			_ = conn.Close()
			l.log.Warn("persistence leadership acquisition failed", zap.Error(err))
			if !waitLeaderRetry(ctx) {
				return
			}
			continue
		}
		if !acquired {
			_ = conn.Close()
			if !waitLeaderRetry(ctx) {
				return
			}
			continue
		}

		l.leader.Store(true)
		l.log.Info("persistence leadership acquired", zap.Int64("lock_key", l.key))
		holdErr := l.hold(ctx, conn)

		// Stop admitting new writes before releasing the database lock. Any
		// already-running bounded write may finish; repositories remain
		// idempotent where cross-pod overlap can occur during handover.
		l.leader.Store(false)
		unlockCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_, _ = conn.ExecContext(unlockCtx, "SELECT pg_advisory_unlock($1)", l.key)
		cancel()
		_ = conn.Close()
		l.log.Info("persistence leadership released", zap.Int64("lock_key", l.key))

		if ctx.Err() != nil {
			return
		}
		if holdErr != nil {
			l.log.Warn("persistence leadership connection lost", zap.Error(holdErr))
		}
		if !waitLeaderRetry(ctx) {
			return
		}
	}
}

func (l *AdvisoryLeader) hold(ctx context.Context, conn *sql.Conn) error {
	ticker := time.NewTicker(leaderProbeInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			probeCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
			var one int
			err := conn.QueryRowContext(probeCtx, "SELECT 1").Scan(&one)
			cancel()
			if err != nil {
				return err
			}
		}
	}
}

func waitLeaderRetry(ctx context.Context) bool {
	timer := time.NewTimer(leaderRetryInterval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

// Wait blocks until the election goroutine releases its dedicated connection.
func (l *AdvisoryLeader) Wait(ctx context.Context) error {
	if l == nil {
		return nil
	}
	done := make(chan struct{})
	go func() {
		l.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
