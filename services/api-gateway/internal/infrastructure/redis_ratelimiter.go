package infrastructure

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis_rate/v10"
	"github.com/redis/go-redis/v9"
	"github.com/tamirat-dejene/veritas/services/api-gateway/internal/domain"
)

// RedisRateLimiter implements the domain.RateLimiter interface using Redis
type RedisRateLimiter struct {
	limiter *redis_rate.Limiter
	limit   int
	window  time.Duration
}

// NewRedisRateLimiter creates a new Redis-based rate limiter
func NewRedisRateLimiter(rdb *redis.Client, limit int, window time.Duration) *RedisRateLimiter {
	return &RedisRateLimiter{
		limiter: redis_rate.NewLimiter(rdb),
		limit:   limit,
		window:  window,
	}
}

// Allow checks if a request is allowed for the given key.
func (r *RedisRateLimiter) Allow(ctx context.Context, key string) (*domain.RateLimitResult, error) {
	rate := redis_rate.Limit{
		Rate:   r.limit,
		Burst:  r.limit, // burst equals rate: no micro-burst above the window rate
		Period: r.window,
	}

	res, err := r.limiter.Allow(ctx, key, rate)
	if err != nil {
		return nil, fmt.Errorf("rate limiter error: %w", err)
	}

	// Convert redis_rate.Result to domain.RateLimitResult
	return &domain.RateLimitResult{
		Allowed:    res.Allowed,
		Remaining:  res.Remaining,
		ResetAfter: res.ResetAfter,
		RetryAfter: res.RetryAfter,
	}, nil
}
