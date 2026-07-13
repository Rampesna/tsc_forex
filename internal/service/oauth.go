package service

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"google.golang.org/api/idtoken"

	"github.com/tsc-forex/tsc_forex/internal/models"
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
	email := strings.ToLower(strings.TrimSpace(id.Email))

	u, err := s.users.ByEmail(ctx, email)
	if err != nil {
		u = models.User{
			ID:          uuid.New(),
			Email:       email,
			Role:        "user",
			Provider:    "google",
			DisplayName: id.Name,
			AvatarURL:   id.Picture,
		}
		if cerr := s.users.Create(ctx, &u); cerr != nil {
			return "", "", cerr
		}
		return s.issueSession(ctx, u)
	}

	// Existing account: backfill profile fields we don't have yet.
	changed := false
	if u.DisplayName == "" && id.Name != "" {
		u.DisplayName = id.Name
		changed = true
	}
	if u.AvatarURL == "" && id.Picture != "" {
		u.AvatarURL = id.Picture
		changed = true
	}
	if changed {
		_ = s.users.Update(ctx, &u)
	}
	return s.issueSession(ctx, u)
}

func containsFold(list []string, v string) bool {
	for _, item := range list {
		if strings.EqualFold(strings.TrimSpace(item), strings.TrimSpace(v)) {
			return true
		}
	}
	return false
}
