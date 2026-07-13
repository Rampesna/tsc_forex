package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/tsc-forex/tsc_forex/internal/models"
	"github.com/tsc-forex/tsc_forex/internal/repository"
	ws "github.com/tsc-forex/tsc_forex/internal/websocket"
)

// UserBroadcaster is satisfied by *websocket.Hub for targeted + global sends.
type UserBroadcaster interface {
	Broadcast(msg []byte)
	SendToUser(userID string, msg []byte)
}

// PushSender delivers a push notification to device tokens (APNs/FCM).
// A real APNs implementation is provided by APNsSender; NopSender is the
// default when push is disabled.
type PushSender interface {
	Send(ctx context.Context, tokens []string, title, body string, data map[string]string) error
}

// NopSender discards pushes (used when APNs is not configured).
type NopSender struct{}

// Send implements PushSender.
func (NopSender) Send(context.Context, []string, string, string, map[string]string) error {
	return nil
}

// Notifier persists notifications, pushes them over WebSocket to the target
// user's live connections, and forwards to APNs/FCM for background delivery.
type Notifier struct {
	notifs  repository.NotificationRepository
	devices repository.DeviceTokenRepository
	hub     UserBroadcaster
	sender  PushSender
	log     *zap.Logger
}

// NewNotifier builds a notifier. repo/devices/sender may be nil (degraded mode).
func NewNotifier(notifs repository.NotificationRepository, devices repository.DeviceTokenRepository, hub UserBroadcaster, sender PushSender, log *zap.Logger) *Notifier {
	if log == nil {
		log = zap.NewNop()
	}
	if sender == nil {
		sender = NopSender{}
	}
	return &Notifier{notifs: notifs, devices: devices, hub: hub, sender: sender, log: log}
}

// Notify records and delivers a notification to a user across all channels.
func (n *Notifier) Notify(ctx context.Context, userID uuid.UUID, alertID *uuid.UUID, code, title, body string, price float64) {
	rec := &models.Notification{
		ID:        uuid.New(),
		UserID:    userID,
		AlertID:   alertID,
		Code:      code,
		Title:     title,
		Body:      body,
		Price:     price,
		CreatedAt: time.Now().UTC(),
	}
	if n.notifs != nil {
		if err := n.notifs.Create(ctx, rec); err != nil {
			n.log.Warn("persist notification failed", zap.Error(err))
		}
	}

	// Live WS push to the user's active connections.
	if n.hub != nil {
		if msg, err := ws.EncodeNotification(ws.NotificationPayload{
			ID:    rec.ID.String(),
			Code:  code,
			Title: title,
			Body:  body,
			Price: price,
		}); err == nil {
			n.hub.SendToUser(userID.String(), msg)
		}
	}

	// Background push via APNs/FCM.
	if n.devices != nil {
		tokens, err := n.devices.ListByUser(ctx, userID)
		if err == nil && len(tokens) > 0 {
			addrs := make([]string, 0, len(tokens))
			for _, t := range tokens {
				addrs = append(addrs, t.Token)
			}
			if err := n.sender.Send(ctx, addrs, title, body, map[string]string{"code": code}); err != nil {
				n.log.Warn("push send failed", zap.Error(err))
			}
		}
	}
}
