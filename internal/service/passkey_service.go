package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"sort"
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
	ErrPasskeyNotFound = errors.New("passkey not found")
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
			Flags: webauthn.CredentialFlags{
				BackupEligible: c.BackupEligible,
				BackupState:    c.BackupState,
			},
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
		s.log.Warn("passkey register: parse attestation failed", zap.Error(err))
		return err
	}
	cred, err := s.wa.CreateCredential(wu, *sd, parsed)
	if err != nil {
		s.log.Warn("passkey register: create credential failed",
			zap.String("rp_id", s.wa.Config.RPID), zap.Error(err))
		return err
	}
	rec := &models.WebAuthnCredential{
		UserID:         userID,
		CredentialID:   cred.ID,
		PublicKey:      cred.PublicKey,
		AAGUID:         cred.Authenticator.AAGUID,
		SignCount:      cred.Authenticator.SignCount,
		Transports:     joinTransports(cred.Transport),
		BackupEligible: cred.Flags.BackupEligible,
		BackupState:    cred.Flags.BackupState,
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
		s.log.Warn("passkey login: parse assertion failed", zap.Error(err))
		return "", "", err
	}

	// Backup Eligible/State are static properties of the credential. Legacy
	// credentials created before we captured these flags default to false in
	// our store, which would trip go-webauthn's consistency check against a
	// synced (iCloud/Google) passkey that always asserts BE=BS=true. Read the
	// live flags so we can reconcile the stored record and let login proceed.
	asrBE := parsed.Response.AuthenticatorData.Flags.HasBackupEligible()
	asrBS := parsed.Response.AuthenticatorData.Flags.HasBackupState()

	var (
		resolved      models.User
		reconcileID   uuid.UUID
		needReconcile bool
	)
	handler := func(rawID, userHandle []byte) (webauthn.User, error) {
		uid, uerr := uuid.FromBytes(userHandle)
		if uerr != nil {
			s.log.Warn("passkey login: bad user handle", zap.Int("len", len(userHandle)), zap.Error(uerr))
			return nil, uerr
		}
		u, uerr := s.auth.Users().ByID(ctx, uid)
		if uerr != nil {
			s.log.Warn("passkey login: user not found", zap.String("uid", uid.String()), zap.Error(uerr))
			return nil, uerr
		}
		creds, _ := s.creds.ListByUser(ctx, uid)
		// Reconcile the flags of the credential actually being used so the
		// consistency check passes; remember it to persist after validation.
		for i := range creds {
			if bytes.Equal(creds[i].CredentialID, rawID) {
				if creds[i].BackupEligible != asrBE || creds[i].BackupState != asrBS {
					reconcileID = creds[i].ID
					needReconcile = true
					creds[i].BackupEligible = asrBE
					creds[i].BackupState = asrBS
				}
			}
		}
		s.log.Info("passkey login: resolved user",
			zap.String("email", u.Email), zap.Int("stored_creds", len(creds)),
			zap.Bool("reconcile_flags", needReconcile))
		resolved = u
		return webUser{user: u, creds: creds}, nil
	}

	cred, err := s.wa.ValidateDiscoverableLogin(handler, *sd, parsed)
	if err != nil {
		s.log.Warn("passkey login: validate assertion failed",
			zap.String("rp_id", s.wa.Config.RPID), zap.Error(err))
		return "", "", err
	}
	// Persist the updated sign counter (clone detection).
	if stored, serr := s.creds.ByCredentialID(ctx, cred.ID); serr == nil {
		_ = s.creds.UpdateSignCount(ctx, stored.ID, cred.Authenticator.SignCount)
	}
	// Persist reconciled backup flags so future logins are consistent.
	if needReconcile {
		if uerr := s.creds.UpdateFlags(ctx, reconcileID, asrBE, asrBS); uerr != nil {
			s.log.Warn("passkey login: reconcile flags failed", zap.Error(uerr))
		} else {
			s.log.Info("passkey login: reconciled backup flags",
				zap.Bool("backup_eligible", asrBE), zap.Bool("backup_state", asrBS))
		}
	}
	return s.auth.IssueSession(ctx, resolved)
}

// PasskeyInfo is the client-facing view of a registered credential.
type PasskeyInfo struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Synced    bool      `json:"synced"` // BackupState: usable across the user's devices
	CreatedAt time.Time `json:"created_at"`
}

// ListCredentials returns the user's registered passkeys (newest first).
func (s *PasskeyService) ListCredentials(ctx context.Context, userID uuid.UUID) ([]PasskeyInfo, error) {
	rows, err := s.creds.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]PasskeyInfo, 0, len(rows))
	for _, c := range rows {
		out = append(out, PasskeyInfo{
			ID:        c.ID.String(),
			Name:      defaultName(c),
			Synced:    c.BackupState,
			CreatedAt: c.CreatedAt,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

// DeleteCredential removes a passkey the user owns. Returns ErrPasskeyNotFound
// when the id doesn't belong to the user.
func (s *PasskeyService) DeleteCredential(ctx context.Context, userID, id uuid.UUID) error {
	n, err := s.creds.DeleteByUser(ctx, userID, id)
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrPasskeyNotFound
	}
	return nil
}

// RenameCredential sets a user-friendly label for a passkey the user owns.
func (s *PasskeyService) RenameCredential(ctx context.Context, userID, id uuid.UUID, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("name required")
	}
	if len(name) > 80 {
		name = name[:80]
	}
	n, err := s.creds.RenameByUser(ctx, userID, id, name)
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrPasskeyNotFound
	}
	return nil
}

// defaultName falls back to a sensible label when the user hasn't named the key.
func defaultName(c models.WebAuthnCredential) string {
	if strings.TrimSpace(c.Name) != "" {
		return c.Name
	}
	if c.BackupState {
		return "Eşitlenen geçiş anahtarı"
	}
	return "Bu cihaz"
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
