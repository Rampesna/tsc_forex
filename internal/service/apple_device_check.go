package service

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/tsc-forex/tsc_forex/internal/config"
)

const maxDeviceCheckTokenBytes = 4096

// DeviceCheckVerifier consumes Apple's persistent per-device bit 0. Unlike an
// App Attest key or Keychain UUID, this bit survives app reinstall and account
// recreation, closing the core one-lifetime-trial abuse path.
type DeviceCheckVerifier interface {
	Ready() bool
	FreeTrialClaimed(ctx context.Context, token string) (bool, error)
	MarkFreeTrialClaimed(ctx context.Context, token string) error
}

type AppleDeviceCheckVerifier struct {
	cfg        config.BillingConfig
	client     *http.Client
	privateKey *ecdsa.PrivateKey
	loadErr    error
}

func NewAppleDeviceCheckVerifier(cfg config.BillingConfig) *AppleDeviceCheckVerifier {
	v := &AppleDeviceCheckVerifier{cfg: cfg, client: &http.Client{Timeout: 12 * time.Second}}
	if !cfg.DeviceCheckEnabled {
		return v
	}
	if strings.TrimSpace(cfg.AppAttestTeamID) == "" || strings.TrimSpace(cfg.DeviceCheckKeyID) == "" ||
		strings.TrimSpace(cfg.DeviceCheckPrivateKeyPath) == "" {
		v.loadErr = errors.New("DeviceCheck team, key ID and private key path are required")
		return v
	}
	pemBytes, err := os.ReadFile(strings.TrimSpace(cfg.DeviceCheckPrivateKeyPath))
	if err != nil {
		v.loadErr = err
		return v
	}
	if len(pemBytes) == 0 || len(pemBytes) > 16<<10 {
		v.loadErr = errors.New("invalid DeviceCheck private key size")
		return v
	}
	privateKey, err := jwt.ParseECPrivateKeyFromPEM(pemBytes)
	if err != nil || privateKey.Curve != elliptic.P256() {
		v.loadErr = errors.New("invalid DeviceCheck ES256 private key")
		return v
	}
	v.privateKey = privateKey
	return v
}

func (v *AppleDeviceCheckVerifier) Ready() bool {
	return v != nil && v.cfg.DeviceCheckEnabled && v.loadErr == nil && v.privateKey != nil
}

func (v *AppleDeviceCheckVerifier) FreeTrialClaimed(ctx context.Context, token string) (bool, error) {
	if !v.Ready() {
		return false, ErrDeviceCheckUnavailable
	}
	token = strings.TrimSpace(token)
	if !validDeviceCheckToken(token) {
		return false, ErrInvalidDeviceCheckToken
	}
	authorization, err := v.authorizationToken()
	if err != nil {
		return false, ErrDeviceCheckUnavailable
	}
	baseURL := "https://api.devicecheck.apple.com"
	if strings.EqualFold(v.cfg.AppAttestEnvironment, "development") {
		baseURL = "https://api.development.devicecheck.apple.com"
	}
	body := map[string]any{
		"device_token": token, "transaction_id": uuid.NewString(),
		"timestamp": time.Now().UTC().UnixMilli(),
	}
	var bits struct {
		Bit0 bool `json:"bit0"`
		Bit1 bool `json:"bit1"`
	}
	if err := v.post(ctx, baseURL+"/v1/query_two_bits", authorization, body, &bits); err != nil {
		return false, ErrDeviceCheckUnavailable
	}
	return bits.Bit0, nil
}

func (v *AppleDeviceCheckVerifier) MarkFreeTrialClaimed(ctx context.Context, token string) error {
	if !v.Ready() {
		return ErrDeviceCheckUnavailable
	}
	token = strings.TrimSpace(token)
	if !validDeviceCheckToken(token) {
		return ErrInvalidDeviceCheckToken
	}
	authorization, err := v.authorizationToken()
	if err != nil {
		return ErrDeviceCheckUnavailable
	}
	baseURL := "https://api.devicecheck.apple.com"
	if strings.EqualFold(v.cfg.AppAttestEnvironment, "development") {
		baseURL = "https://api.development.devicecheck.apple.com"
	}
	body := map[string]any{
		"device_token": token, "transaction_id": uuid.NewString(),
		"timestamp": time.Now().UTC().UnixMilli(),
	}
	body["bit0"] = true
	body["bit1"] = false
	if err := v.post(ctx, baseURL+"/v1/update_two_bits", authorization, body, nil); err != nil {
		return ErrDeviceCheckUnavailable
	}
	return nil
}

func validDeviceCheckToken(value string) bool {
	if value == "" || len(value) > (maxDeviceCheckTokenBytes*4/3)+8 {
		return false
	}
	for _, encoding := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.RawURLEncoding} {
		decoded, err := encoding.DecodeString(value)
		if err == nil && len(decoded) >= 32 && len(decoded) <= maxDeviceCheckTokenBytes {
			return true
		}
	}
	return false
}

func (v *AppleDeviceCheckVerifier) authorizationToken() (string, error) {
	if !v.Ready() {
		if v != nil && v.loadErr != nil {
			return "", v.loadErr
		}
		return "", ErrDeviceCheckUnavailable
	}
	now := time.Now().UTC()
	claims := jwt.RegisteredClaims{
		Issuer: v.cfg.AppAttestTeamID, IssuedAt: jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(5 * time.Minute)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["kid"] = v.cfg.DeviceCheckKeyID
	return token.SignedString(v.privateKey)
}

func (v *AppleDeviceCheckVerifier) post(ctx context.Context, endpoint, authorization string, input, output any) error {
	payload, err := json.Marshal(input)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+authorization)
	request.Header.Set("Content-Type", "application/json")
	response, err := v.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	limited := io.LimitReader(response.Body, 64<<10)
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, limited)
		return fmt.Errorf("DeviceCheck status %d", response.StatusCode)
	}
	if output == nil {
		_, _ = io.Copy(io.Discard, limited)
		return nil
	}
	if err := json.NewDecoder(limited).Decode(output); err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}
