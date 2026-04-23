// Package main is the entry-point for the Veritas enterprise-service.
//
//	@title			Veritas Enterprise Service API
//	@version		1.0
//	@description	Enterprise onboarding, account management, and user administration service.
//
//	@contact.name	Veritas Platform Team
//
//	@tag.name		enterprise
//	@tag.description	Enterprise profile and lifecycle management endpoints.
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
	"github.com/tamirat-dejene/veritas/services/enterprise-service/internal/infrastructure/messaging"
	"github.com/tamirat-dejene/veritas/services/enterprise-service/internal/infrastructure/client"
	"github.com/tamirat-dejene/veritas/services/enterprise-service/internal/repository/postgres"
	"github.com/tamirat-dejene/veritas/services/enterprise-service/internal/router"
	"github.com/tamirat-dejene/veritas/services/enterprise-service/internal/usecase"
	"github.com/tamirat-dejene/veritas/shared/pkg/logger"
	"github.com/tamirat-dejene/veritas/shared/pkg/messaging/kafka"
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

	// 5. Messaging: Kafka producer (for enterprise events)
	kafkaProducer, err := kafka.NewProducer(kafka.Config{
		Brokers: cfg.KafkaBrokers,
	})
	if err != nil {
		log.Fatal("failed to initialize kafka producer", zap.Error(err))
	}
	defer kafkaProducer.Close()
	eventPublisher := messaging.NewKafkaPublisher(kafkaProducer)

	// 6. Payment-service HTTP client (for live subscription enrichment)
	payClient := client.NewPaymentClient(cfg.PaymentServiceURL)

	// 7. Initialize Usecases
	enterpriseUC := usecase.NewEnterpriseUsecase(pool, userRepo, enterpriseRepo, auditRepo, eventPublisher, payClient)
	userUC := usecase.NewUserUsecase(pool, userRepo, enterpriseRepo, auditRepo)

	// 8. Initialize Handlers
	enterpriseHandler := handler.NewEnterpriseHandler(enterpriseUC)
	userHandler := handler.NewUserHandler(userUC)

	// 9. Initialize Router
	r := router.NewRouter(enterpriseHandler, userHandler)

	// 10. Start HTTP Server
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

	// 11. Start Kafka consumer for payment-service events
	kafkaSubscriber, err := kafka.NewSubscriber(kafka.Config{
		Brokers:       cfg.KafkaBrokers,
		ConsumerGroup: "enterprise-service",
	})
	if err != nil {
		log.Error("failed to initialize kafka subscriber — payment suspension events will not be processed", zap.Error(err))
	} else {
		defer kafkaSubscriber.Close()
		subRouter := messaging.NewSubscriptionRouter(enterpriseUC, log)
		consumerCtx, consumerCancel := context.WithCancel(context.Background())
		defer consumerCancel()
		go func() {
			if err := kafkaSubscriber.Subscribe(
				consumerCtx,
				subRouter.Topics(),
				subRouter.Handle,
			); err != nil {
				log.Error("kafka subscriber exited", zap.Error(err))
			}
		}()
		log.Info("kafka subscriber started", zap.Strings("topics", subRouter.Topics()))
	}

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
