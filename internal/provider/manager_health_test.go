package provider_test

import (
	"context"
	"testing"
	"time"

	"github.com/tsc-forex/tsc_forex/internal/config"
	"github.com/tsc-forex/tsc_forex/internal/domain"
	"github.com/tsc-forex/tsc_forex/internal/provider"
)

func TestManagerHealthAndStatuses(t *testing.T) {
	cfg := config.ProviderConfig{
		Primary:          "mock",
		Fallbacks:        []string{"twelvedata"},
		ReconnectInitial: 10 * time.Millisecond,
		ReconnectMax:     20 * time.Millisecond,
		HealthInterval:   50 * time.Millisecond,
		StaleThreshold:   time.Hour,
		Mock:             config.MockProviderConfig{TickInterval: 10 * time.Millisecond},
	}
	mgr, err := provider.NewManager(cfg, provider.DefaultFactory(cfg))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := mgr.Run(ctx); err != nil {
		t.Fatal(err)
	}

	// Consume a few ticks so lastTickAt is set.
	deadline := time.After(2 * time.Second)
	for i := 0; i < 3; i++ {
		select {
		case <-mgr.PriceChannel():
		case <-deadline:
			t.Fatal("no ticks")
		}
	}

	h := mgr.Health()
	if h.Name != "mock" {
		t.Fatalf("health name = %q", h.Name)
	}
	if h.Status != domain.ProviderHealthy && h.Status != domain.ProviderUnknown {
		t.Fatalf("unexpected status %q", h.Status)
	}

	statuses := mgr.Statuses()
	if len(statuses) != 2 {
		t.Fatalf("statuses = %d, want 2 (chain)", len(statuses))
	}
	found := false
	for _, s := range statuses {
		if s.Name == "twelvedata" {
			found = true
		}
	}
	if !found {
		t.Fatal("fallback not in statuses")
	}

	cancel()
	_ = mgr.Close()
}
