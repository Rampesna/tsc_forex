package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/tsc-forex/tsc_forex/internal/config"
	"github.com/tsc-forex/tsc_forex/internal/metrics"
	"github.com/tsc-forex/tsc_forex/internal/middleware"
	"github.com/tsc-forex/tsc_forex/internal/models"
	"github.com/tsc-forex/tsc_forex/internal/repository"
	"github.com/tsc-forex/tsc_forex/internal/service"
)

type authMemRepo struct{ m map[string]models.User }

func (r *authMemRepo) Create(_ context.Context, u *models.User) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	r.m[u.Email] = *u
	return nil
}
func (r *authMemRepo) ByEmail(_ context.Context, e string) (models.User, error) {
	u, ok := r.m[e]
	if !ok {
		return models.User{}, repository.ErrUserNotFound
	}
	return u, nil
}
func (r *authMemRepo) ByID(_ context.Context, id uuid.UUID) (models.User, error) {
	for _, u := range r.m {
		if u.ID == id {
			return u, nil
		}
	}
	return models.User{}, repository.ErrUserNotFound
}
func (r *authMemRepo) Update(_ context.Context, u *models.User) error {
	r.m[u.Email] = *u
	return nil
}

func TestJWTAuthSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := config.JWTConfig{
		Secret: "abcdefghijklmnopqrstuvwxyz012345", Issuer: "t",
		AccessTokenTTL: time.Minute, RefreshTokenTTL: time.Hour,
	}
	auth := service.NewAuthService(&authMemRepo{m: map[string]models.User{}}, cfg)
	ctx := context.Background()
	if _, err := auth.Register(ctx, "a@b.com", "password123", "admin"); err != nil {
		t.Fatal(err)
	}
	access, _, err := auth.Login(ctx, "a@b.com", "password123")
	if err != nil {
		t.Fatal(err)
	}

	r := gin.New()
	r.Use(middleware.JWTAuth(auth))
	r.GET("/p", func(c *gin.Context) {
		claims, ok := c.Get(middleware.ClaimsKey)
		if !ok || claims == nil {
			t.Error("claims not set")
		}
		c.Status(http.StatusOK)
	})

	// Header token
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/p", nil)
	req.Header.Set("Authorization", "Bearer "+access)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("header token status = %d", w.Code)
	}

	// Query token (WebSocket handshake style)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/p?token="+access, nil))
	if w.Code != http.StatusOK {
		t.Fatalf("query token status = %d", w.Code)
	}
}

func TestMetricsMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	m := metrics.New("test")
	r := gin.New()
	r.Use(middleware.Metrics(m))
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}

	// nil metrics is a no-op passthrough.
	r2 := gin.New()
	r2.Use(middleware.Metrics(nil))
	r2.GET("/y", func(c *gin.Context) { c.Status(http.StatusOK) })
	w = httptest.NewRecorder()
	r2.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/y", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("nil metrics status = %d", w.Code)
	}
}

func TestNoCacheHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.NoCache())
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("missing no-store")
	}
}
