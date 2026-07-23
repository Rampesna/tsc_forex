package service

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/tsc-forex/tsc_forex/internal/service/applecerts"
)

const (
	maxAppleCompactJWSBytes = 512 << 10
	maxAppleJWSHeaderBytes  = 48 << 10
	maxAppleJWSPayloadBytes = 384 << 10
	maxAppleX5CCertBytes    = 12 << 10
)

// AppleTransactionInfo is the verified subset of StoreKit's
// JWSTransactionDecodedPayload used by entitlement logic.
type AppleTransactionInfo struct {
	TransactionID         string
	OriginalTransactionID string
	BundleID              string
	ProductID             string
	AppAccountToken       string
	Environment           string
	PurchasedAt           time.Time
	ExpiresAt             time.Time
	RevokedAt             *time.Time
}

type AppleNotificationInfo struct {
	NotificationType      string
	Subtype               string
	NotificationUUID      string
	BundleID              string
	Environment           string
	SignedTransactionInfo string
}

// AppleSignedDataVerifier is injectable so billing state tests do not depend
// on Apple's production certificate chain.
type AppleSignedDataVerifier interface {
	VerifyTransaction(signedJWS string) (AppleTransactionInfo, error)
	VerifyNotification(signedJWS string) (AppleNotificationInfo, error)
}

type AppleJWSVerifier struct {
	roots *x509.CertPool
	now   func() time.Time
}

func NewAppleJWSVerifier() (*AppleJWSVerifier, error) {
	roots, err := appleRootCertPool()
	if err != nil {
		return nil, err
	}
	return &AppleJWSVerifier{roots: roots, now: time.Now}, nil
}

func appleRootCertPool() (*x509.CertPool, error) {
	roots := x509.NewCertPool()
	entries, err := applecerts.RootPEMs.ReadDir(".")
	if err != nil {
		return nil, fmt.Errorf("read embedded Apple root certificates: %w", err)
	}
	loaded := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".pem") {
			continue
		}
		raw, readErr := applecerts.RootPEMs.ReadFile(entry.Name())
		if readErr != nil {
			return nil, fmt.Errorf("read Apple root %s: %w", entry.Name(), readErr)
		}
		block, _ := pem.Decode(raw)
		if block == nil || block.Type != "CERTIFICATE" {
			return nil, fmt.Errorf("decode Apple root %s: missing certificate PEM", entry.Name())
		}
		cert, parseErr := x509.ParseCertificate(block.Bytes)
		if parseErr != nil {
			return nil, fmt.Errorf("parse Apple root %s: %w", entry.Name(), parseErr)
		}
		if !appleRootOrgContains(cert.Subject.Organization, "Apple Inc.") {
			return nil, fmt.Errorf("embedded root %s is not an Apple Inc. certificate", entry.Name())
		}
		roots.AddCert(cert)
		loaded++
	}
	if loaded == 0 {
		return nil, errors.New("no embedded Apple root certificates loaded")
	}
	return roots, nil
}

type appleJWSHeader struct {
	Alg string   `json:"alg"`
	X5C []string `json:"x5c"`
}

type appleTransactionPayload struct {
	TransactionID         string `json:"transactionId"`
	OriginalTransactionID string `json:"originalTransactionId"`
	BundleID              string `json:"bundleId"`
	ProductID             string `json:"productId"`
	AppAccountToken       string `json:"appAccountToken"`
	Environment           string `json:"environment"`
	PurchaseDate          int64  `json:"purchaseDate"`
	ExpiresDate           int64  `json:"expiresDate"`
	RevocationDate        int64  `json:"revocationDate"`
}

type appleNotificationPayload struct {
	NotificationType string `json:"notificationType"`
	Subtype          string `json:"subtype"`
	NotificationUUID string `json:"notificationUUID"`
	Data             struct {
		BundleID              string `json:"bundleId"`
		Environment           string `json:"environment"`
		SignedTransactionInfo string `json:"signedTransactionInfo"`
	} `json:"data"`
}

func (v *AppleJWSVerifier) VerifyTransaction(signedJWS string) (AppleTransactionInfo, error) {
	payload, err := v.verifyCompact(signedJWS)
	if err != nil {
		return AppleTransactionInfo{}, err
	}
	var raw appleTransactionPayload
	if err := json.Unmarshal(payload, &raw); err != nil {
		return AppleTransactionInfo{}, fmt.Errorf("decode transaction payload: %w", err)
	}
	if raw.TransactionID == "" || raw.OriginalTransactionID == "" || raw.BundleID == "" || raw.ProductID == "" || raw.PurchaseDate <= 0 || raw.ExpiresDate <= 0 {
		return AppleTransactionInfo{}, errors.New("incomplete transaction payload")
	}
	info := AppleTransactionInfo{
		TransactionID: raw.TransactionID, OriginalTransactionID: raw.OriginalTransactionID,
		BundleID: raw.BundleID, ProductID: raw.ProductID, AppAccountToken: raw.AppAccountToken,
		Environment: raw.Environment, PurchasedAt: appleMillis(raw.PurchaseDate), ExpiresAt: appleMillis(raw.ExpiresDate),
	}
	if raw.RevocationDate > 0 {
		t := appleMillis(raw.RevocationDate)
		info.RevokedAt = &t
	}
	return info, nil
}

