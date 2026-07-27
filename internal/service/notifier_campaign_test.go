package service_test

import (
	"context"
	"sync"
	"testing"

	"github.com/google/uuid"

	"github.com/tsc-forex/tsc_forex/internal/models"
	"github.com/tsc-forex/tsc_forex/internal/repository"
	"github.com/tsc-forex/tsc_forex/internal/service"
)

type campaignNotificationRepo struct {
	mu      sync.Mutex
	byKey   map[uuid.UUID]models.Notification
	records []models.Notification
}

func (r *campaignNotificationRepo) Create(_ context.Context, n *models.Notification) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records = append(r.records, *n)
	return nil
}

func (r *campaignNotificationRepo) CreateOnce(_ context.Context, n *models.Notification) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.byKey == nil {
		r.byKey = make(map[uuid.UUID]models.Notification)
	}
	key := *n.CampaignRecipientID
	if _, exists := r.byKey[key]; exists {
		return false, nil
	}
	r.byKey[key] = *n
	r.records = append(r.records, *n)
	return true, nil
}

func (*campaignNotificationRepo) ListByUser(context.Context, uuid.UUID, int) ([]models.Notification, error) {
	return nil, nil
}
func (*campaignNotificationRepo) MarkRead(context.Context, uuid.UUID, uuid.UUID) error { return nil }
func (*campaignNotificationRepo) Delete(context.Context, uuid.UUID, uuid.UUID) error   { return nil }
func (*campaignNotificationRepo) DeleteAll(context.Context, uuid.UUID) (int64, error)  { return 0, nil }

type campaignDeviceRepo struct{ rows []models.DeviceToken }

func (*campaignDeviceRepo) Upsert(context.Context, *models.DeviceToken) error { return nil }
func (r *campaignDeviceRepo) ListByUser(_ context.Context, userID uuid.UUID) ([]models.DeviceToken, error) {
	out := make([]models.DeviceToken, 0, len(r.rows))
	for _, row := range r.rows {
		if row.UserID == userID {
			out = append(out, row)
		}
	}
	return out, nil
}

type capturedPush struct {
	mu    sync.Mutex
	calls int
	data  map[string]string
}

func (s *capturedPush) Send(_ context.Context, _ []string, _, _ string, data map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	s.data = data
	return nil
}

type campaignHub struct {
	mu    sync.Mutex
	calls int
}

func (*campaignHub) Broadcast([]byte) {}
func (h *campaignHub) SendToUser(string, []byte) {
	h.mu.Lock()
	h.calls++
	h.mu.Unlock()
}

var _ repository.NotificationRepository = (*campaignNotificationRepo)(nil)
var _ repository.DeviceTokenRepository = (*campaignDeviceRepo)(nil)

func TestCampaignNotificationIsIdempotentAndPlatformScoped(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	userID, recipientID := uuid.New(), uuid.New()
	store := &campaignNotificationRepo{}
	devices := &campaignDeviceRepo{rows: []models.DeviceToken{
		{UserID: userID, Platform: "ios", Token: "ios-token"},
		{UserID: userID, Platform: "android", Token: "android-token"},
	}}
	ios, android := &capturedPush{}, &capturedPush{}
	hub := &campaignHub{}
	notifier := service.NewNotifier(store, devices, hub, map[string]service.PushSender{
		"ios": ios, "android": android,
	}, nil)

	err := notifier.NotifyCampaignWithResult(
		ctx, recipientID, userID, "Piyasa uyarısı", "Takip listenizi gözden geçirin.",
		"tscforex://markets", []string{"ios"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if ios.calls != 1 || android.calls != 0 {
		t.Fatalf("push calls ios=%d android=%d", ios.calls, android.calls)
	}
	if ios.data["deep_link"] != "tscforex://markets" {
		t.Fatalf("deep link payload = %#v", ios.data)
	}
	if hub.calls != 1 || len(store.records) != 1 || store.records[0].DeepLink == "" {
		t.Fatalf("hub=%d records=%+v", hub.calls, store.records)
	}

	if err := notifier.NotifyCampaignWithResult(
		ctx, recipientID, userID, "Piyasa uyarısı", "Takip listenizi gözden geçirin.",
		"tscforex://markets", []string{"ios"},
	); err != nil {
		t.Fatal(err)
	}
	if ios.calls != 1 || android.calls != 0 || hub.calls != 1 || len(store.records) != 1 {
		t.Fatalf("retry duplicated delivery: ios=%d android=%d hub=%d records=%d", ios.calls, android.calls, hub.calls, len(store.records))
	}
}
