package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RateLimiter is a placeholder middleware stub for future Redis-backed rate limiting.
// TODO: Implement sliding-window rate limiting using the shared Redis client.
// Each unique IP (or user ID post-auth) should be tracked with an increment + expire pattern.
func RateLimiter(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Placeholder: log the intent and pass through.
		log.Debug("rate limiter (stub): request passed",
			zap.String("ip", c.ClientIP()),
			zap.String("path", c.FullPath()),
		)

		// Future implementation will check a Redis counter and abort with 429
		// if the limit is exceeded:
		// if exceeded {
		//     c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests"})
		//     return
		// }

		_ = http.StatusTooManyRequests // suppress unused import warning
		c.Next()
	}
}
