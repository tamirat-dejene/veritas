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

// TryEnrollmentOrUserAuth is used on routes that are accessible to BOTH
// ExamCandidates (enrollment JWT) and other authenticated roles (auth-service JWT).
//
// It attempts enrollment token parsing first. If that fails it falls back to
// the standard user JWT. If both fail it aborts with 401. Whichever path
// succeeds populates UserContextKey with a *UserClaims, so TenantResolver,
// InjectUserHeaders, and RequireRole all work identically afterwards.
func TryEnrollmentOrUserAuth(enrollmentSecret, jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			zap.L().Warn("TryEnrollmentOrUserAuth: Authorization header required", zap.String("ip", c.ClientIP()))
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			zap.L().Warn("TryEnrollmentOrUserAuth: Invalid token format", zap.String("ip", c.ClientIP()))
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token format"})
			return
		}

		// --- Attempt 1: enrollment token ---
		if tryParseEnrollmentToken(c, tokenString, enrollmentSecret) {
			c.Next()
			return
		}

		// --- Attempt 2: standard user JWT ---
		if tryParseUserToken(c, tokenString, jwtSecret) {
			c.Next()
			return
		}

		zap.L().Warn("TryEnrollmentOrUserAuth: Both token types failed", zap.String("ip", c.ClientIP()))
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
	}
}

// tryParseEnrollmentToken returns true and populates the context if the token
// is a valid enrollment JWT; returns false without aborting otherwise.
func tryParseEnrollmentToken(c *gin.Context, tokenString, secret string) bool {
	token, err := jwt.ParseWithClaims(tokenString, &enrollmentClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return false
	}
	ec, ok := token.Claims.(*enrollmentClaims)
	if !ok || ec.CandidateID == "" || ec.EnterpriseID == "" || ec.Role == "" {
		return false
	}

	c.Set(string(UserContextKey), &UserClaims{
		UserID:       ec.CandidateID,
		Role:         domain.Role(ec.Role),
		EnterpriseID: ec.EnterpriseID,
	})
	c.Set("subject_id", ec.CandidateID)
	c.Set("enrollment_id", ec.EnrollmentID)
	c.Set("exam_id", ec.ExamID)
	c.Request.Header.Set("X-Subject-Id", sanitizeHeader(ec.CandidateID))
	c.Request.Header.Set("X-Enrollment-Id", sanitizeHeader(ec.EnrollmentID))
	c.Request.Header.Set("X-Exam-Id", sanitizeHeader(ec.ExamID))
	return true
}

// tryParseUserToken returns true and populates the context if the token
// is a valid auth-service JWT; returns false without aborting otherwise.
func tryParseUserToken(c *gin.Context, tokenString, secret string) bool {
	token, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return false
	}
	claims, ok := token.Claims.(*UserClaims)
	if !ok {
		return false
	}
	c.Set(string(UserContextKey), claims)
	return true
}
