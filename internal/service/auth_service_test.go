package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/tsc-forex/tsc_forex/internal/config"
	"github.com/tsc-forex/tsc_forex/internal/models"
	"github.com/tsc-forex/tsc_forex/internal/repository"
	"github.com/tsc-forex/tsc_forex/internal/service"
)

// memUserRepo is an in-memory UserRepository for auth tests.
type memUserRepo struct {
	byEmail map[string]models.User
}

func newMemUserRepo() *memUserRepo { return &memUserRepo{byEmail: map[string]models.User{}} }

func (m *memUserRepo) Create(_ context.Context, u *models.User) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	m.byEmail[u.Email] = *u
	return nil
}

func (m *memUserRepo) ByEmail(_ context.Context, email string) (models.User, error) {
	u, ok := m.byEmail[email]
	if !ok {
		return models.User{}, repository.ErrUserNotFound
	}
	return u, nil
}

func (m *memUserRepo) ByID(_ context.Context, id uuid.UUID) (models.User, error) {
	for _, u := range m.byEmail {
		if u.ID == id {
			return u, nil
		}
	}
	return models.User{}, repository.ErrUserNotFound
}

func (m *memUserRepo) Update(_ context.Context, u *models.User) error {
	m.byEmail[u.Email] = *u
	return nil
}

func testJWTCfg() config.JWTConfig {
	return config.JWTConfig{
		Secret:          "abcdefghijklmnopqrstuvwxyz012345",
		Issuer:          "tsc_forex_test",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: time.Hour,
	}
}

func TestAuthRegisterLoginVerify(t *testing.T) {
	t.Parallel()
	repo := newMemUserRepo()
	auth := service.NewAuthService(repo, testJWTCfg())
	ctx := context.Background()

	if _, err := auth.Register(ctx, "a@b.com", "password123", "user"); err != nil {
		t.Fatal(err)
	}
	if _, err := auth.Register(ctx, "a@b.com", "password123", "user"); err != service.ErrEmailTaken {
		t.Fatalf("expected ErrEmailTaken, got %v", err)
	}

	access, refresh, err := auth.Login(ctx, "a@b.com", "password123")
	if err != nil {
		t.Fatal(err)
	}
	if access == "" || refresh == "" {
		t.Fatal("empty tokens")
	}

	claims, err := auth.Verify(access)
	if err != nil {
		t.Fatal(err)
	}
	if claims.Email != "a@b.com" || claims.Role != "user" {
		t.Fatalf("claims = %+v", claims)
	}

	if _, _, err := auth.Login(ctx, "a@b.com", "wrongpass"); err != service.ErrInvalidCredentials {
		t.Fatalf("expected invalid credentials, got %v", err)
	}
	if _, err := auth.Verify("garbage.token.value"); err == nil {
		t.Fatal("expected verify error")
	}
}
