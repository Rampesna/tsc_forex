package provider

import (
	"testing"
	"time"
)

func TestBackoffExponentialCapped(t *testing.T) {
	t.Parallel()

	b := Backoff{Initial: time.Second, Max: 4 * time.Second}
	got := []time.Duration{b.Next(), b.Next(), b.Next(), b.Next()}
	want := []time.Duration{time.Second, 2 * time.Second, 4 * time.Second, 4 * time.Second}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Next[%d] = %v, want %v", i, got[i], want[i])
		}
	}
	b.Reset()
	if b.Next() != time.Second {
		t.Fatal("expected reset to initial")
	}
}
