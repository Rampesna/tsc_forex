package service_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/tsc-forex/tsc_forex/internal/domain"
	"github.com/tsc-forex/tsc_forex/internal/service"
	ws "github.com/tsc-forex/tsc_forex/internal/websocket"
	"github.com/tsc-forex/tsc_forex/pkg/eventbus"
)

type fakeHub struct {
	msgs [][]byte
}

func (f *fakeHub) Broadcast(msg []byte) { f.msgs = append(f.msgs, msg) }

func TestGoldBroadcaster(t *testing.T) {
	t.Parallel()

	bus := eventbus.New()
	defer bus.Close()
	hub := &fakeHub{}
	_ = service.NewGoldBroadcaster(bus, hub, nil)

	bus.Publish(eventbus.TopicGramGoldUpdated, domain.GramGold{
		Symbol:    domain.SymbolGramGoldTRY,
		Price:     2500,
		MidPrice:  2500,
		Spread:    0.5,
		XAUUSD:    2300,
		USDTRY:    34,
		Provider:  "mock",
		Timestamp: time.Now().UTC(),
	})

	if len(hub.msgs) != 1 {
		t.Fatalf("msgs = %d", len(hub.msgs))
	}
	var env ws.Envelope
	if err := json.Unmarshal(hub.msgs[0], &env); err != nil {
		t.Fatal(err)
	}
	if env.Type != ws.TypePriceUpdate {
		t.Fatalf("type = %s", env.Type)
	}
}
