package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// InjectUserHeaders reads the validated JWT claims from the Gin context
// (set by JWTAuth) and writes them as request headers so that downstream
// services (enterprise-service, exam-service, etc.) can identify the caller
// without re-validating the token.
//
// Headers injected:
//
//	X-User-ID        – JWT subject (user UUID)
//	X-User-Role      – user role string
//	X-Enterprise-ID  – enterprise UUID (only if present in claims)
func InjectUserHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		claimsVal, exists := c.Get(string(UserContextKey))
		if !exists {
			c.Next()
			return
		}
		claims, ok := claimsVal.(*UserClaims)
		if !ok {
			c.Next()
			return
		}

		c.Request.Header.Set("X-User-ID", claims.UserID)
		c.Request.Header.Set("X-User-Role", string(claims.Role))
		if claims.EnterpriseID != "" {
			c.Request.Header.Set("X-Enterprise-ID", claims.EnterpriseID)
		}

		// Strip the Authorization header so it's not forwarded downstream.
		// Downstream services should only trust X-User-* headers, never raw JWTs.
		c.Request.Header.Del("Authorization")

		c.Next()
	}
}

// RequireHTTPS is a lightweight middleware that rejects non-TLS requests.
// Useful for protecting sensitive routes in production.
func RequireHTTPS() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.TLS == nil && c.GetHeader("X-Forwarded-Proto") != "https" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "HTTPS required"})
			return
		}
		c.Next()
	}
}
