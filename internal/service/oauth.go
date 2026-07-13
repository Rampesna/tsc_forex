package service

import (
	"context"
	"errors"
	"strings"

	"google.golang.org/api/idtoken"
)

// Social-login errors.
var (
	ErrOAuthNotConfigured   = errors.New("social login not configured")
	ErrOAuthInvalidToken    = errors.New("invalid identity token")
	ErrOAuthEmailUnverified = errors.New("email not verified by provider")
)

// GoogleIdentity is the trusted subset of a verified Google ID token.
type GoogleIdentity struct {
	Sub     string
	Email   string
	Name    string
	Picture string
	Aud     string
}

// verifyGoogleToken validates the ID token's signature, expiry and issuer
// against Google's rotating public certs, then ensures the audience matches
// one of our configured client IDs (iOS / Android / Web).
func verifyGoogleToken(ctx context.Context, idToken string, allowedAuds []string) (*GoogleIdentity, error) {
	// audience="" → library skips its single-audience check; we verify the
	// aud against our allowed set below (we accept several clients).
	payload, err := idtoken.Validate(ctx, idToken, "")
	if err != nil {
		return nil, ErrOAuthInvalidToken
	}
	if !containsFold(allowedAuds, payload.Audience) {
		return nil, ErrOAuthInvalidToken
	}
	id := &GoogleIdentity{Sub: payload.Subject, Aud: payload.Audience}
	if v, ok := payload.Claims["email"].(string); ok {
		id.Email = v
	}
	if v, ok := payload.Claims["email_verified"].(bool); ok && !v {
		return nil, ErrOAuthEmailUnverified
	}
	if v, ok := payload.Claims["name"].(string); ok {
		id.Name = v
	}
	if v, ok := payload.Claims["picture"].(string); ok {
		id.Picture = v
	}
	if id.Email == "" {
		return nil, ErrOAuthInvalidToken
	}
	return id, nil
}

// LoginWithGoogle verifies a Google ID token and upserts the user, returning a
// fresh access + refresh session. New accounts are created as provider=google.
func (s *AuthService) LoginWithGoogle(ctx context.Context, idToken string) (access, refresh string, err error) {
	if len(s.oauth.GoogleClientIDs) == 0 {
		return "", "", ErrOAuthNotConfigured
	}
	verify := s.googleVerify
	if verify == nil {
		verify = verifyGoogleToken
	}
	id, err := verify(ctx, idToken, s.oauth.GoogleClientIDs)
	if err != nil {
		return "", "", err
	}
	u, err := s.resolveSocialUser(ctx, "google", id.Sub, id.Email, id.Name, id.Picture)
	if err != nil {
		return "", "", err
	}
	return s.issueSession(ctx, u)
}

// VerifyGoogle validates a Google ID token and returns the trusted identity,
// used by the authenticated "link account" flow.
func (s *AuthService) VerifyGoogle(ctx context.Context, idToken string) (*GoogleIdentity, error) {
	if len(s.oauth.GoogleClientIDs) == 0 {
		return nil, ErrOAuthNotConfigured
	}
	verify := s.googleVerify
	if verify == nil {
		verify = verifyGoogleToken
	}
	return verify(ctx, idToken, s.oauth.GoogleClientIDs)
}

func containsFold(list []string, v string) bool {
	for _, item := range list {
		if strings.EqualFold(strings.TrimSpace(item), strings.TrimSpace(v)) {
			return true
		}
	}
	return false
}
