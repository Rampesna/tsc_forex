package eventbus

import (
	"sync"
)

// Event is a typed message published on the in-process bus.
type Event struct {
	Topic   string
	Payload any
}

// Handler processes a single event. Handlers must be non-blocking or
// schedule work themselves; the bus does not wait for slow consumers.
type Handler func(Event)

// Bus is a lightweight in-process pub/sub used on the hot path
// (provider → calculator → downstream). Cross-instance fan-out is
// handled by Redis/NATS, not this bus.
type Bus struct {
	mu       sync.RWMutex
	handlers map[string][]Handler
	closed   bool
}

// New creates an empty event bus.
func New() *Bus {
	return &Bus{
		handlers: make(map[string][]Handler),
	}
}

// Subscribe registers a handler for a topic. Safe for concurrent use.
func (b *Bus) Subscribe(topic string, h Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return
	}
	b.handlers[topic] = append(b.handlers[topic], h)
}

// Publish delivers an event to all handlers of the topic synchronously.
// Handlers run in the publisher's goroutine to keep latency predictable;
// heavy work must be offloaded by the handler.
func (b *Bus) Publish(topic string, payload any) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.closed {
		return
	}
	ev := Event{Topic: topic, Payload: payload}
	for _, h := range b.handlers[topic] {
		h(ev)
	}
}

// Close prevents further publish/subscribe. Idempotent.
func (b *Bus) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.closed = true
	b.handlers = nil
}

// Well-known topics for the gold pipeline.
const (
	TopicTickReceived          = "market.tick"
	TopicGramGoldUpdated       = "gold.gram.updated"
	TopicProviderHealthChanged = "provider.health"
	// TopicInstrumentUpdated carries a single domain.InstrumentPrice whenever a
	// catalog instrument's price changes materially.
	TopicInstrumentUpdated = "instrument.updated"
)
