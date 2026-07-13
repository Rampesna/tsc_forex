package service_test

import (
	"testing"
	"time"

	"github.com/tsc-forex/tsc_forex/internal/domain"
	"github.com/tsc-forex/tsc_forex/internal/service"
	"github.com/tsc-forex/tsc_forex/pkg/eventbus"
)

// TestHistoryWriterDropsWhenQueueFull floods a tiny queue with no workers
// started, forcing the drop path and Dropped() accounting.
func TestHistoryWriterDropsWhenQueueFull(t *testing.T) {
	t.Parallel()

	bus := eventbus.New()
	defer bus.Close()

	repo := &recordingRepo{}
	// queueSize 1, workers not started → queue fills and drops.
	hw := service.NewHistoryWriter(bus, repo, 1, 1, nil)

	for i := 0; i < 200; i++ {
		bus.Publish(eventbus.TopicGramGoldUpdated, domain.GramGold{
			Symbol: domain.SymbolGramGoldTRY, Price: float64(i), Timestamp: time.Now().UTC(),
		})
	}
	if hw.Dropped() == 0 {
		t.Fatal("expected dropped records")
	}
}
