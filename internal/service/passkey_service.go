package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/tsc-forex/tsc_forex/internal/config"
	"github.com/tsc-forex/tsc_forex/internal/models"
	"github.com/tsc-forex/tsc_forex/internal/repository"
)

// Passkey errors.
var (
	ErrPasskeyDisabled = errors.New("passkey not configured")
	ErrPasskeySession  = errors.New("invalid or expired passkey session")
)

// PasskeyService wraps go-webauthn for passwordless registration and login.
// The WebAuthn SessionData (challenge) is round-tripped to the client inside a
// short-lived signed JWT, so the flow is stateless and works across pods.
type PasskeyService struct {
	wa        *webauthn.WebAuthn
	creds     repository.WebAuthnCredentialRepository
	auth      *AuthService
	jwtSecret []byte
	log       *zap.Logger
}

// NewPasskeyService builds the service. Returns an error when disabled/misconfigured.
func NewPasskeyService(cfg config.WebAuthnConfig, jwtSecret string, creds repository.WebAuthnCredentialRepository, auth *AuthService, log *zap.Logger) (*PasskeyService, error) {
	if !cfg.Enabled {
		return nil, ErrPasskeyDisabled
	}
	if cfg.RPID == "" || len(cfg.RPOrigins) == 0 {
		return nil, errors.New("webauthn rp_id/rp_origins required")
	}
	wa, err := webauthn.New(&webauthn.Config{
		RPDisplayName: cfg.RPDisplayName,
		RPID:          cfg.RPID,
		RPOrigins:     cfg.RPOrigins,
	})
	if err != nil {
		return nil, err
	}
	if log == nil {
		log = zap.NewNop()
	}
	return &PasskeyService{wa: wa, creds: creds, auth: auth, jwtSecret: []byte(jwtSecret), log: log}, nil
}

// webUser adapts our models to the webauthn.User interface.
type webUser struct {
	user  models.User
	creds []models.WebAuthnCredential
}

func (u webUser) WebAuthnID() []byte {
	b, _ := u.user.ID.MarshalBinary()
	return b
}
func (u webUser) WebAuthnName() string { return u.user.Email }
func (u webUser) WebAuthnDisplayName() string {
	if u.user.DisplayName != "" {
		return u.user.DisplayName
	}
	return u.user.Email
}
func (u webUser) WebAuthnCredentials() []webauthn.Credential {
	out := make([]webauthn.Credential, 0, len(u.creds))
	for _, c := range u.creds {
		var transports []protocol.AuthenticatorTransport
		for _, t := range strings.Split(c.Transports, ",") {
			if t != "" {
				transports = append(transports, protocol.AuthenticatorTransport(t))
			}
		}
		out = append(out, webauthn.Credential{
			ID:        c.CredentialID,
			PublicKey: c.PublicKey,
			Transport: transports,
			Authenticator: webauthn.Authenticator{
				AAGUID:    c.AAGUID,
				SignCount: c.SignCount,
			},
		})
	}
	return out
}

// --- session token (stateless challenge round-trip) ---

func (s *PasskeyService) encodeSession(sd *webauthn.SessionData) (string, error) {
	raw, err := json.Marshal(sd)
	if err != nil {
		return "", err
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sd":  string(raw),
		"exp": time.Now().Add(5 * time.Minute).Unix(),
	})
	return tok.SignedString(s.jwtSecret)
}

func (s *PasskeyService) decodeSession(token string) (*webauthn.SessionData, error) {
	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("bad alg")
		}
		return s.jwtSecret, nil
	})
	if err != nil {
		return nil, ErrPasskeySession
	}
	raw, _ := claims["sd"].(string)
	var sd webauthn.SessionData
	if err := json.Unmarshal([]byte(raw), &sd); err != nil {
		return nil, ErrPasskeySession
	}
	return &sd, nil
}

// --- registration (authenticated user adds a passkey) ---

