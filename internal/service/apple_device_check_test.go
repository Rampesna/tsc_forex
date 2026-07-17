package service

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/golang-jwt/jwt/v5"

	"github.com/tsc-forex/tsc_forex/internal/config"
)

func TestAppleDeviceCheckReadyValidatesMountedES256Key(t *testing.T) {
	base := config.BillingConfig{
		DeviceCheckEnabled: true, AppAttestTeamID: "TEAM123456",
		DeviceCheckKeyID: "KEY123456",
	}

	missing := NewAppleDeviceCheckVerifier(base)
	if missing.Ready() {
		t.Fatal("missing key path reported ready")
	}

	invalidPath := filepath.Join(t.TempDir(), "invalid.p8")
	if err := os.WriteFile(invalidPath, []byte("not a private key"), 0o600); err != nil {
		t.Fatal(err)
	}
	invalid := base
	invalid.DeviceCheckPrivateKeyPath = invalidPath
	if NewAppleDeviceCheckVerifier(invalid).Ready() {
		t.Fatal("invalid private key reported ready")
	}

	validPath := filepath.Join(t.TempDir(), "devicecheck.p8")
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(validPath, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), 0o600); err != nil {
		t.Fatal(err)
	}
	valid := base
	valid.DeviceCheckPrivateKeyPath = validPath
	verifier := NewAppleDeviceCheckVerifier(valid)
	if !verifier.Ready() {
		t.Fatal("valid mounted ES256 key did not report ready")
	}
	authorization, err := verifier.authorizationToken()
	if err != nil || authorization == "" {
		t.Fatalf("authorization token err=%v", err)
	}
	parsed, err := jwt.Parse(authorization, func(token *jwt.Token) (any, error) { return &key.PublicKey, nil })
	if err != nil || parsed == nil || !parsed.Valid || parsed.Header["kid"] != base.DeviceCheckKeyID {
		t.Fatalf("signed authorization token=%v err=%v", parsed, err)
	}
}
