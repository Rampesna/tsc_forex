package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/tsc-forex/tsc_forex/internal/config"
)

// FCMSender delivers Android background pushes via Firebase Cloud Messaging
// HTTP v1. Auth uses a service-account JSON (OAuth2 bearer token). Enabled only
// when configured; otherwise the app routes Android tokens to NopSender.
type FCMSender struct {
	projectID  string
	httpClient *http.Client
	log        *zap.Logger
}

// NewFCMSender loads the service account and returns a ready sender. Returns an
// error (so the caller can degrade) when FCM is disabled or misconfigured.
func NewFCMSender(cfg config.NotifyConfig, log *zap.Logger) (*FCMSender, error) {
	if !cfg.FCMEnabled {
		return nil, errors.New("fcm disabled")
	}
	if cfg.FCMCredentialPath == "" || cfg.FCMProjectID == "" {
		return nil, errors.New("fcm credential_path/project_id required")
	}
	raw, err := os.ReadFile(cfg.FCMCredentialPath)
	if err != nil {
		return nil, fmt.Errorf("read fcm credential: %w", err)
	}
	creds, err := google.CredentialsFromJSON(context.Background(), raw,
		"https://www.googleapis.com/auth/firebase.messaging")
	if err != nil {
		return nil, fmt.Errorf("parse fcm credential: %w", err)
	}
	if log == nil {
		log = zap.NewNop()
	}
	// oauth2.NewClient wraps a transport that auto-refreshes the bearer token.
	client := oauth2.NewClient(context.Background(), creds.TokenSource)
	client.Timeout = 10 * time.Second
	return &FCMSender{projectID: cfg.FCMProjectID, httpClient: client, log: log}, nil
}

// Send delivers title/body to each Android device token via FCM HTTP v1.
func (s *FCMSender) Send(ctx context.Context, tokens []string, title, body string, data map[string]string) error {
	url := fmt.Sprintf("https://fcm.googleapis.com/v1/projects/%s/messages:send", s.projectID)
	var firstErr error
	for _, token := range tokens {
		msg := map[string]any{
			"message": map[string]any{
				"token":        token,
				"notification": map[string]string{"title": title, "body": body},
				"data":         data,
				"android":      map[string]any{"priority": "high"},
			},
		}
		payload, _ := json.Marshal(msg)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
		if err != nil {
			firstErr = err
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := s.httpClient.Do(req)
		if err != nil {
			firstErr = err
			continue
		}
		_ = resp.Body.Close()
		if resp.StatusCode >= 300 && firstErr == nil {
			firstErr = fmt.Errorf("fcm status %d", resp.StatusCode)
		}
	}
	return firstErr
}
