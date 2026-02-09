package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/tamirat-dejene/veritas/services/api-gateway/internal/config"
	"github.com/tamirat-dejene/veritas/services/api-gateway/internal/infrastructure"
	"github.com/tamirat-dejene/veritas/services/api-gateway/internal/router"
)

func main() {
	cfg := config.Load()

	// Initialize Redis Client
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	// Test Redis connection
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Redis connection failed: %v - rate limiting will fail open", err)
	} else {
		log.Println("Redis connection established successfully")
	}

	// Create rate limiter implementation (dependency injection)
	// Global rate limit: 100 requests per second
	rateLimiter := infrastructure.NewRedisRateLimiter(rdb, 100, time.Second)

	// Initialize Router with injected dependencies
	handler, err := router.NewRouter(cfg, rateLimiter)
	if err != nil {
		log.Fatalf("Failed to initialize router: %v", err)
	}

	log.Printf("Service api-gateway starting on port %s", cfg.Port)

	// Start Server
	if err := http.ListenAndServe(":"+cfg.Port, handler); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}