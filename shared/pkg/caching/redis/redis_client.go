package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis"
	"github.com/tamirat-dejene/veritas/shared/pkg/caching"
)

type redisClient struct {
	client *redis.Client
}

// Ping implements [caching.CacheClient].
func (r *redisClient) Ping(ctx context.Context) error {
	if err := r.client.Ping().Err(); err != nil {
		return fmt.Errorf("failed to ping redis: %w", err)
	}
	return nil
}

// Close implements [caching.CacheClient].
func (r *redisClient) Close() error {
	return r.client.Close()
}

// Decrement implements [caching.CacheClient].
func (r *redisClient) Decrement(ctx context.Context, key string) (int64, error) {
	newValue, err := r.client.Decr(key).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to decrement key %s: %w", key, err)
	}
	return newValue, nil
}

// Delete implements [caching.CacheClient].
func (r *redisClient) Delete(ctx context.Context, key string) error {
	if err := r.client.Del(key).Err(); err != nil {
		return fmt.Errorf("failed to delete key %s: %w", key, err)
	}
	return nil
}

// Exists implements [caching.CacheClient].
func (r *redisClient) Exists(ctx context.Context, key string) (bool, error) {
	exists, err := r.client.Exists(key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check existence of key %s: %w", key, err)
	}
	return exists > 0, nil
}

// Expire implements [caching.CacheClient].
func (r *redisClient) Expire(ctx context.Context, key string, expiration time.Duration) error {
	if err := r.client.Expire(key, expiration).Err(); err != nil {
		return fmt.Errorf("failed to set expiration for key %s: %w", key, err)
	}
	return nil
}

// Get implements [caching.CacheClient].
func (r *redisClient) Get(ctx context.Context, key string) (string, error) {
	value, err := r.client.Get(key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", fmt.Errorf("key %s does not exist", key)
		}
		return "", fmt.Errorf("failed to get key %s: %w", key, err)
	}
	return value, nil
}

// GetClient implements [caching.CacheClient].
func (r *redisClient) GetClient() any {
	return r.client
}

// Increment implements [caching.CacheClient].
func (r *redisClient) Increment(ctx context.Context, key string) (int64, error) {
	newValue, err := r.client.Incr(key).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to increment key %s: %w", key, err)
	}
	return newValue, nil
}

// Set implements [caching.CacheClient].
func (r *redisClient) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
	if err := r.client.Set(key, value, expiration).Err(); err != nil {
		return fmt.Errorf("failed to set key %s: %w", key, err)
	}
	return nil
}

// NewRedisClient initializes the Redis client
func NewRedisClient(host string, port int, password string, db int) (caching.CacheClient, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", host, port),
		Password: password,
		DB:       db,
	})

	return &redisClient{
		client: client,
	}, nil
}
