package service_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/tsc-forex/tsc_forex/internal/domain"
	"github.com/tsc-forex/tsc_forex/internal/models"
	"github.com/tsc-forex/tsc_forex/internal/service"
	"github.com/tsc-forex/tsc_forex/pkg/eventbus"
)

// recordingRepo implements repository.GoldRepository for tests.
type recordingRepo struct {
	mu    sync.Mutex
	saved []domain.GramGold
}

func (r *recordingRepo) Save(_ context.Context, g domain.GramGold) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.saved = append(r.saved, g)
	return nil
}

func (r *recordingRepo) History(context.Context, string, time.Time, time.Time, int) ([]models.GoldPrice, error) {
	return nil, nil
}

func (r *recordingRepo) Aggregate(context.Context, string, string, time.Time, time.Time) ([]models.OHLC, error) {
	return nil, nil
}

func (r *recordingRepo) Latest(context.Context, string) (models.GoldPrice, error) {
	return models.GoldPrice{}, nil
}

func (r *recordingRepo) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.saved)
}

func TestHistoryWriterPersists(t *testing.T) {
	t.Parallel()

	bus := eventbus.New()
	defer bus.Close()

	repo := &recordingRepo{}
	hw := service.NewHistoryWriter(bus, repo, 2, 128, nil)

	ctx, cancel := context.WithCancel(context.Background())
	hw.Start(ctx)

	for i := 0; i < 5; i++ {
		bus.Publish(eventbus.TopicGramGoldUpdated, domain.GramGold{
			Symbol: domain.SymbolGramGoldTRY, Price: float64(2500 + i), Timestamp: time.Now().UTC(),
		})
	}

	deadline := time.After(2 * time.Second)
	for repo.count() < 5 {
		select {
		case <-deadline:
			t.Fatalf("saved = %d, want 5", repo.count())
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}

	cancel()
	hw.Stop()
}
