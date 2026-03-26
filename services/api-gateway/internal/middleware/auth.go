package middleware

import (
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/tamirat-dejene/veritas/services/api-gateway/internal/domain"
	"go.uber.org/zap"
)

type userContextKey string

const UserContextKey userContextKey = "user"

type UserClaims struct {
	EnterpriseID string      `json:"enterpriseId"`
	UserID       string      `json:"sub"`
	Role         domain.Role `json:"role"`
	Tier         string      `json:"tier"`
	jwt.RegisteredClaims
}

func JWTAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			zap.L().Warn("Auth: Authorization header required", zap.String("ip", c.ClientIP()))
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			zap.L().Warn("Auth: Invalid token format", zap.String("ip", c.ClientIP()))
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
			zap.L().Warn("Auth: Invalid token", zap.Error(err), zap.String("ip", c.ClientIP()))
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}

		if claims, ok := token.Claims.(*UserClaims); ok {
			c.Set(string(UserContextKey), claims)
			c.Next()
		} else {
			zap.L().Warn("Auth: Invalid token claims", zap.String("ip", c.ClientIP()))
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
		}
	}
}

func RequireRole(allowedRoles ...domain.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		claimsVal, exists := c.Get(string(UserContextKey))
		if !exists {
			zap.L().Warn("Auth: Unauthorized (missing claims)", zap.String("ip", c.ClientIP()))
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		claims, ok := claimsVal.(*UserClaims)
		if !ok {
			zap.L().Warn("Auth: Unauthorized (invalid claims type)", zap.String("ip", c.ClientIP()))
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		if slices.Contains(allowedRoles, domain.RoleAll) {
			c.Next()
			return
		}

		if slices.Contains(allowedRoles, claims.Role) {
			c.Next()
			return
		}

		zap.L().Warn("Auth: Forbidden",
			zap.String("enterpriseID", claims.EnterpriseID),
			zap.String("userID", claims.UserID),
			zap.String("userRole", string(claims.Role)),
			zap.Any("allowedRoles", allowedRoles),
			zap.String("ip", c.ClientIP()),
		)
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
	}
}

func RequireTier(tier domain.Tier) gin.HandlerFunc {
	return func(c *gin.Context) {
		claimsVal, exists := c.Get(string(UserContextKey))
		if !exists {
			zap.L().Warn("RequireTier: missing claims", zap.String("required_tier", string(tier)), zap.String("ip", c.ClientIP()))
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": tier + " subscription required"})
			return
		}
		claims, ok := claimsVal.(*UserClaims)
		if !ok || claims.Tier != string(tier) {
			zap.L().Warn("RequireTier: insufficient tier",
				zap.String("required_tier", string(tier)),
				zap.String("user_tier", func() string {
					if ok {
						return claims.Tier
					}
					return "unknown"
				}()),
				zap.String("ip", c.ClientIP()),
			)
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": tier + " subscription required"})
			return
		}
		c.Next()
	}
}
