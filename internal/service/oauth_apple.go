package service

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/tsc-forex/tsc_forex/internal/models"
)

const appleIssuer = "https://appleid.apple.com"
const appleKeysURL = "https://appleid.apple.com/auth/keys"

// AppleIdentity is the trusted subset of a verified Apple identity token.
type AppleIdentity struct {
	Sub   string
	Email string
	Aud   string
}

// appleKeyCache caches Apple's JWKS with a short TTL (keys rotate rarely).
type appleKeyCache struct {
	mu      sync.Mutex
	keys    map[string]*rsa.PublicKey
	fetched time.Time
}

var appleKeys = &appleKeyCache{keys: map[string]*rsa.PublicKey{}}

func (c *appleKeyCache) get(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	c.mu.Lock()
	if k, ok := c.keys[kid]; ok && time.Since(c.fetched) < time.Hour {
		c.mu.Unlock()
		return k, nil
	}
	c.mu.Unlock()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, appleKeysURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	var doc struct {
		Keys []struct {
			Kid string `json:"kid"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return nil, err
	}
	parsed := make(map[string]*rsa.PublicKey, len(doc.Keys))
	for _, k := range doc.Keys {
		nb, err := base64.RawURLEncoding.DecodeString(k.N)
		if err != nil {
			continue
		}
		eb, err := base64.RawURLEncoding.DecodeString(k.E)
		if err != nil {
			continue
		}
		parsed[k.Kid] = &rsa.PublicKey{
			N: new(big.Int).SetBytes(nb),
			E: int(new(big.Int).SetBytes(eb).Int64()),
		}
	}
	c.mu.Lock()
	c.keys = parsed
	c.fetched = time.Now()
	c.mu.Unlock()

	if k, ok := parsed[kid]; ok {
		return k, nil
	}
	return nil, errors.New("apple signing key not found")
}

// verifyAppleToken validates an Apple identity token's signature (RS256 via
// Apple JWKS), issuer, expiry and audience against our allowed bundle IDs.
func verifyAppleToken(ctx context.Context, idToken string, allowedAuds []string) (*AppleIdentity, error) {
	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(idToken, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		kid, _ := t.Header["kid"].(string)
		return appleKeys.get(ctx, kid)
	}, jwt.WithIssuer(appleIssuer), jwt.WithValidMethods([]string{"RS256"}))
	if err != nil {
		return nil, ErrOAuthInvalidToken
	}

	aud, _ := claims["aud"].(string)
	if !containsFold(allowedAuds, aud) {
		return nil, ErrOAuthInvalidToken
	}
	sub, _ := claims["sub"].(string)
	if sub == "" {
		return nil, ErrOAuthInvalidToken
	}
	id := &AppleIdentity{Sub: sub, Aud: aud}
	if v, ok := claims["email"].(string); ok {
		id.Email = v
	}
	return id, nil
}

// LoginWithApple verifies an Apple identity token and upserts the user.
// Apple only returns the full name on the first authorization, so callers pass
// it separately (fullName) to seed the display name.
func (s *AuthService) LoginWithApple(ctx context.Context, idToken, fullName string) (access, refresh string, err error) {
	if len(s.oauth.AppleBundleIDs) == 0 {
		return "", "", ErrOAuthNotConfigured
	}
	verify := s.appleVerify
	if verify == nil {
		verify = verifyAppleToken
	}
	id, err := verify(ctx, idToken, s.oauth.AppleBundleIDs)
	if err != nil {
		return "", "", err
	}

	// Apple may hide the real email (private relay) or omit it on later logins;
	// fall back to a stable synthetic address keyed by the Apple subject.
	email := strings.ToLower(strings.TrimSpace(id.Email))
	if email == "" {
		email = "apple_" + id.Sub + "@users.tscforex"
	}

	u, err := s.users.ByEmail(ctx, email)
	if err != nil {
		u = models.User{
			ID:          uuid.New(),
			Email:       email,
			Role:        "user",
			Provider:    "apple",
			DisplayName: strings.TrimSpace(fullName),
		}
		if cerr := s.users.Create(ctx, &u); cerr != nil {
			return "", "", cerr
		}
		return s.issueSession(ctx, u)
	}

	if u.DisplayName == "" && strings.TrimSpace(fullName) != "" {
		u.DisplayName = strings.TrimSpace(fullName)
		_ = s.users.Update(ctx, &u)
	}
	return s.issueSession(ctx, u)
}
