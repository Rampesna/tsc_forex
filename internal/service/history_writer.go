package service

import (
	"context"
	"sync"

	"go.uber.org/zap"

	"github.com/tsc-forex/tsc_forex/internal/domain"
	"github.com/tsc-forex/tsc_forex/internal/repository"
	"github.com/tsc-forex/tsc_forex/pkg/eventbus"
)

// HistoryWriter persists every calculated price asynchronously using a
// worker pool so the hot path (calculation → broadcast) is never blocked.
type HistoryWriter struct {
	repo     repository.GoldRepository
	log      *zap.Logger
	queue    chan domain.GramGold
	workers  int
	canWrite func() bool
	wg       sync.WaitGroup
	dropped  int64
	mu       sync.Mutex
}

// NewHistoryWriter subscribes to gram-gold updates and buffers them.
func NewHistoryWriter(bus *eventbus.Bus, repo repository.GoldRepository, workers, queueSize int, log *zap.Logger) *HistoryWriter {
	if log == nil {
		log = zap.NewNop()
	}
	if workers <= 0 {
		workers = 4
	}
	if queueSize <= 0 {
		queueSize = 4096
	}
	h := &HistoryWriter{
		repo:     repo,
		log:      log,
		queue:    make(chan domain.GramGold, queueSize),
		workers:  workers,
		canWrite: func() bool { return true },
	}
	bus.Subscribe(eventbus.TopicGramGoldUpdated, h.onUpdate)
	return h
}

// WithWriteGate limits durable writes to the cluster persistence leader.
func (h *HistoryWriter) WithWriteGate(canWrite func() bool) *HistoryWriter {
	if canWrite != nil {
		h.canWrite = canWrite
	}
	return h
}

func (h *HistoryWriter) onUpdate(ev eventbus.Event) {
	g, ok := ev.Payload.(domain.GramGold)
	if !ok || !h.canWrite() {
		return
	}
	select {
	case h.queue <- g:
	default:
		// Queue full: drop to protect the hot path; count for metrics/logs.
		h.mu.Lock()
		h.dropped++
		dropped := h.dropped
		h.mu.Unlock()
		if dropped%100 == 1 {
			h.log.Warn("history queue full, dropping record", zap.Int64("dropped_total", dropped))
		}
	}
}

// Start launches the worker pool.
func (h *HistoryWriter) Start(ctx context.Context) {
	for i := 0; i < h.workers; i++ {
		h.wg.Add(1)
		go h.worker(ctx)
	}
}

func (h *HistoryWriter) worker(ctx context.Context) {
	defer h.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case g, ok := <-h.queue:
			if !ok {
				return
			}
			if !h.canWrite() {
				continue
			}
			if err := h.repo.Save(ctx, g); err != nil {
				h.log.Warn("persist gold price failed", zap.Error(err), zap.String("symbol", g.Symbol))
			}
		}
	}
}

// Stop waits for workers to drain (call after context cancel).
func (h *HistoryWriter) Stop() {
	h.wg.Wait()
}

// Dropped returns the number of records dropped due to a full queue.
func (h *HistoryWriter) Dropped() int64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.dropped
}
