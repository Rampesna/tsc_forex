package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
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
	users  repository.UserRepository
	tokens repository.RefreshTokenRepository // optional
	cfg    config.JWTConfig
	oauth  config.OAuthConfig
	// googleVerify is swappable in tests; nil uses the real Google verifier.
	googleVerify func(ctx context.Context, idToken string, allowedAuds []string) (*GoogleIdentity, error)
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
