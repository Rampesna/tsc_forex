package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/tsc-forex/tsc_forex/internal/domain"
	"github.com/tsc-forex/tsc_forex/internal/models"
	"github.com/tsc-forex/tsc_forex/internal/repository"
	"github.com/tsc-forex/tsc_forex/pkg/eventbus"
)

// AlertEngine evaluates user price alerts against live instrument updates and
// fires notifications when a threshold is crossed. Evaluation runs off the hot
// path via a bounded worker so the provider goroutine is never blocked.
type AlertEngine struct {
	alerts   repository.AlertRepository
	notifier *Notifier
	log      *zap.Logger
	canWrite func() bool

	queue chan domain.InstrumentPrice
	wg    sync.WaitGroup
}

// NewAlertEngine subscribes to instrument updates. The duration parameter is
// retained for configuration compatibility; alerts are now strictly one-shot.
func NewAlertEngine(bus *eventbus.Bus, alerts repository.AlertRepository, notifier *Notifier, _ time.Duration, log *zap.Logger) *AlertEngine {
	if log == nil {
		log = zap.NewNop()
	}
	e := &AlertEngine{
		alerts:   alerts,
		notifier: notifier,
		log:      log,
		canWrite: func() bool { return true },
		queue:    make(chan domain.InstrumentPrice, 2048),
	}
	bus.Subscribe(eventbus.TopicInstrumentUpdated, e.onUpdate)
	return e
}

// WithWriteGate makes alert evaluation a single-replica responsibility. Claiming
// remains transactional as a final deduplication boundary during leader handoff.
func (e *AlertEngine) WithWriteGate(canWrite func() bool) *AlertEngine {
	if canWrite != nil {
		e.canWrite = canWrite
	}
	return e
}

func (e *AlertEngine) onUpdate(ev eventbus.Event) {
	ip, ok := ev.Payload.(domain.InstrumentPrice)
	if !ok || !e.canWrite() {
		return
	}
	select {
	case e.queue <- ip:
	default:
		// Drop under backpressure; alerts are best-effort real-time.
	}
}

// Start launches the evaluation worker.
func (e *AlertEngine) Start(ctx context.Context) {
	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case ip, ok := <-e.queue:
				if !ok {
					return
				}
				e.evaluate(ctx, ip)
			}
		}
	}()
}

// Stop waits for the worker to finish.
func (e *AlertEngine) Stop() { e.wg.Wait() }

func (e *AlertEngine) evaluate(ctx context.Context, ip domain.InstrumentPrice) {
	if e.alerts == nil || e.notifier == nil || !e.canWrite() {
		return
	}
	active, err := e.alerts.ActiveByCode(ctx, ip.Code)
	if err != nil {
		return
	}
	now := time.Now().UTC()
	for i := range active {
		a := active[i]
		if !crossed(a, ip.Price) {
			continue
		}
		title, body := alertMessage(a, ip)
		alertID := a.ID
		notification := &models.Notification{
			ID:        uuid.New(),
			UserID:    a.UserID,
			AlertID:   &alertID,
			Code:      a.Code,
			Title:     title,
			Body:      body,
			Price:     ip.Price,
			CreatedAt: now,
		}
		// The conditional UPDATE and notification INSERT share one transaction.
		// This is the cross-process deduplication boundary: only one API replica
		// can transition an active alert to triggered, while a notification-write
		// failure rolls the transition back for a later retry. An alert edited after
		// this snapshot also cannot be claimed with stale threshold data.
		claimed, err := e.alerts.ClaimTriggered(ctx, a, notification, now)
		if err != nil {
			e.log.Warn("claim triggered alert failed", zap.Error(err))
			continue
		}
		if !claimed {
			continue
		}
		e.notifier.deliver(ctx, notification, nil)
	}
}

func crossed(a models.PriceAlert, price float64) bool {
	switch a.Direction {
	case models.AlertAbove:
		return price >= a.Threshold
	case models.AlertBelow:
		return price <= a.Threshold
	default:
		return false
	}
}

func alertMessage(a models.PriceAlert, ip domain.InstrumentPrice) (title, body string) {
	price := formatTRY(ip.Price)
	threshold := formatTRY(a.Threshold)
	if a.Direction == models.AlertBelow {
		title = fmt.Sprintf("%s geriledi", ip.DisplayName)
		body = fmt.Sprintf("%s, ₺%s eşiğinin altına indi. Güncel fiyat ₺%s.", ip.DisplayName, threshold, price)
	} else {
		title = fmt.Sprintf("%s yükseldi", ip.DisplayName)
		body = fmt.Sprintf("%s, ₺%s eşiğini yukarı geçti. Güncel fiyat ₺%s.", ip.DisplayName, threshold, price)
	}
	return title, body
}

// formatTRY renders a value with Turkish grouping (1.234,56). Sub-1 values keep
// 4 decimals so low-priced instruments (JPY, SHIB) don't collapse to 0,00.
func formatTRY(v float64) string {
	neg := v < 0
	if neg {
		v = -v
	}
	decimals := 2
	if v > 0 && v < 1 {
		decimals = 4
	}
	s := strconv.FormatFloat(v, 'f', decimals, 64)
	intPart, frac, _ := strings.Cut(s, ".")
	var b strings.Builder
	n := len(intPart)
	for i := 0; i < n; i++ {
		if i > 0 && (n-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteByte(intPart[i])
	}
	out := b.String() + "," + frac
	if neg {
		out = "-" + out
	}
	return out
}
