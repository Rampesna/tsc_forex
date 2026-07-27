package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/tsc-forex/tsc_forex/internal/config"
	"github.com/tsc-forex/tsc_forex/internal/middleware"
)

func TestRateLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.RateLimit(3, 60))
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	var lastStatus int
	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		req.RemoteAddr = "1.2.3.4:1111"
		r.ServeHTTP(w, req)
		lastStatus = w.Code
	}
	if lastStatus != http.StatusTooManyRequests {
		t.Fatalf("expected 429 after limit, got %d", lastStatus)
	}
}

func TestSecurityHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.SecurityHeaders())
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	r.ServeHTTP(w, req)
	if w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("missing nosniff header")
	}
	if w.Header().Get("X-Frame-Options") != "DENY" {
		t.Fatal("missing frame options")
	}
}

func TestCORSPreflight(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.CORS([]string{"*"}))
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/x", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("preflight status = %d", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatal("missing CORS origin")
	}
}

func TestCORSAllowList(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.CORS([]string{"https://app.example.com"}))
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	// Allowed origin echoes back.
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Origin", "https://app.example.com")
	r.ServeHTTP(w, req)
	if w.Header().Get("Access-Control-Allow-Origin") != "https://app.example.com" {
		t.Fatalf("origin = %q", w.Header().Get("Access-Control-Allow-Origin"))
	}

	// Disallowed origin is not echoed.
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Origin", "https://evil.com")
	r.ServeHTTP(w, req)
	if w.Header().Get("Access-Control-Allow-Origin") == "https://evil.com" {
		t.Fatal("evil origin allowed")
	}
}

func TestCredentialedCORSRequiresExactOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.CORSWithConfig(config.CORSConfig{
		AllowedOrigins:   []string{"https://tscfm.talhacan.com"},
		AllowCredentials: true,
	}))
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	allowed := httptest.NewRecorder()
	allowedRequest := httptest.NewRequest(http.MethodGet, "/x", nil)
	allowedRequest.Header.Set("Origin", "https://tscfm.talhacan.com")
	r.ServeHTTP(allowed, allowedRequest)
	if got := allowed.Header().Get("Access-Control-Allow-Origin"); got != "https://tscfm.talhacan.com" {
		t.Fatalf("allowed origin = %q", got)
	}
	if got := allowed.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("allow credentials = %q", got)
	}

	blocked := httptest.NewRecorder()
	blockedRequest := httptest.NewRequest(http.MethodGet, "/x", nil)
	blockedRequest.Header.Set("Origin", "https://tscfm.talhacan.com.evil.example")
	r.ServeHTTP(blocked, blockedRequest)
	if got := blocked.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("unexpected reflected origin = %q", got)
	}
	if got := blocked.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Fatalf("unexpected credential permission = %q", got)
	}
}

func TestRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.RequestID())
	r.GET("/x", func(c *gin.Context) {
		if c.GetString(middleware.RequestIDKey) == "" {
			t.Error("request id not set")
		}
		c.Status(http.StatusOK)
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	r.ServeHTTP(w, req)
	if w.Header().Get(middleware.RequestIDHeader) == "" {
		t.Fatal("missing request id header")
	}
}
