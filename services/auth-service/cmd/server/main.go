// Package main is the entry-point for the Veritas auth-service.
//
//	@title			Veritas Auth Service API
//	@version		1.0
//	@description	JWT-based authentication service for the Veritas platform.
//	@description	Handles login, token refresh, and token revocation for privileged roles.
//
//	@contact.name	Veritas Platform Team
//
//	@tag.name		auth
//	@tag.description	Authentication endpoints for login, token refresh, and logout.
//	@tag.name		system
//	@tag.description	Operational and health endpoints.
//
//	@schemes		http https
//	@BasePath	/api/v1

package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tamirat-dejene/veritas/services/auth-service/internal/config"
	"github.com/tamirat-dejene/veritas/services/auth-service/internal/handler"
	inframsg "github.com/tamirat-dejene/veritas/services/auth-service/internal/infrastructure/messaging"
	infratoken "github.com/tamirat-dejene/veritas/services/auth-service/internal/infrastructure/token"
	pgRepo "github.com/tamirat-dejene/veritas/services/auth-service/internal/repository/postgres"
	"github.com/tamirat-dejene/veritas/services/auth-service/internal/router"
	"github.com/tamirat-dejene/veritas/services/auth-service/internal/usecase"
	postgres "github.com/tamirat-dejene/veritas/shared/db/pg"
	"github.com/tamirat-dejene/veritas/shared/pkg/logger"
	"github.com/tamirat-dejene/veritas/shared/pkg/messaging/kafka"
	"go.uber.org/zap"

	// Import generated swagger docs so the spec is registered at startup.
	_ "github.com/tamirat-dejene/veritas/services/auth-service/docs/swagger"
)

func main() {
	// --- Logger ---
	log, err := logger.NewLogger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = log.Sync() }()
	zap.ReplaceGlobals(log)

	// --- Config ---
	cfg := config.Load()

	// --- Database ---
	db, err := postgres.NewPostgresClient(cfg.DSN)
	if err != nil {
		log.Fatal("failed to connect to database", zap.Error(err))
	}
	defer db.Close()
	db.LogConnectionInfo()

	// --- Infrastructure: Token Services ---
	jwtService := infratoken.NewJWTService(cfg.JWTSecret, cfg.AccessTokenTTL)
	refreshService := infratoken.NewRefreshTokenService(cfg.RefreshTokenTTL)

	// --- Messaging: Kafka ---
	kafkaProducer, err := kafka.NewProducer(kafka.Config{
		Brokers: cfg.KafkaBrokers,
	})
	if err != nil {
		log.Fatal("failed to initialize kafka producer", zap.Error(err))
	}
	defer kafkaProducer.Close()

	eventPublisher := inframsg.NewKafkaPublisher(kafkaProducer)

	// --- Repositories ---
	userRepo := pgRepo.NewUserRepository(db)
	refreshTokenRepo := pgRepo.NewRefreshTokenRepository(db)

	// --- Use Cases ---
	loginUC := usecase.NewLoginUseCase(
		userRepo,
		refreshTokenRepo,
		jwtService,
		refreshService,
		cfg.AccessTokenTTL,
		cfg.RefreshTokenTTL,
		eventPublisher,
		log,
	)
	refreshUC := usecase.NewRefreshUseCase(
		userRepo,
		refreshTokenRepo,
		jwtService,
		refreshService,
		cfg.AccessTokenTTL,
		cfg.RefreshTokenTTL,
		log,
	)
	logoutUC := usecase.NewLogoutUseCase(refreshTokenRepo, log)

	// --- HTTP Layer ---
	authHandler := handler.NewAuthHandler(loginUC, refreshUC, logoutUC, log)
	engine := router.NewRouter(authHandler, log)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      engine,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine so we can listen for signals.
	go func() {
		log.Info("HTTP server listening", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("HTTP server error", zap.Error(err))
		}
	}()

	// --- Graceful Shutdown ---
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("server shutdown error", zap.Error(err))
	}
	log.Info("server stopped")
}
