package eventbus

import "testing"

func TestBusPublishSubscribe(t *testing.T) {
	t.Parallel()

	bus := New()
	defer bus.Close()

	var got any
	bus.Subscribe(TopicTickReceived, func(ev Event) {
		got = ev.Payload
	})

	bus.Publish(TopicTickReceived, "tick-1")
	if got != "tick-1" {
		t.Fatalf("payload = %v, want tick-1", got)
	}
}

func TestBusCloseStopsPublish(t *testing.T) {
	t.Parallel()

	bus := New()
	called := false
	bus.Subscribe(TopicGramGoldUpdated, func(Event) { called = true })
	bus.Close()
	bus.Publish(TopicGramGoldUpdated, struct{}{})
	if called {
		t.Fatal("handler must not run after Close")
	}
}
