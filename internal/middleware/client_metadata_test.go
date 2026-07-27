package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/tsc-forex/tsc_forex/internal/service"
)

func TestClientMetadataMiddleware(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(ClientMetadata())
	r.GET("/", func(c *gin.Context) {
		metadata, ok := service.ClientMetadataFromContext(c.Request.Context())
		if !ok {
			c.Status(http.StatusBadRequest)
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"platform": metadata.Platform,
			"locale":   metadata.Locale,
			"version":  metadata.AppVersion,
			"model":    metadata.DeviceModel,
		})
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(clientPlatformHeader, "ANDROID")
	req.Header.Set(clientLocaleHeader, "tr-TR")
	req.Header.Set(clientVersionHeader, "1.2.0")
	req.Header.Set(clientModelHeader, "Pixel 9")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestClientMetadataMiddlewareInfersPublishedAndroidClient(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(ClientMetadata())
	r.GET("/", func(c *gin.Context) {
		metadata, ok := service.ClientMetadataFromContext(c.Request.Context())
		if !ok {
			c.Status(http.StatusBadRequest)
			return
		}
		c.String(http.StatusOK, metadata.Platform)
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("User-Agent", "okhttp/5.1.0")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || rec.Body.String() != "android" {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
}
