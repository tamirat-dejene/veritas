package main

import (
	"context"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/tamirat-dejene/veritas/services/api-gateway/internal/config"
	"github.com/tamirat-dejene/veritas/services/api-gateway/internal/infrastructure"
	"github.com/tamirat-dejene/veritas/services/api-gateway/internal/logger"
	"github.com/tamirat-dejene/veritas/services/api-gateway/internal/router"
	"go.uber.org/zap"
)

func main() {
	log, err := logger.NewLogger()
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = log.Sync()
	}()
	zap.ReplaceGlobals(log)

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
		zap.L().Warn("Redis connection failed; rate limiting will fail open", zap.Error(err))
	} else {
		zap.L().Info("Redis connection established")
	}

	// Create rate limiter implementation (dependency injection)
	// Global rate limit: 100 requests per second
	rateLimiter := infrastructure.NewRedisRateLimiter(rdb, 100, time.Second)

	// Initialize Router with injected dependencies
	handler, err := router.NewRouter(cfg, rateLimiter)
	if err != nil {
		zap.L().Fatal("Failed to initialize router", zap.Error(err))
	}

	zap.L().Info("Service api-gateway starting", zap.String("port", cfg.Port))

	// Start Server
	if err := http.ListenAndServe(":"+cfg.Port, handler); err != nil {
		zap.L().Fatal("Failed to start server", zap.Error(err))
	}
}
