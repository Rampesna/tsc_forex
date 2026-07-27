package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/tsc-forex/tsc_forex/internal/config"
	"github.com/tsc-forex/tsc_forex/internal/models"
)

var (
	ErrAdminForbidden     = errors.New("admin access required")
	ErrWeakAdminPassword  = errors.New("admin password does not meet policy")
	ErrPasswordUnchanged  = errors.New("new password must be different")
	ErrAdminNotConfigured = errors.New("admin bootstrap is not configured")
)

// EnsureBootstrapAdmin creates the first administrator exactly once. Existing
// passwords are never reset by a deployment; the bootstrap secret can safely
// remain mounted until the administrator changes it in the console.
func EnsureBootstrapAdmin(ctx context.Context, db *gorm.DB, cfg config.AdminConfig) error {
	if !cfg.Enabled {
		return nil
	}
	email := normalizeEmail(cfg.BootstrapEmail)
	if email == "" {
		return fmt.Errorf("%w: email is empty", ErrAdminNotConfigured)
	}

	var user models.User
	err := db.WithContext(ctx).Where("email = ?", email).Take(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if err := validateAdminPassword(cfg.BootstrapPassword); err != nil {
			return fmt.Errorf("%w: bootstrap password is missing or weak", ErrAdminNotConfigured)
		}
		hash, err := HashPassword(cfg.BootstrapPassword)
		if err != nil {
			return err
		}
		user = models.User{
			ID:                 uuid.New(),
			Email:              email,
			PasswordHash:       hash,
			Role:               "admin",
			Provider:           "local",
			DisplayName:        "Talha Can",
			MustChangePassword: true,
		}
		return db.WithContext(ctx).Create(&user).Error
	}
	if err != nil {
		return err
	}

	updates := map[string]any{}
	if user.Role != "admin" {
		// Never turn an arbitrary pre-existing consumer account into an
		// administrator merely because its email matches configuration. The
		// bootstrap secret must also prove control of that local account.
		if strings.TrimSpace(user.PasswordHash) != "" {
			if err := VerifyPassword(user.PasswordHash, cfg.BootstrapPassword); err != nil {
				return fmt.Errorf("%w: existing account requires an explicit administrative promotion", ErrAdminNotConfigured)
			}
		}
		updates["role"] = "admin"
		updates["must_change_password"] = true
	}
	if strings.TrimSpace(user.PasswordHash) == "" {
		if err := validateAdminPassword(cfg.BootstrapPassword); err != nil {
			return fmt.Errorf("%w: existing account has no local password", ErrAdminNotConfigured)
		}
		hash, hashErr := HashPassword(cfg.BootstrapPassword)
		if hashErr != nil {
			return hashErr
		}
		updates["password_hash"] = hash
		updates["must_change_password"] = true
	}
	if len(updates) > 0 {
		return db.WithContext(ctx).Model(&models.User{}).Where("id = ?", user.ID).Updates(updates).Error
	}
	return nil
}

// LoginAdmin verifies a local administrator and records a security-relevant
// last-login timestamp before issuing the normal short-lived JWT pair.
func (s *AuthService) LoginAdmin(ctx context.Context, email, password string) (models.User, string, string, error) {
	user, err := s.users.ByEmail(ctx, normalizeEmail(email))
	if err != nil || user.Role != "admin" {
		return models.User{}, "", "", ErrInvalidCredentials
	}
	if err := VerifyPassword(user.PasswordHash, password); err != nil {
		return models.User{}, "", "", ErrInvalidCredentials
	}
	now := time.Now().UTC()
	user.LastLoginAt = &now
	if err := s.users.Update(ctx, &user); err != nil {
		return models.User{}, "", "", err
	}
	access, refresh, err := s.issueSession(ctx, user)
	return user, access, refresh, err
}

// RefreshAdmin rotates a refresh token only while its owner is still an admin.
func (s *AuthService) RefreshAdmin(ctx context.Context, refreshToken string) (models.User, string, string, error) {
	if s.tokens == nil {
		return models.User{}, "", "", ErrNoTokenStore
	}
	rec, err := s.tokens.ByHash(ctx, hashToken(refreshToken))
	if err != nil || rec.RevokedAt != nil || time.Now().After(rec.ExpiresAt) {
		return models.User{}, "", "", ErrTokenRevoked
	}
	user, err := s.userByID(ctx, rec.UserID)
	if err != nil || user.Role != "admin" {
		_ = s.tokens.Revoke(ctx, rec.ID)
		return models.User{}, "", "", ErrAdminForbidden
	}
	if err := s.tokens.Revoke(ctx, rec.ID); err != nil {
		return models.User{}, "", "", err
	}
	access, refresh, err := s.issueSession(ctx, user)
	return user, access, refresh, err
}

func (s *AuthService) AdminUser(ctx context.Context, userID uuid.UUID) (models.User, error) {
	user, err := s.userByID(ctx, userID)
	if err != nil {
		return models.User{}, err
	}
	if user.Role != "admin" {
		return models.User{}, ErrAdminForbidden
	}
	return user, nil
}

// ChangeAdminPassword atomically replaces the hash and revokes every existing
// refresh session. The handler issues one fresh session after this succeeds.
func (s *AuthService) ChangeAdminPassword(ctx context.Context, userID uuid.UUID, current, next string) (models.User, error) {
	if err := validateAdminPassword(next); err != nil {
		return models.User{}, err
	}
	user, err := s.AdminUser(ctx, userID)
	if err != nil {
		return models.User{}, err
	}
	if err := VerifyPassword(user.PasswordHash, current); err != nil {
		return models.User{}, ErrInvalidCredentials
	}
	if VerifyPassword(user.PasswordHash, next) == nil {
		return models.User{}, ErrPasswordUnchanged
	}
	hash, err := HashPassword(next)
	if err != nil {
		return models.User{}, err
	}
	now := time.Now().UTC()
	user.PasswordHash = hash
	user.MustChangePassword = false
	user.PasswordChangedAt = &now
	if err := s.users.Update(ctx, &user); err != nil {
		return models.User{}, err
	}
	if s.tokens != nil {
		if err := s.tokens.RevokeAllForUser(ctx, user.ID); err != nil {
			return models.User{}, err
		}
	}
	return user, nil
}

func validateAdminPassword(password string) error {
	if len([]rune(password)) < 12 || len(password) > 128 {
		return ErrWeakAdminPassword
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
	if !lower || !upper || !digit || !symbol {
		return ErrWeakAdminPassword
	}
	return nil
}
