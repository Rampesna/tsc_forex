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
	// senders routes push by device platform ("ios" → APNs, "android" → FCM).
	senders map[string]PushSender
	log     *zap.Logger
}

// NewNotifier builds a notifier. senders maps a device platform to its push
// backend; missing platforms are silently skipped. Any arg may be nil.
func NewNotifier(notifs repository.NotificationRepository, devices repository.DeviceTokenRepository, hub UserBroadcaster, senders map[string]PushSender, log *zap.Logger) *Notifier {
	if log == nil {
		log = zap.NewNop()
	}
	if senders == nil {
		senders = map[string]PushSender{}
	}
	return &Notifier{notifs: notifs, devices: devices, hub: hub, senders: senders, log: log}
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
	n.deliver(ctx, rec)
}

// deliver sends a notification that has already been constructed (and, for a
// one-shot price alert, atomically persisted with the alert state transition).
// Push/WebSocket delivery remains best-effort; the durable notification record
// is the source of truth users can always see in their notification history.
func (n *Notifier) deliver(ctx context.Context, rec *models.Notification) {
	if rec == nil {
		return
	}

	// Live WS push to the user's active connections.
	if n.hub != nil {
		if msg, err := ws.EncodeNotification(ws.NotificationPayload{
			ID:    rec.ID.String(),
			Code:  rec.Code,
			Title: rec.Title,
			Body:  rec.Body,
			Price: rec.Price,
		}); err == nil {
			n.hub.SendToUser(rec.UserID.String(), msg)
		}
	}

	// Background push via APNs/FCM, routed per device platform.
	if n.devices != nil && len(n.senders) > 0 {
		tokens, err := n.devices.ListByUser(ctx, rec.UserID)
		if err == nil && len(tokens) > 0 {
			byPlatform := make(map[string][]string, len(n.senders))
			for _, t := range tokens {
				byPlatform[t.Platform] = append(byPlatform[t.Platform], t.Token)
			}
			for platform, addrs := range byPlatform {
				sender, ok := n.senders[platform]
				if !ok || len(addrs) == 0 {
					continue
				}
				if err := sender.Send(ctx, addrs, rec.Title, rec.Body, map[string]string{"code": rec.Code}); err != nil {
					n.log.Warn("push send failed", zap.String("platform", platform), zap.Error(err))
				}
			}
		}
	}
}
