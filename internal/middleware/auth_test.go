package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/tsc-forex/tsc_forex/internal/config"
	"github.com/tsc-forex/tsc_forex/internal/middleware"
	"github.com/tsc-forex/tsc_forex/internal/service"
)

type memUsers struct{}

func TestJWTAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Build an auth service with a throwaway user via in-memory repo is overkill;
	// we only need Verify, so issue a token through Login is not possible without repo.
	// Instead test the reject path (missing/invalid token) which needs no repo.
	auth := service.NewAuthService(nil, config.JWTConfig{
		Secret: "abcdefghijklmnopqrstuvwxyz012345",
		Issuer: "tsc_forex_test",
	})

	r := gin.New()
	r.Use(middleware.JWTAuth(auth))
	r.GET("/p", func(c *gin.Context) { c.Status(http.StatusOK) })

	// No token → 401.
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/p", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("no token status = %d", w.Code)
	}

	// Invalid token → 401.
	w = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/p", nil)
	req.Header.Set("Authorization", "Bearer not.a.jwt")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("bad token status = %d", w.Code)
	}
}

var _ = memUsers{}
