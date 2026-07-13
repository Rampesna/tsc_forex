package service

import (
	"go.uber.org/zap"

	"github.com/tsc-forex/tsc_forex/internal/domain"
	ws "github.com/tsc-forex/tsc_forex/internal/websocket"
	"github.com/tsc-forex/tsc_forex/pkg/eventbus"
)

// Broadcaster is satisfied by *websocket.Hub.
type Broadcaster interface {
	Broadcast(msg []byte)
}

// GoldBroadcaster subscribes to gram-gold updates and pushes JSON to the WS hub.
// Redis fan-out is added in step 7; this keeps a single calculation → many clients path.
type GoldBroadcaster struct {
	hub Broadcaster
	log *zap.Logger
}

// NewGoldBroadcaster wires bus → hub encoding.
func NewGoldBroadcaster(bus *eventbus.Bus, hub Broadcaster, log *zap.Logger) *GoldBroadcaster {
	if log == nil {
		log = zap.NewNop()
	}
	b := &GoldBroadcaster{hub: hub, log: log}
	bus.Subscribe(eventbus.TopicGramGoldUpdated, b.onUpdate)
	return b
}

func (b *GoldBroadcaster) onUpdate(ev eventbus.Event) {
	g, ok := ev.Payload.(domain.GramGold)
	if !ok {
		return
	}
	msg, err := ws.EncodePrice(ws.PricePayload{
		Symbol:   g.Symbol,
		Price:    g.Price,
		MidPrice: g.MidPrice,
		Spread:   g.Spread,
		XAUUSD:   g.XAUUSD,
		USDTRY:   g.USDTRY,
		Provider: g.Provider,
	})
	if err != nil {
		b.log.Warn("encode gram gold failed", zap.Error(err))
		return
	}
	b.hub.Broadcast(msg)
}
