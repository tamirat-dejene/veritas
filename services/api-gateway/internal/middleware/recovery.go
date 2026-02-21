package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func Recoverer() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				requestID := c.GetString(RequestIDKey)
				if requestID == "" {
					requestID = "-"
				}
				zap.L().Error("panic recovered", zap.String("request_id", requestID), zap.Any("panic", rec))
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
			}
		}()

		c.Next()
	}
}
