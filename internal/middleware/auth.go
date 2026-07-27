package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/tsc-forex/tsc_forex/internal/service"
)

// ClaimsKey is the gin context key for verified JWT claims.
const ClaimsKey = "auth_claims"

// JWTAuth verifies the bearer token and stores claims in the context.
func JWTAuth(auth *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		token := ""
		if strings.HasPrefix(header, "Bearer ") {
			token = strings.TrimPrefix(header, "Bearer ")
		} else if q := c.Query("token"); q != "" {
			// Allow token via query for WebSocket handshakes.
			token = q
		}
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			return
		}
		claims, err := auth.Verify(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		c.Set(ClaimsKey, claims)
		c.Next()
	}
}

// ManagerJWTAuth verifies only operations-console JWTs. It intentionally does
// not accept query-string credentials.
func ManagerJWTAuth(auth *service.ManagerAuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			return
		}
		claims, err := auth.Verify(strings.TrimPrefix(header, "Bearer "))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		c.Set(ClaimsKey, claims)
		c.Next()
	}
}

// ManagerOnly adds a second fail-closed check after ManagerJWTAuth.
func ManagerOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		raw, ok := c.Get(ClaimsKey)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing claims"})
			return
		}
		claims, ok := raw.(*service.Claims)
		if !ok || claims.Role != "manager" || claims.Kind != "manager" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "manager access required"})
			return
		}
		c.Next()
	}
}
