package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/tsc-forex/tsc_forex/internal/config"
	"github.com/tsc-forex/tsc_forex/internal/models"
	"github.com/tsc-forex/tsc_forex/internal/repository"
	"github.com/tsc-forex/tsc_forex/internal/service"
)

func TestAdminBootstrapLoginAndPasswordRotation(t *testing.T) {
	t.Parallel()
	db, err := gorm.Open(sqlite.Open("file:admin-auth?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.RefreshToken{}); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	adminCfg := config.AdminConfig{
		Enabled: true, BootstrapEmail: "Talha@TalhaCan.com",
		BootstrapPassword: "Initial-Admin-2026!",
	}
	if err := service.EnsureBootstrapAdmin(ctx, db, adminCfg); err != nil {
		t.Fatal(err)
	}

	users := repository.NewUserRepository(db, nil)
	tokens := repository.NewRefreshTokenRepository(db, nil)
	auth := service.NewAuthService(users, testJWTCfg()).WithRefreshStore(tokens)
	user, access, refresh, err := auth.LoginAdmin(ctx, "talha@talhacan.com", adminCfg.BootstrapPassword)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := auth.Verify(access)
	if err != nil || claims.Role != "admin" || user.Role != "admin" || !user.MustChangePassword {
		t.Fatalf("admin session user=%+v claims=%+v err=%v", user, claims, err)
	}

	rotated, err := auth.ChangeAdminPassword(ctx, user.ID, adminCfg.BootstrapPassword, "Rotated-Admin-2026!")
	if err != nil || rotated.MustChangePassword {
		t.Fatalf("rotate user=%+v err=%v", rotated, err)
	}
	if _, _, _, err := auth.RefreshAdmin(ctx, refresh); !errors.Is(err, service.ErrTokenRevoked) {
		t.Fatalf("old refresh token error=%v, want revoked", err)
	}
	if _, _, _, err := auth.LoginAdmin(ctx, user.Email, "Rotated-Admin-2026!"); err != nil {
		t.Fatalf("login after rotation: %v", err)
	}

	// A later deployment must never reset an administrator's chosen password.
	adminCfg.BootstrapPassword = "Different-Bootstrap-2026!"
	if err := service.EnsureBootstrapAdmin(ctx, db, adminCfg); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := auth.LoginAdmin(ctx, user.Email, "Rotated-Admin-2026!"); err != nil {
		t.Fatalf("deployment reset chosen password: %v", err)
	}
}

func TestAdminBootstrapDoesNotPromoteAccountWithDifferentPassword(t *testing.T) {
	t.Parallel()
	db, err := gorm.Open(sqlite.Open("file:admin-bootstrap-mismatch?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.User{}); err != nil {
		t.Fatal(err)
	}
	hash, err := service.HashPassword("Consumer-Password-2026!")
	if err != nil {
		t.Fatal(err)
	}
	user := models.User{
		ID: uuid.New(), Email: "talha@talhacan.com", PasswordHash: hash,
		Role: "user", Provider: "local",
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	err = service.EnsureBootstrapAdmin(context.Background(), db, config.AdminConfig{
		Enabled: true, BootstrapEmail: user.Email, BootstrapPassword: "Different-Admin-2026!",
	})
	if !errors.Is(err, service.ErrAdminNotConfigured) {
		t.Fatalf("bootstrap error=%v, want ErrAdminNotConfigured", err)
	}
	var persisted models.User
	if err := db.Take(&persisted, "id = ?", user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if persisted.Role != "user" {
		t.Fatalf("unexpected role promotion: %s", persisted.Role)
	}
}
