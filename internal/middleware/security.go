package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/tsc-forex/tsc_forex/internal/config"
)

// SecurityHeaders adds baseline hardening headers for API and upgrade responses.
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("X-XSS-Protection", "0")
		c.Next()
	}
}

// CORS applies a simple allow-list. Full JWT/rate-limit land in later steps.
func CORS(origins []string) gin.HandlerFunc {
	return CORSWithConfig(config.CORSConfig{AllowedOrigins: origins})
}

// CORSWithConfig applies an exact origin allow-list and supports credentialed
// admin refresh cookies only when the response echoes an approved origin.
func CORSWithConfig(cfg config.CORSConfig) gin.HandlerFunc {
	origins := cfg.AllowedOrigins
	allowAll := len(origins) == 0 || (len(origins) == 1 && origins[0] == "*")
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if allowAll {
			// The CORS specification forbids wildcard origins with credentials.
			if cfg.AllowCredentials && origin != "" {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Vary", "Origin")
			} else {
				c.Header("Access-Control-Allow-Origin", "*")
			}
		} else if origin != "" {
			for _, o := range origins {
				if o == origin {
					c.Header("Access-Control-Allow-Origin", origin)
					c.Header("Vary", "Origin")
					break
				}
			}
		}
		if cfg.AllowCredentials && c.Writer.Header().Get("Access-Control-Allow-Origin") != "" {
			c.Header("Access-Control-Allow-Credentials", "true")
		}
		methods := "GET,POST,PATCH,PUT,DELETE,OPTIONS"
		if len(cfg.AllowedMethods) > 0 {
			methods = joinHeaderValues(cfg.AllowedMethods)
		}
		headers := "Authorization,Content-Type,X-Request-ID,X-Idempotency-Key"
		if len(cfg.AllowedHeaders) > 0 {
			headers = joinHeaderValues(cfg.AllowedHeaders)
		}
		maxAge := cfg.MaxAge
		if maxAge <= 0 {
			maxAge = 86400
		}
		c.Header("Access-Control-Allow-Methods", methods)
		c.Header("Access-Control-Allow-Headers", headers)
		c.Header("Access-Control-Max-Age", strconv.Itoa(maxAge))
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func joinHeaderValues(values []string) string {
	out := ""
	for _, value := range values {
		if value == "" {
			continue
		}
		if out != "" {
			out += ","
		}
		out += value
	}
	return out
}

// NoCache discourages intermediary caching of live market responses.
func NoCache() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		c.Header("Pragma", "no-cache")
		c.Header("Expires", time.Unix(0, 0).UTC().Format(http.TimeFormat))
		c.Next()
	}
}
