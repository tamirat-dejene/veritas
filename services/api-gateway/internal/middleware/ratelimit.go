package middleware

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/tamirat-dejene/veritas/services/api-gateway/internal/domain"
	"go.uber.org/zap"
)

// RateLimitMiddleware wraps a domain.RateLimiter for HTTP middleware
type RateLimitMiddleware struct {
	limiter domain.RateLimiter
}

// NewRateLimitMiddleware creates a new rate limit middleware
func NewRateLimitMiddleware(limiter domain.RateLimiter) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		limiter: limiter,
	}
}

// extractIP extracts the real client IP from the request
// It checks X-Forwarded-For and X-Real-IP headers for proxy scenarios
func extractIP(r *http.Request) string {
	// Check X-Forwarded-For header (can contain multiple IPs)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the list
		if ip, _, err := net.SplitHostPort(xff); err == nil {
			return ip
		}
		// If no port, return as is
		return xff
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fallback to RemoteAddr
	if ip, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return ip
	}

	return r.RemoteAddr
}

// Handler returns an HTTP middleware handler
func (m *RateLimitMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.Background()
		ip := extractIP(r)

		// Create a unique key for this IP
		key := "ratelimit:" + ip

		// Check rate limit using the domain interface
		result, err := m.limiter.Allow(ctx, key)
		if err != nil {
			// If rate limiter fails, log the error and allow the request (fail open)
			zap.L().Warn("Rate limiter error; allowing request", zap.Error(err))
			next.ServeHTTP(w, r)
			return
		}

		// Set rate limit headers
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(result.ResetAfter).Unix(), 10))

		// Check if rate limit exceeded
		if result.Allowed == 0 {
			w.Header().Set("Retry-After", strconv.FormatInt(int64(result.RetryAfter.Seconds()), 10))
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}
