package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/tamirat-dejene/veritas/services/api-gateway/internal/domain"
)

func TestEnrollmentAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := "test-secret"
	
	t.Run("Valid Token", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, r := gin.CreateTestContext(w)
		
		eid := uuid.New().String()
		cid := uuid.New().String()
		xid := uuid.New().String()
		ent := uuid.New().String()
		role := string(domain.RoleExamCandidate)
		
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"eid": eid,
			"cid": cid,
			"xid": xid,
			"ent": ent,
			"role": role,
			"iat": time.Now().Unix(),
		})
		tokenString, _ := token.SignedString([]byte(secret))
		
		r.GET("/test", EnrollmentAuth(secret), func(c *gin.Context) {
			claims, exists := c.Get(string(UserContextKey))
			assert.True(t, exists)
			userClaims := claims.(*UserClaims)
			assert.Equal(t, cid, userClaims.UserID)
			assert.Equal(t, domain.RoleExamCandidate, userClaims.Role)
			assert.Equal(t, ent, userClaims.EnterpriseID)
			
			subID, exists := c.Get("subject_id")
			assert.True(t, exists)
			assert.Equal(t, cid, subID)
			
			assert.Equal(t, cid, c.GetHeader("X-Subject-Id"))
			c.Status(http.StatusOK)
		})
		
		c.Request, _ = http.NewRequest(http.MethodGet, "/test", nil)
		c.Request.Header.Set("Authorization", "Bearer "+tokenString)
		r.ServeHTTP(w, c.Request)
		
		assert.Equal(t, http.StatusOK, w.Code)
	})
	
	t.Run("Missing Header", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, r := gin.CreateTestContext(w)
		
		r.GET("/test", EnrollmentAuth(secret))
		
		c.Request, _ = http.NewRequest(http.MethodGet, "/test", nil)
		r.ServeHTTP(w, c.Request)
		
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "Authorization header required")
	})
	
	t.Run("Invalid Token Format", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, r := gin.CreateTestContext(w)
		
		r.GET("/test", EnrollmentAuth(secret))
		
		c.Request, _ = http.NewRequest(http.MethodGet, "/test", nil)
		c.Request.Header.Set("Authorization", "InvalidToken")
		r.ServeHTTP(w, c.Request)
		
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "Invalid token format")
	})
	
	t.Run("Wrong Secret", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, r := gin.CreateTestContext(w)
		
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"eid": uuid.New().String(),
			"cid": uuid.New().String(),
			"role": "ExamCandidate",
		})
		tokenString, _ := token.SignedString([]byte("wrong-secret"))
		
		r.GET("/test", EnrollmentAuth(secret))
		
		c.Request, _ = http.NewRequest(http.MethodGet, "/test", nil)
		c.Request.Header.Set("Authorization", "Bearer "+tokenString)
		r.ServeHTTP(w, c.Request)
		
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Expired Token", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, r := gin.CreateTestContext(w)

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"eid":  uuid.New().String(),
			"cid":  uuid.New().String(),
			"xid":  uuid.New().String(),
			"ent":  uuid.New().String(),
			"role": "ExamCandidate",
			"iat":  time.Now().Add(-2 * time.Hour).Unix(),
			"exp":  time.Now().Add(-1 * time.Hour).Unix(),
		})
		tokenString, _ := token.SignedString([]byte(secret))

		r.GET("/test", EnrollmentAuth(secret))

		c.Request, _ = http.NewRequest(http.MethodGet, "/test", nil)
		c.Request.Header.Set("Authorization", "Bearer "+tokenString)
		r.ServeHTTP(w, c.Request)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "Invalid or expired enrollment token")
	})
}

func TestTryEnrollmentOrUserAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	enrollmentSecret := "enrollment-secret"
	userSecret := "user-secret"
	
	t.Run("Valid Enrollment Token", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, r := gin.CreateTestContext(w)
		
		cid := uuid.New().String()
		ent := uuid.New().String()
		
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"eid": uuid.New().String(),
			"cid": cid,
			"xid": uuid.New().String(),
			"ent": ent,
			"role": "ExamCandidate",
		})
		tokenString, _ := token.SignedString([]byte(enrollmentSecret))
		
		r.GET("/test", TryEnrollmentOrUserAuth(enrollmentSecret, userSecret), func(c *gin.Context) {
			claims, _ := c.Get(string(UserContextKey))
			userClaims := claims.(*UserClaims)
			assert.Equal(t, domain.RoleExamCandidate, userClaims.Role)
			c.Status(http.StatusOK)
		})
		
		c.Request, _ = http.NewRequest(http.MethodGet, "/test", nil)
		c.Request.Header.Set("Authorization", "Bearer "+tokenString)
		r.ServeHTTP(w, c.Request)
		
		assert.Equal(t, http.StatusOK, w.Code)
	})
	
	t.Run("Valid User Token", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, r := gin.CreateTestContext(w)
		
		uid := uuid.New().String()
		
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub": uid,
			"role": string(domain.RoleEnterpriseAdmin),
			"enterpriseId": uuid.New().String(),
		})
		tokenString, _ := token.SignedString([]byte(userSecret))
		
		r.GET("/test", TryEnrollmentOrUserAuth(enrollmentSecret, userSecret), func(c *gin.Context) {
			claims, _ := c.Get(string(UserContextKey))
			userClaims := claims.(*UserClaims)
			assert.Equal(t, domain.RoleEnterpriseAdmin, userClaims.Role)
			c.Status(http.StatusOK)
		})
		
		c.Request, _ = http.NewRequest(http.MethodGet, "/test", nil)
		c.Request.Header.Set("Authorization", "Bearer "+tokenString)
		r.ServeHTTP(w, c.Request)
		
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
