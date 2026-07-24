package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"google.golang.org/api/androidpublisher/v3"
	"google.golang.org/api/option"

	"github.com/tsc-forex/tsc_forex/internal/config"
)

// GooglePlayPurchaseInfo is the normalized result of a Play Developer API lookup.
type GooglePlayPurchaseInfo struct {
	PackageName         string
	ProductID           string
	PurchaseToken       string
	OrderID             string
	LinkedPurchaseToken string
	ObfuscatedAccountID string
	Environment         string
	PurchasedAt         time.Time
	ExpiresAt           time.Time
	RevokedAt           *time.Time
	Acknowledged        bool
	RawJSON             []byte
}

// GooglePlayPurchaseVerifier verifies a Play Billing purchase token server-side.
type GooglePlayPurchaseVerifier interface {
	VerifySubscription(ctx context.Context, packageName, productID, purchaseToken string) (GooglePlayPurchaseInfo, error)
	Ready() bool
}

type googlePlayAPIVerifier struct {
	service     *androidpublisher.Service
	packageName string
	allowTest   bool
}

// NewGooglePlayVerifier builds a Play Developer API client from a service-account
// JSON file. When credentials are missing the verifier is nil/not ready and
// Google sync stays disabled without affecting Apple billing.
func NewGooglePlayVerifier(cfg config.BillingConfig) (GooglePlayPurchaseVerifier, error) {
	path := strings.TrimSpace(cfg.GoogleCredentialsPath)
	if path == "" {
		return nil, nil
	}
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("google play credentials: %w", err)
	}
	svc, err := androidpublisher.NewService(context.Background(), option.WithCredentialsFile(path))
	if err != nil {
		return nil, err
	}
	pkg := strings.TrimSpace(cfg.GooglePackageName)
	if pkg == "" {
		pkg = "com.talhacan.tscforex"
	}
	return &googlePlayAPIVerifier{service: svc, packageName: pkg, allowTest: cfg.AllowSandbox}, nil
}

func (v *googlePlayAPIVerifier) Ready() bool { return v != nil && v.service != nil }

func (v *googlePlayAPIVerifier) VerifySubscription(ctx context.Context, packageName, productID, purchaseToken string) (GooglePlayPurchaseInfo, error) {
	if !v.Ready() {
		return GooglePlayPurchaseInfo{}, ErrGoogleBillingUnavailable
	}
	pkg := strings.TrimSpace(packageName)
	if pkg == "" {
		pkg = v.packageName
	}
	if pkg != v.packageName {
		return GooglePlayPurchaseInfo{}, fmt.Errorf("%w: package mismatch", ErrInvalidGooglePurchase)
	}
	token := strings.TrimSpace(purchaseToken)
	productID = strings.TrimSpace(productID)
	if token == "" || productID == "" {
		return GooglePlayPurchaseInfo{}, ErrInvalidGooglePurchase
	}

	sub, err := v.service.Purchases.Subscriptionsv2.Get(pkg, token).Context(ctx).Do()
	if err != nil {
		return GooglePlayPurchaseInfo{}, fmt.Errorf("%w: %v", ErrInvalidGooglePurchase, err)
	}
	raw, _ := json.Marshal(sub)

	info := GooglePlayPurchaseInfo{
		PackageName:   pkg,
		PurchaseToken: token,
		OrderID:       strings.TrimSpace(sub.LatestOrderId),
		RawJSON:       raw,
		Acknowledged:  strings.EqualFold(sub.AcknowledgementState, "ACKNOWLEDGEMENT_STATE_ACKNOWLEDGED"),
	}
	if sub.ExternalAccountIdentifiers != nil {
		info.ObfuscatedAccountID = strings.TrimSpace(sub.ExternalAccountIdentifiers.ObfuscatedExternalAccountId)
	}
	if sub.TestPurchase != nil {
		info.Environment = "Sandbox"
	} else {
		info.Environment = "Production"
	}
	if !v.allowTest && strings.EqualFold(info.Environment, "Sandbox") {
		return GooglePlayPurchaseInfo{}, fmt.Errorf("%w: sandbox purchase rejected", ErrInvalidGooglePurchase)
	}

	switch strings.ToUpper(strings.TrimSpace(sub.SubscriptionState)) {
	case "SUBSCRIPTION_STATE_ACTIVE", "SUBSCRIPTION_STATE_IN_GRACE_PERIOD", "SUBSCRIPTION_STATE_ON_HOLD":
		// keep active path
	case "SUBSCRIPTION_STATE_CANCELED":
		// still entitled until expiry
	case "SUBSCRIPTION_STATE_EXPIRED", "SUBSCRIPTION_STATE_PENDING":
		// still parse times; apply layer decides status
	case "SUBSCRIPTION_STATE_PAUSED":
		// paused: treat as non-active at apply time via expiry/status
	default:
		if sub.SubscriptionState != "" && !strings.Contains(strings.ToUpper(sub.SubscriptionState), "ACTIVE") &&
			!strings.Contains(strings.ToUpper(sub.SubscriptionState), "CANCELED") &&
			!strings.Contains(strings.ToUpper(sub.SubscriptionState), "GRACE") {
			// Unknown states still proceed if we have a usable line item.
		}
	}

	if start, err := time.Parse(time.RFC3339, strings.TrimSpace(sub.StartTime)); err == nil {
		info.PurchasedAt = start.UTC()
	}
	var matched bool
	for _, item := range sub.LineItems {
		if item == nil {
			continue
		}
		if strings.TrimSpace(item.ProductId) != productID {
			continue
		}
		matched = true
		info.ProductID = productID
		if exp, err := time.Parse(time.RFC3339, strings.TrimSpace(item.ExpiryTime)); err == nil {
			info.ExpiresAt = exp.UTC()
		}
		if item.PrepaidPlan != nil && info.ExpiresAt.IsZero() {
			if exp, err := time.Parse(time.RFC3339, strings.TrimSpace(item.ExpiryTime)); err == nil {
				info.ExpiresAt = exp.UTC()
			}
		}
		break
	}
	if !matched {
		// Fall back to the first line item when the client product_id matches none
		// (rare race during upgrade). Prefer rejecting over granting the wrong plan.
		return GooglePlayPurchaseInfo{}, fmt.Errorf("%w: product_id not in subscription", ErrInvalidGooglePurchase)
	}
	if info.PurchasedAt.IsZero() {
		info.PurchasedAt = time.Now().UTC()
	}
	if info.ExpiresAt.IsZero() {
		return GooglePlayPurchaseInfo{}, fmt.Errorf("%w: missing expiry", ErrInvalidGooglePurchase)
	}
	if strings.EqualFold(sub.SubscriptionState, "SUBSCRIPTION_STATE_EXPIRED") {
		revoked := info.ExpiresAt
		info.RevokedAt = &revoked
	}
	return info, nil
}
