package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/tsc-forex/tsc_forex/internal/config"
	"github.com/tsc-forex/tsc_forex/internal/models"
	"github.com/tsc-forex/tsc_forex/internal/repository"
)

const managerTokenAudience = "tsc-forex-manager"

var (
	ErrManagerForbidden     = errors.New("manager access required")
	ErrWeakManagerPassword  = errors.New("manager password does not meet policy")
	ErrPasswordUnchanged    = errors.New("new password must be different")
	ErrManagerNotConfigured = errors.New("manager bootstrap is not configured")
)

// ManagerAuthService is a separate authentication realm for the operations
// console. It never reads users or refresh_tokens.
type ManagerAuthService struct {
	managers repository.ManagerRepository
	sessions repository.ManagerSessionRepository
	cfg      config.JWTConfig
}

func NewManagerAuthService(
	managers repository.ManagerRepository,
	sessions repository.ManagerSessionRepository,
	cfg config.JWTConfig,
) *ManagerAuthService {
	return &ManagerAuthService{managers: managers, sessions: sessions, cfg: cfg}
}

// EnsureBootstrapManager creates the first manager exactly once. A later
// deployment never overwrites a password chosen in the console.
func EnsureBootstrapManager(
	ctx context.Context,
	db *gorm.DB,
	managers repository.ManagerRepository,
	cfg config.AdminConfig,
) error {
	if !cfg.Enabled {
		return nil
	}
	email := normalizeEmail(cfg.BootstrapEmail)
	if email == "" {
		return fmt.Errorf("%w: email is empty", ErrManagerNotConfigured)
	}
	manager, err := managers.ByEmail(ctx, email)
	if err != nil && !errors.Is(err, repository.ErrManagerNotFound) {
		return err
	}
	if errors.Is(err, repository.ErrManagerNotFound) {
		if err := validateManagerPassword(cfg.BootstrapPassword); err != nil {
			return fmt.Errorf("%w: bootstrap password is missing or weak", ErrManagerNotConfigured)
		}
		hash, err := HashPassword(cfg.BootstrapPassword)
		if err != nil {
			return err
		}
		manager = models.Manager{
			ID:                 uuid.New(),
			Email:              email,
			PasswordHash:       hash,
			DisplayName:        "Talha Can",
			MustChangePassword: true,
		}
		if err := managers.Create(ctx, &manager); err != nil {
			return err
		}
		// Another rollout pod may have won the unique-email insert. Always use
		// the durable row rather than the locally generated UUID.
		manager, err = managers.ByEmail(ctx, email)
		if err != nil {
			return err
		}
	}
	return detachLegacyManagerUser(ctx, db, manager)
}

// detachLegacyManagerUser is an idempotent one-time compatibility migration
// from the earlier user-backed admin implementation. Consumer data remains
// intact; only the role is normalized and historical manager ownership is
// pointed at the new managers row.
func detachLegacyManagerUser(ctx context.Context, db *gorm.DB, manager models.Manager) error {
	if db == nil {
		return errors.New("database is required for manager isolation")
	}
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var legacy models.User
		err := tx.Where("email = ?", manager.Email).Take(&legacy).Error
		if err == nil {
			if err := tx.Model(&models.NotificationCampaign{}).
				Where("manager_id = ?", legacy.ID).
				Update("manager_id", manager.ID).Error; err != nil {
				return err
			}
			if err := tx.Model(&models.AdminAuditLog{}).
				Where("manager_id = ?", legacy.ID).
				Update("manager_id", manager.ID).Error; err != nil {
				return err
			}
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		return tx.Model(&models.User{}).
			Where("role IS NULL OR role <> ?", "user").
			Update("role", "user").Error
	})
}

func (s *ManagerAuthService) Login(
	ctx context.Context,
	email, password string,
) (models.Manager, string, string, error) {
	manager, err := s.managers.ByEmail(ctx, normalizeEmail(email))
	if err != nil || VerifyPassword(manager.PasswordHash, password) != nil {
		return models.Manager{}, "", "", ErrInvalidCredentials
	}
	now := time.Now().UTC()
	manager.LastLoginAt = &now
	if err := s.managers.Update(ctx, &manager); err != nil {
		return models.Manager{}, "", "", err
	}
	access, refresh, err := s.issueSession(ctx, manager)
	return manager, access, refresh, err
}

