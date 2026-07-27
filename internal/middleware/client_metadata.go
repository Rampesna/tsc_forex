package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/tsc-forex/tsc_forex/internal/service"
)

const (
	clientPlatformHeader = "X-TSC-Client-Platform"
	clientLocaleHeader   = "X-TSC-Client-Locale"
	clientVersionHeader  = "X-TSC-App-Version"
	clientModelHeader    = "X-TSC-Device-Model"
)

// ClientMetadata captures native-client provenance before public
// authentication handlers run. Invalid or browser-supplied values are ignored;
// they never block a request and never become an arbitrary database value.
func ClientMetadata() gin.HandlerFunc {
	return func(c *gin.Context) {
		platform := c.GetHeader(clientPlatformHeader)
		if platform == "" {
			platform = inferNativePlatform(c.GetHeader("User-Agent"))
		}
		locale := c.GetHeader(clientLocaleHeader)
		if locale == "" {
			locale = strings.SplitN(c.GetHeader("Accept-Language"), ",", 2)[0]
		}
		metadata := service.NormalizeClientMetadata(
			platform,
			locale,
			c.GetHeader(clientVersionHeader),
			c.GetHeader(clientModelHeader),
		)
		if metadata.Platform != "" {
			c.Request = c.Request.WithContext(service.WithClientMetadata(c.Request.Context(), metadata))
		}
		c.Next()
	}
}

// inferNativePlatform keeps currently published clients compatible while they
// roll onto explicit X-TSC headers. OkHttp is the Android application's
// transport; URLSession advertises CFNetwork/Darwin on Apple platforms.
func inferNativePlatform(userAgent string) string {
	userAgent = strings.ToLower(userAgent)
	switch {
	case strings.Contains(userAgent, "okhttp"):
		return service.ClientPlatformAndroid
	case strings.Contains(userAgent, "cfnetwork"),
		strings.Contains(userAgent, "iphone"),
		strings.Contains(userAgent, "ipad"):
		return service.ClientPlatformIOS
	default:
		return ""
	}
}
