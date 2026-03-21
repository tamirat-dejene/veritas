package middleware

import (
	"net/http"
	"strings"

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

		c.Request.Header.Set("X-User-ID", sanitizeHeader(claims.UserID))
		c.Request.Header.Set("X-User-Role", sanitizeHeader(string(claims.Role)))
		if claims.EnterpriseID != "" {
			c.Request.Header.Set("X-Enterprise-ID", sanitizeHeader(claims.EnterpriseID))
		}

		// Strip the Authorization header so it's not forwarded downstream.
		// Downstream services should only trust X-User-* headers, never raw JWTs.
		c.Request.Header.Del("Authorization")

		c.Next()
	}
}

// sanitizeHeader removes CR and LF characters from a string to prevent
// HTTP header injection attacks.
func sanitizeHeader(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '\r' || r == '\n' {
			return -1 // drop the character
		}
		return r
	}, s)
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
