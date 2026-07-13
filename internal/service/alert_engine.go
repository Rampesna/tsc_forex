package service

import (
	"context"
	"fmt"
	"sync"
	"time"

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
	cooldown time.Duration
	log      *zap.Logger

	queue chan domain.InstrumentPrice
	wg    sync.WaitGroup
}

// NewAlertEngine subscribes to instrument updates. cooldown throttles repeated
// firings of the same alert.
func NewAlertEngine(bus *eventbus.Bus, alerts repository.AlertRepository, notifier *Notifier, cooldown time.Duration, log *zap.Logger) *AlertEngine {
	if log == nil {
		log = zap.NewNop()
	}
	if cooldown <= 0 {
		cooldown = time.Minute
	}
	e := &AlertEngine{
		alerts:   alerts,
		notifier: notifier,
		cooldown: cooldown,
		log:      log,
		queue:    make(chan domain.InstrumentPrice, 2048),
	}
	bus.Subscribe(eventbus.TopicInstrumentUpdated, e.onUpdate)
	return e
}

func (e *AlertEngine) onUpdate(ev eventbus.Event) {
	ip, ok := ev.Payload.(domain.InstrumentPrice)
	if !ok {
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
	if e.alerts == nil || e.notifier == nil {
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
		if a.LastFiredAt != nil && now.Sub(*a.LastFiredAt) < e.cooldown {
			continue
		}
		title, body := alertMessage(a, ip)
		id := a.ID
		e.notifier.Notify(ctx, a.UserID, &id, a.Code, title, body, ip.Price)
		if err := e.alerts.MarkFired(ctx, a.ID, now); err != nil {
			e.log.Warn("mark alert fired failed", zap.Error(err))
		}
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
	dir := "üstüne çıktı"
	if a.Direction == models.AlertBelow {
		dir = "altına indi"
	}
	title = fmt.Sprintf("%s fiyat alarmı", ip.DisplayName)
	body = fmt.Sprintf("%s, %.2f eşiğinin %s. Güncel fiyat: %.2f ₺", ip.DisplayName, a.Threshold, dir, ip.Price)
	return title, body
}
