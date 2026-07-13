package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/tsc-forex/tsc_forex/internal/config"
	"github.com/tsc-forex/tsc_forex/internal/models"
	"github.com/tsc-forex/tsc_forex/internal/repository"
)

// tinyUserRepo is a minimal in-memory UserRepository for internal oauth tests.
type tinyUserRepo struct{ m map[string]models.User }

func (r *tinyUserRepo) Create(_ context.Context, u *models.User) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	r.m[u.Email] = *u
	return nil
}
func (r *tinyUserRepo) ByEmail(_ context.Context, e string) (models.User, error) {
	u, ok := r.m[e]
	if !ok {
		return models.User{}, repository.ErrUserNotFound
	}
	return u, nil
}
func (r *tinyUserRepo) ByID(_ context.Context, id uuid.UUID) (models.User, error) {
	for _, u := range r.m {
		if u.ID == id {
			return u, nil
		}
	}
	return models.User{}, repository.ErrUserNotFound
}
func (r *tinyUserRepo) Update(_ context.Context, u *models.User) error {
	r.m[u.Email] = *u
	return nil
}

func oauthTestCfg() config.JWTConfig {
	return config.JWTConfig{
		Secret:          "abcdefghijklmnopqrstuvwxyz012345",
		Issuer:          "tsc_forex_test",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: time.Hour,
	}
}

func TestLoginWithGoogle_NotConfigured(t *testing.T) {
	repo := &tinyUserRepo{m: map[string]models.User{}}
	svc := NewAuthService(repo, oauthTestCfg())
	if _, _, err := svc.LoginWithGoogle(context.Background(), "tok"); err != ErrOAuthNotConfigured {
		t.Fatalf("want ErrOAuthNotConfigured, got %v", err)
	}
}

func TestLoginWithApple_NotConfigured(t *testing.T) {
	repo := &tinyUserRepo{m: map[string]models.User{}}
	svc := NewAuthService(repo, oauthTestCfg())
	if _, _, err := svc.LoginWithApple(context.Background(), "tok", ""); err != ErrOAuthNotConfigured {
		t.Fatalf("want ErrOAuthNotConfigured, got %v", err)
	}
}

func TestLoginWithApple_CreatesAndReuses(t *testing.T) {
	repo := &tinyUserRepo{m: map[string]models.User{}}
	svc := NewAuthService(repo, oauthTestCfg()).
		WithOAuth(config.OAuthConfig{AppleBundleIDs: []string{"talhacan.tscforex"}})
	svc.appleVerify = func(_ context.Context, _ string, auds []string) (*AppleIdentity, error) {
		if !containsFold(auds, "talhacan.tscforex") {
			t.Fatalf("allowed auds not propagated: %v", auds)
		}
		return &AppleIdentity{Sub: "a1", Email: "Apple@Example.com", Aud: "talhacan.tscforex"}, nil
	}

	access, refresh, err := svc.LoginWithApple(context.Background(), "tok", "Ada Lovelace")
	if err != nil {
		t.Fatalf("apple login: %v", err)
	}
	if access == "" || refresh == "" {
		t.Fatal("expected tokens")
	}
	u, err := repo.ByEmail(context.Background(), "apple@example.com")
	if err != nil {
		t.Fatalf("user not created: %v", err)
	}
	if u.Provider != "apple" || u.DisplayName != "Ada Lovelace" {
		t.Fatalf("profile not populated: %+v", u)
	}
	if _, _, err := svc.LoginWithApple(context.Background(), "tok2", ""); err != nil {
		t.Fatalf("second apple login: %v", err)
	}
	if len(repo.m) != 1 {
		t.Fatalf("expected 1 user, got %d", len(repo.m))
	}
}

func TestLoginWithApple_SyntheticEmailWhenHidden(t *testing.T) {
	repo := &tinyUserRepo{m: map[string]models.User{}}
	svc := NewAuthService(repo, oauthTestCfg()).
		WithOAuth(config.OAuthConfig{AppleBundleIDs: []string{"talhacan.tscforex"}})
	svc.appleVerify = func(_ context.Context, _ string, _ []string) (*AppleIdentity, error) {
		return &AppleIdentity{Sub: "sub-xyz", Email: "", Aud: "talhacan.tscforex"}, nil
	}
	if _, _, err := svc.LoginWithApple(context.Background(), "tok", ""); err != nil {
		t.Fatalf("apple login: %v", err)
	}
	if _, err := repo.ByEmail(context.Background(), "apple_sub-xyz@users.tscforex"); err != nil {
		t.Fatalf("synthetic email account not created: %v", err)
	}
}

func TestLoginWithGoogle_CreatesAndReuses(t *testing.T) {
	repo := &tinyUserRepo{m: map[string]models.User{}}
	svc := NewAuthService(repo, oauthTestCfg()).
		WithOAuth(config.OAuthConfig{GoogleClientIDs: []string{"aud-ios"}})
	// Inject a fake verifier so no network / real token is needed.
	svc.googleVerify = func(_ context.Context, _ string, auds []string) (*GoogleIdentity, error) {
		if !containsFold(auds, "aud-ios") {
			t.Fatalf("allowed auds not propagated: %v", auds)
		}
		return &GoogleIdentity{Sub: "g1", Email: "New@Example.com", Name: "Neo", Picture: "http://x/y.png", Aud: "aud-ios"}, nil
	}

	access, refresh, err := svc.LoginWithGoogle(context.Background(), "tok")
	if err != nil {
		t.Fatalf("google login: %v", err)
	}
	if access == "" || refresh == "" {
		t.Fatal("expected tokens")
	}
	// Email is normalized to lowercase and the account is created as google.
	u, err := repo.ByEmail(context.Background(), "new@example.com")
	if err != nil {
		t.Fatalf("user not created: %v", err)
	}
	if u.Provider != "google" || u.DisplayName != "Neo" || u.AvatarURL == "" {
		t.Fatalf("profile not populated: %+v", u)
	}

	// Second login must reuse the same user (no duplicate, valid claims).
	if _, _, err := svc.LoginWithGoogle(context.Background(), "tok2"); err != nil {
		t.Fatalf("second google login: %v", err)
	}
	if len(repo.m) != 1 {
		t.Fatalf("expected 1 user, got %d", len(repo.m))
	}
}
