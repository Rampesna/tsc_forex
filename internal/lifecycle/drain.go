// Package lifecycle coordinates application state with an external
// orchestrator such as Kubernetes.
package lifecycle

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

// ErrDraining is returned by readiness checks after shutdown has started.
var ErrDraining = errors.New("service is draining")

// Drainer is the process-wide, monotonic transition from ready to draining.
// Once draining starts it cannot be reversed; Kubernetes must replace the pod.
type Drainer struct {
	draining atomic.Bool
	started  chan struct{}
	once     sync.Once
}

// NewDrainer creates a coordinator in the accepting/ready state.
func NewDrainer() *Drainer {
	return &Drainer{started: make(chan struct{})}
}

// BeginDrain atomically marks the process as draining. It returns true only for
// the caller that performed the transition.
func (d *Drainer) BeginDrain() bool {
	if d == nil || !d.draining.CompareAndSwap(false, true) {
		return false
	}
	d.once.Do(func() { close(d.started) })
	return true
}

// IsDraining reports whether the process has begun its terminal drain phase.
func (d *Drainer) IsDraining() bool {
	return d != nil && d.draining.Load()
}

// ReadinessError can be composed into /readyz. Liveness deliberately remains
// independent so Kubernetes does not kill a pod that is draining correctly.
func (d *Drainer) ReadinessError() error {
	if d.IsDraining() {
		return ErrDraining
	}
	return nil
}

// Started is closed exactly once when draining begins.
func (d *Drainer) Started() <-chan struct{} {
	if d == nil {
		ch := make(chan struct{})
		return ch
	}
	return d.started
}

// WaitDrainDelay keeps existing requests operational while readiness changes
// propagate through EndpointSlices, kube-proxy and external load balancers.
func WaitDrainDelay(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
