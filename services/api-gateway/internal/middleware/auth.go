package middleware

import (
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/tamirat-dejene/veritas/services/api-gateway/internal/domain"
)

type userContextKey string

const UserContextKey userContextKey = "user"

type UserClaims struct {
	UserID       string      `json:"sub"`
	Role         domain.Role `json:"role"`
	EnterpriseID string      `json:"enterpriseId"`
	Tier         string      `json:"tier"`
	jwt.RegisteredClaims
}

func JWTAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token format"})
			return
		}

		token, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token: " + err.Error()})
			return
		}

		if claims, ok := token.Claims.(*UserClaims); ok {
			c.Set(string(UserContextKey), claims)
			c.Next()
		} else {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
		}
	}
}

func RequireRole(allowedRoles ...domain.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		claimsVal, exists := c.Get(string(UserContextKey))
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		claims, ok := claimsVal.(*UserClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		// Check if "All" is in allowedRoles, effectively disabling the check for authenticated users
		if slices.Contains(allowedRoles, domain.RoleAll) {
			c.Next()
			return
		}

		if slices.Contains(allowedRoles, claims.Role) {
			c.Next()
			return
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
	}
}
