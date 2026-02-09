package domain

import (
	"context"
	"time"
)

// RateLimitResult contains the result of a rate limit check
type RateLimitResult struct {
	Allowed    int           // Number of requests allowed (0 if rate limited)
	Remaining  int           // Number of requests remaining in the current window
	ResetAfter time.Duration // Duration until the rate limit resets
	RetryAfter time.Duration // Duration to wait before retrying (when rate limited)
}

// RateLimiter defines the interface for rate limiting operations
type RateLimiter interface {
	// Allow checks if a request is allowed for the given key
	// Returns the rate limit result and any error encountered
	Allow(ctx context.Context, key string) (*RateLimitResult, error)
}
