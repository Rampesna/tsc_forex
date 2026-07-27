package lifecycle

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDrainerTransitionsOnceAndFailsReadiness(t *testing.T) {
	t.Parallel()

	d := NewDrainer()
	if d.IsDraining() {
		t.Fatal("new drainer is draining")
	}
	if err := d.ReadinessError(); err != nil {
		t.Fatalf("new drainer readiness: %v", err)
	}
	if !d.BeginDrain() {
		t.Fatal("first BeginDrain did not transition")
	}
	if d.BeginDrain() {
		t.Fatal("second BeginDrain transitioned")
	}
	if !d.IsDraining() {
		t.Fatal("drainer did not retain terminal state")
	}
	if !errors.Is(d.ReadinessError(), ErrDraining) {
		t.Fatalf("readiness error = %v", d.ReadinessError())
	}
	select {
	case <-d.Started():
	default:
		t.Fatal("Started was not closed")
	}
}

func TestWaitDrainDelayHonorsDelayAndCancellation(t *testing.T) {
	t.Parallel()

	start := time.Now()
	if err := WaitDrainDelay(context.Background(), 20*time.Millisecond); err != nil {
		t.Fatalf("delay: %v", err)
	}
	if elapsed := time.Since(start); elapsed < 15*time.Millisecond {
		t.Fatalf("delay returned too early: %v", elapsed)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := WaitDrainDelay(ctx, time.Minute); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled delay = %v", err)
	}
}
