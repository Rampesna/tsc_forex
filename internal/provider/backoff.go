package provider

import (
	"fmt"
	"time"
)

// Backoff computes exponential reconnect delays with a hard ceiling.
type Backoff struct {
	Initial time.Duration
	Max     time.Duration
	attempt int
}

// Next returns the delay for the current attempt and increments the counter.
func (b *Backoff) Next() time.Duration {
	if b.Initial <= 0 {
		b.Initial = time.Second
	}
	if b.Max <= 0 {
		b.Max = 30 * time.Second
	}
	d := b.Initial << b.attempt // Initial * 2^attempt
	if d > b.Max || d <= 0 {
		d = b.Max
	}
	b.attempt++
	return d
}

// Reset clears the attempt counter after a successful connection.
func (b *Backoff) Reset() {
	b.attempt = 0
}

// Attempt returns how many failures have been recorded since the last reset.
func (b *Backoff) Attempt() int {
	return b.attempt
}

// String helps logging.
func (b Backoff) String() string {
	return fmt.Sprintf("backoff{initial=%s max=%s attempt=%d}", b.Initial, b.Max, b.attempt)
}
