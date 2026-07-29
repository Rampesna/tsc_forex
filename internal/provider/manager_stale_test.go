package provider

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/tsc-forex/tsc_forex/internal/config"
	"github.com/tsc-forex/tsc_forex/internal/domain"
)

func TestForwardGivesFreshSessionItsOwnStaleWindow(t *testing.T) {
	const (
		healthInterval = 10 * time.Millisecond
		staleThreshold = 60 * time.Millisecond
	)

	mgr, err := NewManager(config.ProviderConfig{
		Primary:        "idle",
		HealthInterval: healthInterval,
		StaleThreshold: staleThreshold,
	}, func(string) (MarketProvider, error) {
		return nil, nil
	})
	if err != nil {
		t.Fatal(err)
	}

	// Simulate an old tick left behind by the preceding provider session.
	mgr.lastTickAt.Store(time.Now().Add(-time.Hour).UnixNano())

	p := newIdleProvider()
	done := make(chan struct{})
	go func() {
		mgr.forward(context.Background(), p)
		close(done)
	}()

	select {
	case <-done:
		t.Fatal("fresh session inherited stale state from the previous session")
	case <-time.After(3 * healthInterval):
	}

	select {
	case <-done:
	case <-time.After(3 * staleThreshold):
		t.Fatal("idle session was not reconnected after its own stale window")
	}
}

type idleProvider struct {
	ticks     chan domain.Tick
	closeOnce sync.Once
}

func newIdleProvider() *idleProvider {
	return &idleProvider{ticks: make(chan domain.Tick)}
}

func (*idleProvider) Name() string                       { return "idle" }
func (*idleProvider) Connect(context.Context) error      { return nil }
func (*idleProvider) Subscribe(string) error             { return nil }
func (p *idleProvider) PriceChannel() <-chan domain.Tick { return p.ticks }
func (*idleProvider) Health() domain.ProviderHealth {
	return domain.ProviderHealth{Name: "idle", Status: domain.ProviderDegraded}
}
func (p *idleProvider) Close() error {
	p.closeOnce.Do(func() { close(p.ticks) })
	return nil
}
