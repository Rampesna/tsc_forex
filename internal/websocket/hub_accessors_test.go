package websocket_test

import (
	"testing"
	"time"

	"github.com/tsc-forex/tsc_forex/internal/config"
	ws "github.com/tsc-forex/tsc_forex/internal/websocket"
)

func TestHubAccessorsAndCloseIdempotent(t *testing.T) {
	t.Parallel()
	cfg := config.WebSocketConfig{
		Path: "/ws", WriteWait: time.Second, PongWait: time.Second,
		PingPeriod: time.Second, MaxMessageSize: 2048, SendBufferSize: 0, // exercise default
	}
	hub := ws.NewHub(cfg, nil)
	go hub.Run()

	if hub.Config().Path != "/ws" {
		t.Fatal("config path")
	}
	if hub.ClientCount() != 0 {
		t.Fatal("clients not zero")
	}
	if hub.DroppedMessages() != 0 {
		t.Fatal("dropped not zero")
	}

	hub.Broadcast([]byte(`{"type":"heartbeat"}`))
	time.Sleep(20 * time.Millisecond)
	if hub.BroadcastCount() < 1 {
		t.Fatal("broadcast not counted")
	}

	hub.Close()
	hub.Close() // idempotent

	// Broadcast/Register after close must not panic.
	hub.Broadcast([]byte("x"))
}