// BeginRegistration returns credential-creation options + a session token.
func (s *PasskeyService) BeginRegistration(ctx context.Context, userID uuid.UUID) (*protocol.CredentialCreation, string, error) {
	u, err := s.auth.Users().ByID(ctx, userID)
	if err != nil {
		return nil, "", err
	}
	existing, _ := s.creds.ListByUser(ctx, userID)
	wu := webUser{user: u, creds: existing}

	sel := protocol.AuthenticatorSelection{
		ResidentKey:      protocol.ResidentKeyRequirementRequired,
		UserVerification: protocol.VerificationPreferred,
	}
	options, session, err := s.wa.BeginRegistration(wu,
		webauthn.WithAuthenticatorSelection(sel),
		webauthn.WithExclusions(credentialDescriptors(existing)),
	)
	if err != nil {
		return nil, "", err
	}
	tok, err := s.encodeSession(session)
	if err != nil {
		return nil, "", err
	}
	return options, tok, nil
}

// FinishRegistration verifies the attestation and persists the new credential.
func (s *PasskeyService) FinishRegistration(ctx context.Context, userID uuid.UUID, sessionToken string, body io.Reader) error {
	sd, err := s.decodeSession(sessionToken)
	if err != nil {
		return err
	}
	u, err := s.auth.Users().ByID(ctx, userID)
	if err != nil {
		return err
	}
	existing, _ := s.creds.ListByUser(ctx, userID)
	wu := webUser{user: u, creds: existing}

	parsed, err := protocol.ParseCredentialCreationResponseBody(body)
	if err != nil {
		return err
	}
	cred, err := s.wa.CreateCredential(wu, *sd, parsed)
	if err != nil {
		return err
	}
	rec := &models.WebAuthnCredential{
		UserID:       userID,
		CredentialID: cred.ID,
		PublicKey:    cred.PublicKey,
		AAGUID:       cred.Authenticator.AAGUID,
		SignCount:    cred.Authenticator.SignCount,
		Transports:   joinTransports(cred.Transport),
	}
	return s.creds.Create(ctx, rec)
}

// --- discoverable login (usernameless) ---

// BeginLogin returns assertion options + a session token (no user required).
func (s *PasskeyService) BeginLogin() (*protocol.CredentialAssertion, string, error) {
	options, session, err := s.wa.BeginDiscoverableLogin()
	if err != nil {
		return nil, "", err
	}
	tok, err := s.encodeSession(session)
	if err != nil {
		return nil, "", err
	}
	return options, tok, nil
}

// FinishLogin verifies the assertion, updates the sign counter and issues our
// own access+refresh session for the resolved user.
func (s *PasskeyService) FinishLogin(ctx context.Context, sessionToken string, body io.Reader) (access, refresh string, err error) {
	sd, err := s.decodeSession(sessionToken)
	if err != nil {
		return "", "", err
	}
	parsed, err := protocol.ParseCredentialRequestResponseBody(body)
	if err != nil {
		return "", "", err
	}

	var resolved models.User
	handler := func(rawID, userHandle []byte) (webauthn.User, error) {
		uid, uerr := uuid.FromBytes(userHandle)
		if uerr != nil {
			return nil, uerr
		}
		u, uerr := s.auth.Users().ByID(ctx, uid)
		if uerr != nil {
			return nil, uerr
		}
		creds, _ := s.creds.ListByUser(ctx, uid)
		resolved = u
		return webUser{user: u, creds: creds}, nil
	}

	cred, err := s.wa.ValidateDiscoverableLogin(handler, *sd, parsed)
	if err != nil {
		return "", "", err
	}
	// Persist the updated sign counter (clone detection).
	if stored, serr := s.creds.ByCredentialID(ctx, cred.ID); serr == nil {
		_ = s.creds.UpdateSignCount(ctx, stored.ID, cred.Authenticator.SignCount)
	}
	return s.auth.IssueSession(ctx, resolved)
}

func credentialDescriptors(creds []models.WebAuthnCredential) []protocol.CredentialDescriptor {
	out := make([]protocol.CredentialDescriptor, 0, len(creds))
	for _, c := range creds {
		out = append(out, protocol.CredentialDescriptor{
			Type:         protocol.PublicKeyCredentialType,
			CredentialID: c.CredentialID,
		})
	}
	return out
}

func joinTransports(ts []protocol.AuthenticatorTransport) string {
	parts := make([]string, 0, len(ts))
	for _, t := range ts {
		parts = append(parts, string(t))
	}
	return strings.Join(parts, ",")
}
