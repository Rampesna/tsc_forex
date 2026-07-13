package service

import (
	"go.uber.org/zap"

	"github.com/tsc-forex/tsc_forex/internal/domain"
	ws "github.com/tsc-forex/tsc_forex/internal/websocket"
	"github.com/tsc-forex/tsc_forex/pkg/eventbus"
)

// InstrumentBroadcaster subscribes to per-instrument updates and pushes them to
// ALL connected WS clients (gold products + currencies), enabling live UI
// updates on web and mobile without polling.
type InstrumentBroadcaster struct {
	hub Broadcaster
	log *zap.Logger
}

// NewInstrumentBroadcaster wires bus → hub for every instrument update.
func NewInstrumentBroadcaster(bus *eventbus.Bus, hub Broadcaster, log *zap.Logger) *InstrumentBroadcaster {
	if log == nil {
		log = zap.NewNop()
	}
	b := &InstrumentBroadcaster{hub: hub, log: log}
	bus.Subscribe(eventbus.TopicInstrumentUpdated, b.onUpdate)
	return b
}

func (b *InstrumentBroadcaster) onUpdate(ev eventbus.Event) {
	ip, ok := ev.Payload.(domain.InstrumentPrice)
	if !ok {
		return
	}
	msg, err := ws.EncodeInstrument(ws.InstrumentPayload{
		Code:        ip.Code,
		DisplayName: ip.DisplayName,
		Kind:        ip.Kind,
		Price:       ip.Price,
		Bid:         ip.Bid,
		Ask:         ip.Ask,
		Change:      ip.Change,
		ChangePct:   ip.ChangePct,
		Provider:    ip.Provider,
	})
	if err != nil {
		b.log.Warn("encode instrument failed", zap.Error(err))
		return
	}
	b.hub.Broadcast(msg)
}
