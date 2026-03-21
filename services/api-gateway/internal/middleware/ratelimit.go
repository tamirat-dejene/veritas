package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tamirat-dejene/veritas/services/api-gateway/internal/domain"
	"go.uber.org/zap"
)

// RateLimitMiddleware wraps a domain.RateLimiter for HTTP middleware
type RateLimitMiddleware struct {
	limiter domain.RateLimiter
	limit   int // total allowed requests per window — used for X-RateLimit-Limit header
}

// NewRateLimitMiddleware creates a new rate limit middleware.
// limit is the maximum number of requests allowed per window (used for response headers).
func NewRateLimitMiddleware(limiter domain.RateLimiter, limit int) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		limiter: limiter,
		limit:   limit,
	}
}

// Handler returns a Gin middleware handler
func (m *RateLimitMiddleware) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		ip := c.ClientIP()

		// Create a unique key for this IP
		key := "ratelimit:" + ip

		// Check rate limit using the domain interface
		result, err := m.limiter.Allow(ctx, key)
		if err != nil {
			// If rate limiter fails, log the error and allow the request (fail open)
			zap.L().Warn("Rate limiter error; allowing request", zap.Error(err))
			c.Next()
			return
		}

		c.Header("X-RateLimit-Limit", strconv.Itoa(m.limit))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(result.ResetAfter).Unix(), 10))

		// Check if rate limit exceeded
		if result.Allowed == 0 {
			zap.L().Warn("Rate limit exceeded", zap.String("ip", ip), zap.Duration("retry_after", result.RetryAfter))
			c.Header("Retry-After", strconv.FormatInt(int64(result.RetryAfter.Seconds()), 10))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "Too Many Requests"})
			return
		}

		c.Next()
	}
}
