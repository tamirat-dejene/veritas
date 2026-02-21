package main

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/tamirat-dejene/veritas/services/api-gateway/internal/config"
	"github.com/tamirat-dejene/veritas/services/api-gateway/internal/infrastructure"
	"github.com/tamirat-dejene/veritas/services/api-gateway/internal/router"
	redisShared "github.com/tamirat-dejene/veritas/shared/pkg/caching/redis"
	"github.com/tamirat-dejene/veritas/shared/pkg/logger"
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

	// Initialize Shared Redis Client
	host, port := parseAddr(cfg.RedisAddr)
	rdbClient, err := redisShared.NewRedisClient(host, port, cfg.RedisPassword, cfg.RedisDB)
	if err != nil {
		zap.L().Fatal("Failed to initialize Redis", zap.Error(err))
	}

	// Test Redis connection
	ctx := context.Background()
	if err := rdbClient.Ping(ctx); err != nil {
		zap.L().Warn("Redis connection failed; rate limiting will fail open", zap.Error(err))
	} else {
		zap.L().Info("Redis connection established")
	}

	rdb := rdbClient.GetClient().(*redis.Client)

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

func parseAddr(addr string) (string, int) {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return addr, 6379
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return host, 6379
	}
	return host, port
}
