package middleware

import (
	"github.com/gin-gonic/gin"
)

const TenantIDKey = "tenantID"

// TenantResolver middleware extracts the enterpriseId from the JWT claims
// and injects it into the request context.
// This relies on JWTAuth middleware being run beforehand.
func TenantResolver() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try to get claims from context (populated by JWTAuth)
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

		if claims.EnterpriseID != "" {
			c.Set(TenantIDKey, claims.EnterpriseID)
		}

		c.Next()
	}
}
