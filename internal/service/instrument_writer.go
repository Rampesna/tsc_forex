package service

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/tsc-forex/tsc_forex/internal/domain"
	"github.com/tsc-forex/tsc_forex/internal/repository"
	"github.com/tsc-forex/tsc_forex/pkg/eventbus"
)

// tickThrottle is the minimum interval between persisted raw ticks per code.
// Charts/history now read from pre-aggregated candles, so raw ticks only need to
// capture recent fine detail — persisting at most one row per code per interval
// cuts write volume by ~10-50x on a volatile market without losing candles
// (candles are built from ALL updates in the CandleStore, not from these rows).
const tickThrottle = 3 * time.Second

// InstrumentWriter persists throttled instrument updates asynchronously for
// recent fine-grained history. Uses a worker pool to protect the hot path.
type InstrumentWriter struct {
	repo    repository.PriceRepository
	log     *zap.Logger
	queue   chan domain.InstrumentPrice
	workers int
	wg      sync.WaitGroup

	mu        sync.Mutex
	dropped   int64
	lastSaved map[string]time.Time
}

// NewInstrumentWriter subscribes to instrument updates and buffers them.
func NewInstrumentWriter(bus *eventbus.Bus, repo repository.PriceRepository, workers, queueSize int, log *zap.Logger) *InstrumentWriter {
	if log == nil {
		log = zap.NewNop()
	}
	if workers <= 0 {
		workers = 4
	}
	if queueSize <= 0 {
		queueSize = 8192
	}
	w := &InstrumentWriter{
		repo:      repo,
		log:       log,
		queue:     make(chan domain.InstrumentPrice, queueSize),
		workers:   workers,
		lastSaved: make(map[string]time.Time, 256),
	}
	bus.Subscribe(eventbus.TopicInstrumentUpdated, w.onUpdate)
	return w
}

func (w *InstrumentWriter) onUpdate(ev eventbus.Event) {
	ip, ok := ev.Payload.(domain.InstrumentPrice)
	if !ok {
		return
	}
	// Per-code throttle: drop updates that arrive within tickThrottle of the last
	// persisted row for the same instrument.
	now := time.Now()
	w.mu.Lock()
	if last, seen := w.lastSaved[ip.Code]; seen && now.Sub(last) < tickThrottle {
		w.mu.Unlock()
		return
	}
	w.lastSaved[ip.Code] = now
	w.mu.Unlock()

	select {
	case w.queue <- ip:
	default:
		w.mu.Lock()
		w.dropped++
		dropped := w.dropped
		w.mu.Unlock()
		if dropped%100 == 1 {
			w.log.Warn("instrument history queue full", zap.Int64("dropped_total", dropped))
		}
	}
}

// Start launches the worker pool.
func (w *InstrumentWriter) Start(ctx context.Context) {
	for i := 0; i < w.workers; i++ {
		w.wg.Add(1)
		go w.worker(ctx)
	}
}

func (w *InstrumentWriter) worker(ctx context.Context) {
	defer w.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case ip, ok := <-w.queue:
			if !ok {
				return
			}
			if err := w.repo.Save(ctx, ip); err != nil {
				w.log.Warn("persist instrument failed", zap.Error(err), zap.String("code", ip.Code))
			}
		}
	}
}

// Stop waits for workers to drain.
func (w *InstrumentWriter) Stop() { w.wg.Wait() }

// Dropped returns records dropped due to a full queue.
func (w *InstrumentWriter) Dropped() int64 {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.dropped
}
