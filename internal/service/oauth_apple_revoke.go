package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	appleTokenURL  = "https://appleid.apple.com/auth/token"
	appleRevokeURL = "https://appleid.apple.com/auth/revoke"
)

var (
	ErrAppleRevocationNotConfigured = errors.New("apple token revocation not configured")
	ErrAppleAuthorizationMismatch   = errors.New("apple authorization belongs to another account")
	ErrAppleRevocationFailed        = errors.New("apple token revocation failed")
)

// AppleRevocationReady is an operational/App Review readiness signal. It
// validates that Team ID, Key ID, client ID and the mounted ES256 private key
// can produce a client secret without exposing any credential detail.
func (s *AuthService) AppleRevocationReady() bool {
	if s == nil {
		return false
	}
	_, err := s.appleClientSecret()
	return err == nil
}

// RevokeAppleAuthorizationCode exchanges the short-lived code produced by a
// fresh native Sign in with Apple sheet, verifies that it belongs to the Apple
// identity linked to userID, then revokes the returned refresh/access token.
// Account deletion must stop if any step fails.
func (s *AuthService) RevokeAppleAuthorizationCode(ctx context.Context, userID uuid.UUID, authorizationCode string) error {
	authorizationCode = strings.TrimSpace(authorizationCode)
	if authorizationCode == "" || s.identities == nil {
		return ErrAppleAuthorizationMismatch
	}
	clientSecret, err := s.appleClientSecret()
	if err != nil {
		return err
	}
	form := url.Values{
		"client_id":     {s.oauth.AppleRevokeClientID},
		"client_secret": {clientSecret},
		"code":          {authorizationCode},
		"grant_type":    {"authorization_code"},
	}
	var tokenResponse struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		IDToken      string `json:"id_token"`
		Error        string `json:"error"`
	}
	if err := postAppleForm(ctx, appleTokenURL, form, &tokenResponse); err != nil {
		return fmt.Errorf("%w: exchange", ErrAppleRevocationFailed)
	}
	identity, err := verifyAppleToken(ctx, tokenResponse.IDToken, s.oauth.AppleBundleIDs)
	if err != nil || !s.appleIdentityBelongsTo(ctx, userID, identity.Sub) {
		return ErrAppleAuthorizationMismatch
	}
	token := tokenResponse.RefreshToken
	hint := "refresh_token"
	if token == "" {
		token = tokenResponse.AccessToken
		hint = "access_token"
	}
	if token == "" {
		return ErrAppleRevocationFailed
	}
	revokeForm := url.Values{
		"client_id":       {s.oauth.AppleRevokeClientID},
		"client_secret":   {clientSecret},
		"token":           {token},
		"token_type_hint": {hint},
	}
	if err := postAppleForm(ctx, appleRevokeURL, revokeForm, nil); err != nil {
		return fmt.Errorf("%w: revoke", ErrAppleRevocationFailed)
	}
	return nil
}

func (s *AuthService) appleIdentityBelongsTo(ctx context.Context, userID uuid.UUID, subject string) bool {
	identities, err := s.identities.ListByUser(ctx, userID)
	if err != nil {
		return false
	}
	for _, identity := range identities {
		if identity.Provider == "apple" && identity.Subject == subject {
			return true
		}
	}
	return false
}

func (s *AuthService) appleClientSecret() (string, error) {
	if strings.TrimSpace(s.oauth.AppleTeamID) == "" || strings.TrimSpace(s.oauth.AppleKeyID) == "" ||
		strings.TrimSpace(s.oauth.AppleRevokeClientID) == "" || strings.TrimSpace(s.oauth.ApplePrivateKeyPath) == "" {
		return "", ErrAppleRevocationNotConfigured
	}
	pemBytes, err := os.ReadFile(s.oauth.ApplePrivateKeyPath)
	if err != nil {
		return "", ErrAppleRevocationNotConfigured
	}
	privateKey, err := jwt.ParseECPrivateKeyFromPEM(pemBytes)
	if err != nil {
		return "", ErrAppleRevocationNotConfigured
	}
	now := time.Now().UTC()
	claims := jwt.RegisteredClaims{
		Issuer: s.oauth.AppleTeamID, Subject: s.oauth.AppleRevokeClientID,
		Audience: jwt.ClaimStrings{appleIssuer}, IssuedAt: jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(5 * time.Minute)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["kid"] = s.oauth.AppleKeyID
	signed, err := token.SignedString(privateKey)
	if err != nil {
		return "", ErrAppleRevocationNotConfigured
	}
	return signed, nil
}

func postAppleForm(ctx context.Context, endpoint string, form url.Values, output any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := &http.Client{Timeout: 15 * time.Second}
	response, err := client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	limited := io.LimitReader(response.Body, 1<<20)
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, limited)
		return fmt.Errorf("apple status %d", response.StatusCode)
	}
	if output == nil {
		_, _ = io.Copy(io.Discard, limited)
		return nil
	}
	return json.NewDecoder(limited).Decode(output)
}