func (s *ManagerAuthService) Refresh(
	ctx context.Context,
	refreshToken string,
) (models.Manager, string, string, error) {
	session, err := s.sessions.ByHash(ctx, hashToken(refreshToken))
	if err != nil || session.RevokedAt != nil || time.Now().After(session.ExpiresAt) {
		return models.Manager{}, "", "", ErrTokenRevoked
	}
	manager, err := s.managers.ByID(ctx, session.ManagerID)
	if err != nil {
		_ = s.sessions.Revoke(ctx, session.ID)
		return models.Manager{}, "", "", ErrManagerForbidden
	}
	if err := s.sessions.Revoke(ctx, session.ID); err != nil {
		return models.Manager{}, "", "", err
	}
	access, refresh, err := s.issueSession(ctx, manager)
	return manager, access, refresh, err
}

func (s *ManagerAuthService) Logout(ctx context.Context, refreshToken string) error {
	session, err := s.sessions.ByHash(ctx, hashToken(refreshToken))
	if err != nil {
		return nil
	}
	return s.sessions.Revoke(ctx, session.ID)
}

func (s *ManagerAuthService) Manager(ctx context.Context, managerID uuid.UUID) (models.Manager, error) {
	manager, err := s.managers.ByID(ctx, managerID)
	if err != nil {
		return models.Manager{}, ErrManagerForbidden
	}
	return manager, nil
}

func (s *ManagerAuthService) ChangePassword(
	ctx context.Context,
	managerID uuid.UUID,
	current, next string,
) (models.Manager, error) {
	if err := validateManagerPassword(next); err != nil {
		return models.Manager{}, err
	}
	manager, err := s.Manager(ctx, managerID)
	if err != nil {
		return models.Manager{}, err
	}
	if VerifyPassword(manager.PasswordHash, current) != nil {
		return models.Manager{}, ErrInvalidCredentials
	}
	if VerifyPassword(manager.PasswordHash, next) == nil {
		return models.Manager{}, ErrPasswordUnchanged
	}
	hash, err := HashPassword(next)
	if err != nil {
		return models.Manager{}, err
	}
	now := time.Now().UTC()
	manager.PasswordHash = hash
	manager.MustChangePassword = false
	manager.PasswordChangedAt = &now
	if err := s.managers.Update(ctx, &manager); err != nil {
		return models.Manager{}, err
	}
	if err := s.sessions.RevokeAllForManager(ctx, manager.ID); err != nil {
		return models.Manager{}, err
	}
	return manager, nil
}

func (s *ManagerAuthService) IssueSession(
	ctx context.Context,
	manager models.Manager,
) (string, string, error) {
	return s.issueSession(ctx, manager)
}

func (s *ManagerAuthService) issueSession(
	ctx context.Context,
	manager models.Manager,
) (string, string, error) {
	access, err := s.issue(manager, s.cfg.AccessTokenTTL)
	if err != nil {
		return "", "", err
	}
	raw, err := randomToken()
	if err != nil {
		return "", "", err
	}
	session := &models.ManagerSession{
		ID:        uuid.New(),
		ManagerID: manager.ID,
		TokenHash: hashToken(raw),
		ExpiresAt: time.Now().Add(s.cfg.RefreshTokenTTL),
		CreatedAt: time.Now().UTC(),
	}
	if err := s.sessions.Store(ctx, session); err != nil {
		return "", "", err
	}
	return access, raw, nil
}

func (s *ManagerAuthService) issue(manager models.Manager, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: manager.ID.String(),
		Email:  manager.Email,
		Role:   "manager",
		Kind:   "manager",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.cfg.Issuer,
			Subject:   manager.ID.String(),
			Audience:  jwt.ClaimStrings{managerTokenAudience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.Secret))
}

func (s *ManagerAuthService) Verify(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(s.cfg.Secret), nil
	}, jwt.WithIssuer(s.cfg.Issuer), jwt.WithAudience(managerTokenAudience))
	if err != nil || !token.Valid || claims.Kind != "manager" || claims.Role != "manager" {
		return nil, ErrInvalidCredentials
	}
	return claims, nil
}

func validateManagerPassword(password string) error {
	if len([]rune(password)) < 12 || len(password) > 128 {
		return ErrWeakManagerPassword
	}
	var lower, upper, digit, symbol bool
	for _, r := range password {
		switch {
		case unicode.IsLower(r):
			lower = true
		case unicode.IsUpper(r):
			upper = true
		case unicode.IsDigit(r):
			digit = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			symbol = true
		}
	}
	if !lower || !upper || !digit || !symbol || strings.TrimSpace(password) == "" {
		return ErrWeakManagerPassword
	}
	return nil
}
