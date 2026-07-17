package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/tsc-forex/tsc_forex/internal/config"
	"github.com/tsc-forex/tsc_forex/internal/models"
	"github.com/tsc-forex/tsc_forex/internal/repository"
)

// Auth-related errors.
var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrEmailTaken         = errors.New("email already registered")
	ErrTokenRevoked       = errors.New("refresh token revoked or expired")
	ErrNoTokenStore       = errors.New("refresh token store not configured")
)

// Claims is the JWT payload for authenticated users.
type Claims struct {
	UserID string `json:"uid"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// AuthService handles registration, login, and JWT issuing/verification.
// When a RefreshTokenRepository is attached, refresh tokens become opaque,
// stored hashed, and rotated on every refresh (revocable per-session).
type AuthService struct {
	users      repository.UserRepository
	tokens     repository.RefreshTokenRepository // optional
	identities repository.AuthIdentityRepository // optional: multi-provider linking
	cfg        config.JWTConfig
	oauth      config.OAuthConfig
	// googleVerify/appleVerify are swappable in tests; nil uses the real verifier.
	googleVerify func(ctx context.Context, idToken string, allowedAuds []string) (*GoogleIdentity, error)
	appleVerify  func(ctx context.Context, idToken string, allowedAuds []string) (*AppleIdentity, error)
}

// NewAuthService builds the auth service.
func NewAuthService(users repository.UserRepository, cfg config.JWTConfig) *AuthService {
	return &AuthService{users: users, cfg: cfg}
}

// WithRefreshStore enables persistent, rotating, revocable refresh tokens.
func (s *AuthService) WithRefreshStore(tokens repository.RefreshTokenRepository) *AuthService {
	s.tokens = tokens
	return s
}

// WithOAuth enables social login by supplying the accepted client IDs.
func (s *AuthService) WithOAuth(cfg config.OAuthConfig) *AuthService {
	s.oauth = cfg
	return s
}

// WithIdentities enables multi-provider account linking (Google + Apple on one
// account, including Apple private-relay emails).
func (s *AuthService) WithIdentities(ids repository.AuthIdentityRepository) *AuthService {
	s.identities = ids
	return s
}

// ErrProviderLinkedElsewhere is returned when linking an identity that already
// belongs to a different account.
var ErrProviderLinkedElsewhere = errors.New("this provider account is already linked to another user")

// resolveSocialUser maps a verified provider identity to a local user, creating
// or linking as needed:
//  1. known (provider, subject) → that user;
//  2. else known email → reuse & link this provider to it;
//  3. else create a fresh account and record the identity.
func (s *AuthService) resolveSocialUser(ctx context.Context, provider, subject, email, name, avatar string) (models.User, error) {
	email = normalizeEmail(email)

	// 1) Resolve by provider identity (stable across email changes / relay).
	if s.identities != nil && subject != "" {
		if idn, err := s.identities.ByProviderSubject(ctx, provider, subject); err == nil {
			if u, uerr := s.users.ByID(ctx, idn.UserID); uerr == nil {
				s.backfillProfile(ctx, &u, name, avatar)
				return u, nil
			}
		}
	}

	// 2) Resolve by email — reuse the existing account and link this provider.
	if u, err := s.users.ByEmail(ctx, email); err == nil {
		s.linkIdentity(ctx, u.ID, provider, subject, email)
		s.backfillProfile(ctx, &u, name, avatar)
		return u, nil
	}

	// 3) Brand-new account.
	u := models.User{
		ID:          uuid.New(),
		Email:       email,
		Role:        "user",
		Provider:    provider,
		DisplayName: strings.TrimSpace(name),
		AvatarURL:   avatar,
	}
	if err := s.users.Create(ctx, &u); err != nil {
		return models.User{}, err
	}
	s.linkIdentity(ctx, u.ID, provider, subject, email)
	return u, nil
}

func (s *AuthService) linkIdentity(ctx context.Context, userID uuid.UUID, provider, subject, email string) {
	if s.identities == nil || subject == "" {
		return
	}
	if _, err := s.identities.ByProviderSubject(ctx, provider, subject); err == nil {
		return // already recorded
	}
	_ = s.identities.Create(ctx, &models.AuthIdentity{
		UserID:   userID,
		Provider: provider,
		Subject:  subject,
		Email:    email,
	})
}

func (s *AuthService) backfillProfile(ctx context.Context, u *models.User, name, avatar string) {
	changed := false
	if u.DisplayName == "" && strings.TrimSpace(name) != "" {
		u.DisplayName = strings.TrimSpace(name)
		changed = true
	}
	if u.AvatarURL == "" && avatar != "" {
		u.AvatarURL = avatar
		changed = true
	}
	if changed {
		_ = s.users.Update(ctx, u)
	}
}

// LinkProvider attaches a verified provider identity to an already
// authenticated user. Used by the in-app "connect account" flow so a user who
// signed in with Google can also sign in with Apple (and vice versa).
func (s *AuthService) LinkProvider(ctx context.Context, userID uuid.UUID, provider, subject, email string) error {
	if s.identities == nil {
		return ErrNoTokenStore
	}
	if idn, err := s.identities.ByProviderSubject(ctx, provider, subject); err == nil {
		if idn.UserID != userID {
			return ErrProviderLinkedElsewhere
		}
		return nil // idempotent
	}
	return s.identities.Create(ctx, &models.AuthIdentity{
		UserID:   userID,
		Provider: provider,
		Subject:  subject,
		Email:    normalizeEmail(email),
	})
}

// LinkedProviders lists the providers currently bound to a user (for profile).
func (s *AuthService) LinkedProviders(ctx context.Context, userID uuid.UUID) []string {
	seen := map[string]bool{}
	out := []string{}
	if u, err := s.users.ByID(ctx, userID); err == nil && u.Provider != "" {
		seen[u.Provider] = true
		out = append(out, u.Provider)
	}
	if s.identities != nil {
		if ids, err := s.identities.ListByUser(ctx, userID); err == nil {
			for _, idn := range ids {
				if !seen[idn.Provider] {
					seen[idn.Provider] = true
					out = append(out, idn.Provider)
				}
			}
		}
	}
	return out
}

// HasAppleIdentity is the fail-closed policy lookup used before destructive
// account deletion. Unlike LinkedProviders (display-only), repository errors
// are returned so an outage can never silently skip Apple's token revocation.
func (s *AuthService) HasAppleIdentity(ctx context.Context, userID uuid.UUID) (bool, error) {
	user, err := s.users.ByID(ctx, userID)
	if err != nil {
		return false, err
	}
	if user.Provider == "apple" {
		return true, nil
	}
	if s.identities == nil {
		return false, nil
	}
	identities, err := s.identities.ListByUser(ctx, userID)
	if err != nil {
		return false, err
	}
	for _, identity := range identities {
		if identity.Provider == "apple" {
			return true, nil
		}
	}
	return false, nil
}

func normalizeEmail(e string) string { return strings.ToLower(strings.TrimSpace(e)) }

// IssueSession mints a fresh access+refresh pair for a known user. Used by
// passwordless flows (passkey) after the user has been authenticated elsewhere.
func (s *AuthService) IssueSession(ctx context.Context, u models.User) (access, refresh string, err error) {
	return s.issueSession(ctx, u)
}

// Users exposes the user repository for passwordless flows that must upsert or
// look up accounts (e.g. passkey registration/login).
func (s *AuthService) Users() repository.UserRepository { return s.users }

// Register creates a new user with an argon2id-hashed password.
func (s *AuthService) Register(ctx context.Context, email, password, role string) (models.User, error) {
	if role == "" {
		role = "user"
	}
	if _, err := s.users.ByEmail(ctx, email); err == nil {
		return models.User{}, ErrEmailTaken
	}
	hash, err := HashPassword(password)
	if err != nil {
		return models.User{}, err
	}
	u := &models.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: hash,
		Role:         role,
	}
	if err := s.users.Create(ctx, u); err != nil {
		return models.User{}, err
	}
	return *u, nil
}

// Login verifies credentials and returns access + refresh tokens.
func (s *AuthService) Login(ctx context.Context, email, password string) (access, refresh string, err error) {
	u, err := s.users.ByEmail(ctx, email)
	if err != nil {
		return "", "", ErrInvalidCredentials
	}
	if err := VerifyPassword(u.PasswordHash, password); err != nil {
		return "", "", ErrInvalidCredentials
	}
	return s.issueSession(ctx, u)
}

// Refresh validates a refresh token, rotates it (revoking the old), and
// returns a fresh access + refresh pair. Requires a token store.
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (access, refresh string, err error) {
	if s.tokens == nil {
		return "", "", ErrNoTokenStore
	}
	rec, err := s.tokens.ByHash(ctx, hashToken(refreshToken))
	if err != nil {
		return "", "", ErrTokenRevoked
	}
	if rec.RevokedAt != nil || time.Now().After(rec.ExpiresAt) {
		return "", "", ErrTokenRevoked
	}
	usr, err := s.userByID(ctx, rec.UserID)
	if err != nil {
		return "", "", ErrInvalidCredentials
	}
	_ = s.tokens.Revoke(ctx, rec.ID)
	return s.issueSession(ctx, usr)
}

// Logout revokes a specific refresh token (best-effort).
func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	if s.tokens == nil {
		return nil
	}
	rec, err := s.tokens.ByHash(ctx, hashToken(refreshToken))
	if err != nil {
		return nil
	}
	return s.tokens.Revoke(ctx, rec.ID)
}

// issueSession mints an access JWT plus a refresh token. When a token store is
// present the refresh token is opaque and persisted (hashed); otherwise a JWT
// refresh token is returned (stateless fallback).
func (s *AuthService) issueSession(ctx context.Context, u models.User) (access, refresh string, err error) {
	access, err = s.issue(u, s.cfg.AccessTokenTTL)
	if err != nil {
		return "", "", err
	}
	if s.tokens == nil {
		refresh, err = s.issue(u, s.cfg.RefreshTokenTTL)
		return access, refresh, err
	}
	raw, err := randomToken()
	if err != nil {
		return "", "", err
	}
	rec := &models.RefreshToken{
		ID:        uuid.New(),
		UserID:    u.ID,
		TokenHash: hashToken(raw),
		ExpiresAt: time.Now().Add(s.cfg.RefreshTokenTTL),
		CreatedAt: time.Now().UTC(),
	}
	if err := s.tokens.Store(ctx, rec); err != nil {
		return "", "", err
	}
	return access, raw, nil
}

// userByID fetches a user through the repository.
func (s *AuthService) userByID(ctx context.Context, id uuid.UUID) (models.User, error) {
	return s.users.ByID(ctx, id)
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func (s *AuthService) issue(u models.User, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: u.ID.String(),
		Email:  u.Email,
		Role:   u.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.cfg.Issuer,
			Subject:   u.ID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.Secret))
}

// Verify parses and validates a token, returning its claims.
func (s *AuthService) Verify(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(s.cfg.Secret), nil
	}, jwt.WithIssuer(s.cfg.Issuer))
	if err != nil || !token.Valid {
		return nil, ErrInvalidCredentials
	}
	return claims, nil
}
