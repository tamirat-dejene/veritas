// Package main is the entry-point for the Veritas enterprise-service.
//
//	@title			Veritas Enterprise Service API
//	@version		1.0
//	@description	Enterprise onboarding, account management, subscription, and user administration service.
//
//	@contact.name	Veritas Platform Team
//
//	@tag.name		enterprise
//	@tag.description	Enterprise profile and lifecycle management endpoints.
//	@tag.name		subscription
//	@tag.description	Subscription and billing lifecycle endpoints.
//	@tag.name		user
//	@tag.description	Enterprise user management endpoints.
//	@tag.name		system
//	@tag.description	Operational and health endpoints.
//
//	@schemes		http https
//	@BasePath	/

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tamirat-dejene/veritas/services/enterprise-service/internal/config"
	"github.com/tamirat-dejene/veritas/services/enterprise-service/internal/handler"
	"github.com/tamirat-dejene/veritas/services/enterprise-service/internal/repository/postgres"
	"github.com/tamirat-dejene/veritas/services/enterprise-service/internal/router"
	"github.com/tamirat-dejene/veritas/services/enterprise-service/internal/usecase"
	"github.com/tamirat-dejene/veritas/shared/pkg/logger"
	"go.uber.org/zap"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	// Import generated swagger docs so the spec is registered at startup.
	_ "github.com/tamirat-dejene/veritas/services/enterprise-service/docs/swagger"
)

func main() {
	// 1. Initialize Logger
	log, err := logger.NewLogger("enterprise-service")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	// 2. Load Configuration
	cfg := config.Load()

	// 3. Initialize Database Client
	ctxSetup, cancelSetup := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelSetup()

	poolCfg, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		log.Fatal("failed to parse database config", zap.Error(err))
	}
	poolCfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	pool, err := pgxpool.NewWithConfig(ctxSetup, poolCfg)
	if err != nil {
		log.Fatal("failed to connect to database", zap.Error(err))
	}
	defer pool.Close()

	// 4. Initialize Repositories
	userRepo := postgres.NewUserRepository(pool)
	enterpriseRepo := postgres.NewEnterpriseRepository(pool)
	auditRepo := postgres.NewAuditRepository(pool)

	// 5. Initialize Usecases
	enterpriseUC := usecase.NewEnterpriseUsecase(pool, userRepo, enterpriseRepo, auditRepo)
	userUC := usecase.NewUserUsecase(pool, userRepo, enterpriseRepo, auditRepo)

	// 6. Initialize Handlers
	enterpriseHandler := handler.NewEnterpriseHandler(enterpriseUC)
	subscriptionHandler := handler.NewSubscriptionHandler(enterpriseUC)
	userHandler := handler.NewUserHandler(userUC)

	// 7. Initialize Router
	r := router.NewRouter(enterpriseHandler, subscriptionHandler, userHandler)

	// 8. Start HTTP Server
	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	go func() {
		fmt.Printf("Enterprise Service starting on port %s...\n", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("listen error", zap.Error(err))
		}
	}()

	// Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown", zap.Error(err))
	}

	log.Info("Server exiting")
}
