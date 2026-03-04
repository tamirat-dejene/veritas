package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
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
		zap.NewNop().Fatal("failed to initialize logger", zap.Error(err))
	}
	defer func() {
		_ = log.Sync()
	}()
	zap.ReplaceGlobals(log)

	cfg := config.Load()

	// Initialize Shared Redis Client
	rdbClient, err := redisShared.NewRedisClient(cfg.RedisHost, cfg.RedisPort, cfg.RedisPassword, cfg.RedisDB)
	if err != nil {
		zap.L().Fatal("Failed to initialize Redis", zap.Error(err))
	}
	defer func() {
		if err := rdbClient.Close(); err != nil {
			zap.L().Warn("failed to close Redis client", zap.Error(err))
		}
	}()

	// Test Redis connection
	pingCtx, cancelPing := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelPing()
	if err := rdbClient.Ping(pingCtx); err != nil {
		zap.L().Warn("Redis connection failed; rate limiting will fail open", zap.Error(err))
	} else {
		zap.L().Info("Redis connection established")
	}

	rdb := rdbClient.GetClient().(*redis.Client)

	// Create rate limiter implementation (dependency injection)
	// Global rate limit: 10 requests per second
	rateLimiter := infrastructure.NewRedisRateLimiter(rdb, 10, time.Second)

	// Initialize Router with injected dependencies
	handler, err := router.NewRouter(cfg, rateLimiter)
	if err != nil {
		zap.L().Fatal("Failed to initialize router", zap.Error(err))
	}

	zap.L().Info("Service api-gateway starting", zap.String("port", cfg.API_Gateway_Port))

	server := &http.Server{
		Addr:              ":" + cfg.API_Gateway_Port,
		Handler:           handler,
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zap.L().Fatal("Failed to start server", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	zap.L().Info("Shutting down api-gateway...")

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()

	if err := server.Shutdown(shutdownCtx); err != nil {
		zap.L().Fatal("Server forced to shutdown", zap.Error(err))
	}

	zap.L().Info("api-gateway exited gracefully")
}
