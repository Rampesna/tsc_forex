package service

import (
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"

	"github.com/tsc-forex/tsc_forex/internal/config"
)

// APNsSender delivers pushes to Apple Push Notification service using
// token-based (p8 key) auth over HTTP/2. It is enabled only when a valid key
// is configured; otherwise the app falls back to NopSender.
type APNsSender struct {
	cfg    config.NotifyConfig
	key    *ecdsa.PrivateKey
	client *http.Client
	log    *zap.Logger

	mu       sync.Mutex
	cached   string
	cachedAt time.Time
}

// NewAPNsSender loads the p8 key and returns a ready sender. Returns an error
// if the key cannot be read/parsed so the caller can degrade to NopSender.
func NewAPNsSender(cfg config.NotifyConfig, log *zap.Logger) (*APNsSender, error) {
	if !cfg.APNsEnabled {
		return nil, errors.New("apns disabled")
	}
	if cfg.APNsKeyPath == "" || cfg.APNsKeyID == "" || cfg.APNsTeamID == "" {
		return nil, errors.New("apns key_path/key_id/team_id required")
	}
	raw, err := os.ReadFile(cfg.APNsKeyPath)
	if err != nil {
		return nil, fmt.Errorf("read apns key: %w", err)
	}
	block, _ := pem.Decode(raw)
	if block == nil {
		return nil, errors.New("invalid p8 pem")
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse p8: %w", err)
	}
	key, ok := parsed.(*ecdsa.PrivateKey)
	if !ok {
		return nil, errors.New("apns key is not ECDSA")
	}
	if log == nil {
		log = zap.NewNop()
	}
	return &APNsSender{
		cfg:    cfg,
		key:    key,
		client: &http.Client{Timeout: 10 * time.Second},
		log:    log,
	}, nil
}

// Send delivers title/body to each device token.
func (s *APNsSender) Send(ctx context.Context, tokens []string, title, body string, data map[string]string) error {
	tok, err := s.authToken()
	if err != nil {
		return err
	}
	host := "https://api.push.apple.com"
	if s.cfg.APNsSandbox {
		host = "https://api.sandbox.push.apple.com"
	}
	payload := buildAPNsPayload(title, body, data)
	var firstErr error
	for _, device := range tokens {
		url := fmt.Sprintf("%s/3/device/%s", host, device)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(payload))
		if err != nil {
			firstErr = err
			continue
		}
		req.Header.Set("authorization", "bearer "+tok)
		req.Header.Set("apns-topic", s.cfg.APNsTopic)
		req.Header.Set("apns-push-type", "alert")
		resp, err := s.client.Do(req)
		if err != nil {
			firstErr = err
			continue
		}
		_ = resp.Body.Close()
		if resp.StatusCode >= 300 && firstErr == nil {
			firstErr = fmt.Errorf("apns status %d", resp.StatusCode)
		}
	}
	return firstErr
}

// authToken returns a cached provider JWT, refreshing every ~50 minutes.
func (s *APNsSender) authToken() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cached != "" && time.Since(s.cachedAt) < 50*time.Minute {
		return s.cached, nil
	}
	t := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.MapClaims{
		"iss": s.cfg.APNsTeamID,
		"iat": time.Now().Unix(),
	})
	t.Header["kid"] = s.cfg.APNsKeyID
	signed, err := t.SignedString(s.key)
	if err != nil {
		return "", err
	}
	s.cached = signed
	s.cachedAt = time.Now()
	return signed, nil
}

func buildAPNsPayload(title, body string, data map[string]string) string {
	var b strings.Builder
	b.WriteString(`{"aps":{"alert":{"title":`)
	b.WriteString(jsonString(title))
	b.WriteString(`,"body":`)
	b.WriteString(jsonString(body))
	b.WriteString(`},"sound":"default"}`)
	for k, v := range data {
		b.WriteString(",")
		b.WriteString(jsonString(k))
		b.WriteString(":")
		b.WriteString(jsonString(v))
	}
	b.WriteString("}")
	return b.String()
}

func jsonString(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`, "\t", `\t`)
	return `"` + r.Replace(s) + `"`
}
