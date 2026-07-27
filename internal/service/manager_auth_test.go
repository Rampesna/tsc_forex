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

func newManagerAuthDB(t *testing.T, name string) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.Manager{},
		&models.ManagerSession{},
		&models.NotificationCampaign{},
		&models.AdminAuditLog{},
	); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestManagerRealmIsSeparatedFromConsumerAccounts(t *testing.T) {
	t.Parallel()
	db := newManagerAuthDB(t, "manager-auth-isolation")
	ctx := context.Background()

	consumerHash, err := service.HashPassword("Consumer-Password-2026!")
	if err != nil {
		t.Fatal(err)
	}
	legacyUser := models.User{
		ID: uuid.New(), Email: "talha@talhacan.com", PasswordHash: consumerHash,
		Role: "admin", Provider: "local", DisplayName: "Consumer Talha",
	}
	if err := db.Create(&legacyUser).Error; err != nil {
		t.Fatal(err)
	}
	campaign := models.NotificationCampaign{
		ID: uuid.New(), ManagerID: legacyUser.ID, Name: "Legacy", Prompt: "legacy prompt",
		Status: "draft", TargetingPlan: models.JSONDocument(`{}`), Preview: models.JSONDocument(`{}`),
		Title: "Title", Body: "Body", ApprovalFingerprint: "fingerprint",
	}
	if err := db.Create(&campaign).Error; err != nil {
		t.Fatal(err)
	}
	audit := models.AdminAuditLog{
		ID: uuid.New(), ManagerID: &legacyUser.ID, Action: "admin.login",
		ResourceType: "session", Metadata: models.JSONDocument(`{}`),
	}
	if err := db.Create(&audit).Error; err != nil {
		t.Fatal(err)
	}

	managerRepo := repository.NewManagerRepository(db, nil)
	sessionRepo := repository.NewManagerSessionRepository(db, nil)
	cfg := config.AdminConfig{
		Enabled: true, BootstrapEmail: "Talha@TalhaCan.com",
		BootstrapPassword: "Initial-Manager-2026!",
	}
	if err := service.EnsureBootstrapManager(ctx, db, managerRepo, cfg); err != nil {
		t.Fatal(err)
	}

	manager, err := managerRepo.ByEmail(ctx, "talha@talhacan.com")
	if err != nil {
		t.Fatal(err)
	}
	if manager.ID == legacyUser.ID || !manager.MustChangePassword {
		t.Fatalf("manager was not isolated: %+v", manager)
	}
	if service.VerifyPassword(manager.PasswordHash, cfg.BootstrapPassword) != nil {
		t.Fatal("manager bootstrap password was not installed")
	}

	var consumer models.User
	if err := db.Take(&consumer, "id = ?", legacyUser.ID).Error; err != nil {
		t.Fatal(err)
	}
	if consumer.Role != "user" {
		t.Fatalf("consumer role=%q, want user", consumer.Role)
	}
	if service.VerifyPassword(consumer.PasswordHash, "Consumer-Password-2026!") != nil {
		t.Fatal("consumer password was modified by manager bootstrap")
	}
	if err := db.Take(&campaign, "id = ?", campaign.ID).Error; err != nil {
		t.Fatal(err)
	}
	if campaign.ManagerID != manager.ID {
		t.Fatalf("campaign manager=%s, want %s", campaign.ManagerID, manager.ID)
	}
	if err := db.Take(&audit, "id = ?", audit.ID).Error; err != nil {
		t.Fatal(err)
	}
	if audit.ManagerID == nil || *audit.ManagerID != manager.ID {
		t.Fatalf("audit manager=%v, want %s", audit.ManagerID, manager.ID)
	}

	auth := service.NewManagerAuthService(managerRepo, sessionRepo, testJWTCfg())
	loggedIn, access, refresh, err := auth.Login(ctx, manager.Email, cfg.BootstrapPassword)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := auth.Verify(access)
	if err != nil || claims.Role != "manager" || claims.Kind != "manager" || loggedIn.ID != manager.ID {
		t.Fatalf("manager session manager=%+v claims=%+v err=%v", loggedIn, claims, err)
	}

	rotated, err := auth.ChangePassword(ctx, manager.ID, cfg.BootstrapPassword, "Rotated-Manager-2026!")
	if err != nil || rotated.MustChangePassword {
		t.Fatalf("rotate manager=%+v err=%v", rotated, err)
	}
	if _, _, _, err := auth.Refresh(ctx, refresh); !errors.Is(err, service.ErrTokenRevoked) {
		t.Fatalf("old manager refresh error=%v, want revoked", err)
	}
	if _, _, _, err := auth.Login(ctx, manager.Email, "Rotated-Manager-2026!"); err != nil {
		t.Fatalf("login after rotation: %v", err)
	}

	// Deploys never reset an established manager password.
	cfg.BootstrapPassword = "Different-Bootstrap-2026!"
	if err := service.EnsureBootstrapManager(ctx, db, managerRepo, cfg); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := auth.Login(ctx, manager.Email, "Rotated-Manager-2026!"); err != nil {
		t.Fatalf("deployment reset manager password: %v", err)
	}
}

func TestManagerAndConsumerTokensAreMutuallyRejected(t *testing.T) {
	t.Parallel()
	db := newManagerAuthDB(t, "manager-auth-token-realms")
	ctx := context.Background()
	managerRepo := repository.NewManagerRepository(db, nil)
	sessionRepo := repository.NewManagerSessionRepository(db, nil)
	managerCfg := config.AdminConfig{
		Enabled: true, BootstrapEmail: "manager@example.com",
		BootstrapPassword: "Manager-Isolated-2026!",
	}
	if err := service.EnsureBootstrapManager(ctx, db, managerRepo, managerCfg); err != nil {
		t.Fatal(err)
	}
	managerAuth := service.NewManagerAuthService(managerRepo, sessionRepo, testJWTCfg())
	_, managerAccess, _, err := managerAuth.Login(ctx, managerCfg.BootstrapEmail, managerCfg.BootstrapPassword)
	if err != nil {
		t.Fatal(err)
	}

	userRepo := repository.NewUserRepository(db, nil)
	userAuth := service.NewAuthService(userRepo, testJWTCfg())
	if _, err := userAuth.Register(ctx, "consumer@example.com", "Consumer-Password-2026!", "admin"); err != nil {
		t.Fatal(err)
	}
	userAccess, _, err := userAuth.Login(ctx, "consumer@example.com", "Consumer-Password-2026!")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := userAuth.Verify(managerAccess); !errors.Is(err, service.ErrInvalidCredentials) {
		t.Fatalf("consumer verifier accepted manager token: %v", err)
	}
	if _, err := managerAuth.Verify(userAccess); !errors.Is(err, service.ErrInvalidCredentials) {
		t.Fatalf("manager verifier accepted consumer token: %v", err)
	}
}

func TestManagerBootstrapRequiresStrongSecret(t *testing.T) {
	t.Parallel()
	db := newManagerAuthDB(t, "manager-bootstrap-weak")
	err := service.EnsureBootstrapManager(
		context.Background(),
		db,
		repository.NewManagerRepository(db, nil),
		config.AdminConfig{
			Enabled: true, BootstrapEmail: "manager@example.com",
			BootstrapPassword: "weak",
		},
	)
	if !errors.Is(err, service.ErrManagerNotConfigured) {
		t.Fatalf("bootstrap error=%v, want ErrManagerNotConfigured", err)
	}
}