func (v *AppleJWSVerifier) VerifyNotification(signedJWS string) (AppleNotificationInfo, error) {
	payload, err := v.verifyCompact(signedJWS)
	if err != nil {
		return AppleNotificationInfo{}, err
	}
	var raw appleNotificationPayload
	if err := json.Unmarshal(payload, &raw); err != nil {
		return AppleNotificationInfo{}, fmt.Errorf("decode notification payload: %w", err)
	}
	if raw.NotificationUUID == "" || raw.Data.BundleID == "" || raw.Data.Environment == "" {
		return AppleNotificationInfo{}, errors.New("incomplete notification payload")
	}
	return AppleNotificationInfo{
		NotificationType: raw.NotificationType, Subtype: raw.Subtype, NotificationUUID: raw.NotificationUUID,
		BundleID: raw.Data.BundleID, Environment: raw.Data.Environment, SignedTransactionInfo: raw.Data.SignedTransactionInfo,
	}, nil
}

func appleMillis(ms int64) time.Time {
	return time.Unix(ms/1000, (ms%1000)*int64(time.Millisecond)).UTC()
}

func (v *AppleJWSVerifier) verifyCompact(compact string) ([]byte, error) {
	compact = strings.TrimSpace(compact)
	if len(compact) == 0 || len(compact) > maxAppleCompactJWSBytes {
		return nil, errors.New("compact JWS exceeds size limit")
	}
	parts := strings.Split(compact, ".")
	if len(parts) != 3 {
		return nil, errors.New("malformed compact JWS")
	}
	if len(parts[0]) > (maxAppleJWSHeaderBytes*4/3)+8 || len(parts[1]) > (maxAppleJWSPayloadBytes*4/3)+8 || len(parts[2]) > 128 {
		return nil, errors.New("compact JWS segment exceeds size limit")
	}
	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil || len(headerJSON) > maxAppleJWSHeaderBytes {
		return nil, errors.New("invalid JWS header encoding")
	}
	var header appleJWSHeader
	if err := json.Unmarshal(headerJSON, &header); err != nil || header.Alg != "ES256" || len(header.X5C) < 2 || len(header.X5C) > 4 {
		return nil, errors.New("unsupported Apple JWS header")
	}
	certs := make([]*x509.Certificate, 0, len(header.X5C))
	for _, encoded := range header.X5C {
		if len(encoded) > (maxAppleX5CCertBytes*4/3)+8 {
			return nil, errors.New("x5c certificate exceeds size limit")
		}
		der, decodeErr := base64.StdEncoding.DecodeString(encoded)
		if decodeErr != nil || len(der) == 0 || len(der) > maxAppleX5CCertBytes {
			return nil, errors.New("invalid x5c certificate")
		}
		cert, parseErr := x509.ParseCertificate(der)
		if parseErr != nil {
			return nil, errors.New("invalid x5c certificate")
		}
		certs = append(certs, cert)
	}
	intermediates := x509.NewCertPool()
	for _, cert := range certs[1:] {
		intermediates.AddCert(cert)
	}
	chains, err := certs[0].Verify(x509.VerifyOptions{
		Roots: v.roots, Intermediates: intermediates, CurrentTime: v.now(),
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	})
	if err != nil || len(chains) == 0 {
		return nil, fmt.Errorf("untrusted JWS certificate chain: %w", err)
	}
	trustedAppleRoot := false
	for _, chain := range chains {
		root := chain[len(chain)-1]
		if strings.HasPrefix(root.Subject.CommonName, "Apple Root") && appleRootOrgContains(root.Subject.Organization, "Apple Inc.") {
			trustedAppleRoot = true
			break
		}
	}
	if !trustedAppleRoot || !strings.Contains(strings.ToLower(certs[0].Issuer.String()), "apple") {
		return nil, errors.New("certificate chain is not an Apple signing chain")
	}
	pub, ok := certs[0].PublicKey.(*ecdsa.PublicKey)
	if !ok || pub.Curve == nil || pub.Curve.Params().Name != "P-256" {
		return nil, errors.New("unexpected JWS public key")
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || len(signature) != 64 {
		return nil, errors.New("invalid ES256 signature")
	}
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	r := new(big.Int).SetBytes(signature[:32])
	s := new(big.Int).SetBytes(signature[32:])
	if !ecdsa.Verify(pub, digest[:], r, s) {
		return nil, errors.New("JWS signature verification failed")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || len(payload) > maxAppleJWSPayloadBytes {
		return nil, errors.New("invalid JWS payload encoding")
	}
	return payload, nil
}

func appleRootOrgContains(values []string, expected string) bool {
	for _, value := range values {
		if strings.EqualFold(value, expected) {
			return true
		}
	}
	return false
}
