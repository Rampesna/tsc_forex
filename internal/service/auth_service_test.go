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

type memRefreshRepo struct {
	byHash map[string]models.RefreshToken
	last   models.RefreshToken
}

func newMemRefreshRepo() *memRefreshRepo {
	return &memRefreshRepo{byHash: map[string]models.RefreshToken{}}
}

func (m *memRefreshRepo) Store(_ context.Context, token *models.RefreshToken) error {
	m.last = *token
	m.byHash[token.TokenHash] = *token
	return nil
}

func (m *memRefreshRepo) ByHash(_ context.Context, hash string) (models.RefreshToken, error) {
	token, ok := m.byHash[hash]
	if !ok {
		return models.RefreshToken{}, repository.ErrNotFound
	}
	return token, nil
}

func (m *memRefreshRepo) Revoke(_ context.Context, id uuid.UUID) error {
	now := time.Now().UTC()
	for hash, token := range m.byHash {
		if token.ID == id {
			token.RevokedAt = &now
			m.byHash[hash] = token
		}
	}
	return nil
}

func (m *memRefreshRepo) RevokeAllForUser(_ context.Context, userID uuid.UUID) error {
	now := time.Now().UTC()
	for hash, token := range m.byHash {
		if token.UserID == userID {
			token.RevokedAt = &now
			m.byHash[hash] = token
		}
	}
	return nil
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
func (m *memUserRepo) DeleteUser(_ context.Context, id uuid.UUID, _ ...string) error {
	for e, u := range m.byEmail {
		if u.ID == id {
			delete(m.byEmail, e)
		}
	}
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

func TestAuthSessionPersistsAndCarriesClientMetadata(t *testing.T) {
	t.Parallel()
	users := newMemUserRepo()
	tokens := newMemRefreshRepo()
	auth := service.NewAuthService(users, testJWTCfg()).WithRefreshStore(tokens)
	base := context.Background()
	if _, err := auth.Register(base, "android@example.com", "password123", "user"); err != nil {
		t.Fatal(err)
	}
	metadata := service.NormalizeClientMetadata("android", "tr-TR", "1.2.0", "Pixel 9")
	ctx := service.WithClientMetadata(base, metadata)
	_, refresh, err := auth.Login(ctx, "android@example.com", "password123")
	if err != nil {
		t.Fatal(err)
	}
	if tokens.last.Platform != "android" || tokens.last.Locale != "tr-TR" ||
		tokens.last.AppVersion != "1.2.0" || tokens.last.DeviceModel != "Pixel 9" {
		t.Fatalf("stored metadata=%+v", tokens.last)
	}

	// A refresh from an older client without headers keeps the provenance of
	// the session it rotates instead of regressing the user to "unknown".
	if _, _, err := auth.Refresh(base, refresh); err != nil {
		t.Fatal(err)
	}
	if tokens.last.Platform != "android" || tokens.last.AppVersion != "1.2.0" {
		t.Fatalf("rotated metadata=%+v", tokens.last)
	}
}
