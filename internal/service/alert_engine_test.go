package service_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/tsc-forex/tsc_forex/internal/domain"
	"github.com/tsc-forex/tsc_forex/internal/models"
	"github.com/tsc-forex/tsc_forex/internal/service"
	"github.com/tsc-forex/tsc_forex/pkg/eventbus"
)

// fakeAlertRepo deliberately keeps returning the same active snapshot so the
// engine test proves ClaimTriggered, rather than a refreshed list, prevents a
// duplicate delivery.
type fakeAlertRepo struct {
	active  []models.PriceAlert
	fired   chan uuid.UUID
	mu      sync.Mutex
	claimed map[uuid.UUID]bool
}

func (f *fakeAlertRepo) Create(context.Context, *models.PriceAlert) error { return nil }
func (f *fakeAlertRepo) Update(context.Context, uuid.UUID, uuid.UUID, string, float64) (models.PriceAlert, error) {
	return models.PriceAlert{}, nil
}
func (f *fakeAlertRepo) Delete(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
}
func (f *fakeAlertRepo) ListByUser(context.Context, uuid.UUID) ([]models.PriceAlert, error) {
	return f.active, nil
}
func (f *fakeAlertRepo) ActiveByCode(_ context.Context, code string) ([]models.PriceAlert, error) {
	out := []models.PriceAlert{}
	for _, a := range f.active {
		if a.Code == code {
			out = append(out, a)
		}
	}
	return out, nil
}
func (f *fakeAlertRepo) ClaimTriggered(_ context.Context, alert models.PriceAlert, _ *models.Notification, _ time.Time) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.claimed == nil {
		f.claimed = make(map[uuid.UUID]bool)
	}
	if f.claimed[alert.ID] {
		return false, nil
	}
	f.claimed[alert.ID] = true
	select {
	case f.fired <- alert.ID:
	default:
	}
	return true, nil
}

// alertFakeHub captures targeted sends.
type alertFakeHub struct {
	direct chan string
}

func (h *alertFakeHub) Broadcast([]byte) {}
func (h *alertFakeHub) SendToUser(userID string, _ []byte) {
	select {
	case h.direct <- userID:
	default:
	}
}

func TestAlertEngineFiresOnlyOnceOnCross(t *testing.T) {
	bus := eventbus.New()
	defer bus.Close()

	uid := uuid.New()
	repo := &fakeAlertRepo{
		fired: make(chan uuid.UUID, 1),
		active: []models.PriceAlert{{
			ID:        uuid.New(),
			UserID:    uid,
			Code:      "GRAM_GOLD",
			Direction: models.AlertAbove,
			Threshold: 2600,
			Active:    true,
		}},
	}
	hub := &alertFakeHub{direct: make(chan string, 1)}
	notifier := service.NewNotifier(nil, nil, hub, nil, nil)
	engine := service.NewAlertEngine(bus, repo, notifier, time.Minute, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	engine.Start(ctx)

	// Price above threshold → should fire.
	bus.Publish(eventbus.TopicInstrumentUpdated, domain.InstrumentPrice{
		Code: "GRAM_GOLD", DisplayName: "Gram Altın", Price: 2650,
	})

	select {
	case got := <-hub.direct:
		if got != uid.String() {
			t.Fatalf("notified %s, want %s", got, uid)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for notification")
	}

	select {
	case <-repo.fired:
	case <-time.After(2 * time.Second):
		t.Fatal("ClaimTriggered not called")
	}

	// A later matching tick may still race with a stale in-memory list on this
	// or another replica. The atomic claim must suppress a second delivery.
	bus.Publish(eventbus.TopicInstrumentUpdated, domain.InstrumentPrice{
		Code: "GRAM_GOLD", DisplayName: "Gram Altın", Price: 2700,
	})
	select {
	case <-hub.direct:
		t.Fatal("one-shot alert delivered more than once")
	case <-time.After(300 * time.Millisecond):
	}
}

func TestAlertEngineNoFireBelowThreshold(t *testing.T) {
	bus := eventbus.New()
	defer bus.Close()
	repo := &fakeAlertRepo{
		fired: make(chan uuid.UUID, 1),
		active: []models.PriceAlert{{
			ID: uuid.New(), UserID: uuid.New(), Code: "USDTRY",
			Direction: models.AlertAbove, Threshold: 100, Active: true,
		}},
	}
	hub := &alertFakeHub{direct: make(chan string, 1)}
	notifier := service.NewNotifier(nil, nil, hub, nil, nil)
	engine := service.NewAlertEngine(bus, repo, notifier, time.Minute, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	engine.Start(ctx)

	bus.Publish(eventbus.TopicInstrumentUpdated, domain.InstrumentPrice{Code: "USDTRY", Price: 34.5})

	select {
	case <-hub.direct:
		t.Fatal("should not fire below threshold")
	case <-time.After(300 * time.Millisecond):
	}
}
