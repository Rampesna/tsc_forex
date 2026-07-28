package service

import (
	"go.uber.org/zap"

	"github.com/tsc-forex/tsc_forex/internal/domain"
	"github.com/tsc-forex/tsc_forex/internal/localization"
	ws "github.com/tsc-forex/tsc_forex/internal/websocket"
	"github.com/tsc-forex/tsc_forex/pkg/eventbus"
)

// InstrumentBroadcaster subscribes to per-instrument updates and pushes them to
// ALL connected WS clients (gold products + currencies), enabling live UI
// updates on web and mobile without polling.
type InstrumentBroadcaster struct {
	hub   Broadcaster
	names *localization.InstrumentLocalizer
	log   *zap.Logger
}

type localizedInstrumentBroadcaster interface {
	BroadcastInstrument(payload ws.InstrumentPayload, displayNames map[string]string)
}

// NewInstrumentBroadcaster wires bus → hub for every instrument update.
func NewInstrumentBroadcaster(
	bus *eventbus.Bus,
	hub Broadcaster,
	log *zap.Logger,
	names ...*localization.InstrumentLocalizer,
) *InstrumentBroadcaster {
	if log == nil {
		log = zap.NewNop()
	}
	b := &InstrumentBroadcaster{hub: hub, log: log}
	if len(names) > 0 {
		b.names = names[0]
	}
	bus.Subscribe(eventbus.TopicInstrumentUpdated, b.onUpdate)
	return b
}

func (b *InstrumentBroadcaster) onUpdate(ev eventbus.Event) {
	ip, ok := ev.Payload.(domain.InstrumentPrice)
	if !ok {
		return
	}
	payload := ws.InstrumentPayload{
		Code:        ip.Code,
		DisplayName: ip.DisplayName,
		Kind:        ip.Kind,
		Price:       ip.Price,
		Bid:         ip.Bid,
		Ask:         ip.Ask,
		Change:      ip.Change,
		ChangePct:   ip.ChangePct,
		Currency:    ip.Currency,
		PriceTRY:    ip.PriceTRY,
		Provider:    ip.Provider,
	}
	if localizedHub, ok := b.hub.(localizedInstrumentBroadcaster); ok && b.names != nil {
		localizedHub.BroadcastInstrument(payload, b.names.DisplayNames(ip.Code))
		return
	}
	msg, err := ws.EncodeInstrument(payload)
	if err != nil {
		b.log.Warn("encode instrument failed", zap.Error(err))
		return
	}
	b.hub.Broadcast(msg)
}
