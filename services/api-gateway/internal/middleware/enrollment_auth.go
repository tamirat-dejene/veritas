package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/tamirat-dejene/veritas/services/api-gateway/internal/domain"
	"go.uber.org/zap"
)

// enrollmentClaims mirrors the payload written by candidate-service's token service.
type enrollmentClaims struct {
	EnrollmentID string `json:"eid"`
	CandidateID  string `json:"cid"`
	ExamID       string `json:"xid"`
	EnterpriseID string `json:"ent"`
	Role         string `json:"role"`
	jwt.RegisteredClaims
}

// EnrollmentAuth validates an enrollment JWT (issued by candidate-service) and
// populates the Gin context with a synthesized *UserClaims so all downstream
// middleware (TenantResolver, InjectUserHeaders, RequireRole) work without change.
//
// It also sets the "subject_id" context key and "X-Subject-Id" request header
// with the candidate UUID, which candidate-service handlers read via getCandidateID.
func EnrollmentAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			zap.L().Warn("EnrollmentAuth: Authorization header required", zap.String("ip", c.ClientIP()))
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			zap.L().Warn("EnrollmentAuth: Invalid token format", zap.String("ip", c.ClientIP()))
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token format"})
			return
		}

		token, err := jwt.ParseWithClaims(tokenString, &enrollmentClaims{}, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			zap.L().Warn("EnrollmentAuth: Invalid enrollment token", zap.Error(err), zap.String("ip", c.ClientIP()))
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired enrollment token"})
			return
		}

		ec, ok := token.Claims.(*enrollmentClaims)
		if !ok || ec.CandidateID == "" || ec.EnterpriseID == "" || ec.EnrollmentID == "" || ec.ExamID == "" {
			zap.L().Warn("EnrollmentAuth: Incomplete enrollment token claims", zap.String("ip", c.ClientIP()))
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Incomplete enrollment token claims"})
			return
		}

		if ec.Role == "" {
			zap.L().Warn("EnrollmentAuth: Missing role in enrollment token", zap.String("ip", c.ClientIP()))
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Missing role in enrollment token"})
			return
		}

		// Synthesize a *UserClaims using the role that was embedded in the token at
		// enrollment time. TenantResolver and InjectUserHeaders read this exact struct.
		userClaims := &UserClaims{
			UserID:       ec.CandidateID,
			Role:         domain.Role(ec.Role),
			EnterpriseID: ec.EnterpriseID,
		}
		c.Set(string(UserContextKey), userClaims)

		// candidate-service handlers call getCandidateID which reads "subject_id"
		// from the Gin context or falls back to the X-Subject-Id request header.
		c.Set("subject_id", ec.CandidateID)
		c.Request.Header.Set("X-Subject-Id", sanitizeHeader(ec.CandidateID))

		c.Next()
	}
}
