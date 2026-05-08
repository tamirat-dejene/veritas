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
	"sync"
	"syscall"
	"time"

	"github.com/tamirat-dejene/veritas/services/enterprise-service/internal/config"
	"github.com/tamirat-dejene/veritas/services/enterprise-service/internal/handler"
	"github.com/tamirat-dejene/veritas/services/enterprise-service/internal/infrastructure/client"
	"github.com/tamirat-dejene/veritas/services/enterprise-service/internal/infrastructure/messaging"
	"github.com/tamirat-dejene/veritas/services/enterprise-service/internal/infrastructure/scheduler"
	"github.com/tamirat-dejene/veritas/services/enterprise-service/internal/repository/postgres"
	"github.com/tamirat-dejene/veritas/services/enterprise-service/internal/router"
	"github.com/tamirat-dejene/veritas/services/enterprise-service/internal/usecase"
	"github.com/tamirat-dejene/veritas/shared/pkg/cronjob"
	"github.com/tamirat-dejene/veritas/shared/pkg/logger"
	"github.com/tamirat-dejene/veritas/shared/pkg/messaging/kafka"
	"go.uber.org/zap"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tamirat-dejene/veritas/shared/pkg/storage/cloudinary"

	// Import generated swagger docs so the spec is registered at startup.
	_ "github.com/tamirat-dejene/veritas/services/enterprise-service/docs/swagger"
)

func main() {
	// 1. Logger
	log, err := logger.NewLogger("enterprise-service")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	// 2. Config
	cfg := config.Load()

	// 3. Database — cancel setup context immediately after use
	ctxSetup, cancelSetup := context.WithTimeout(context.Background(), 10*time.Second)
	poolCfg, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		cancelSetup()
		log.Fatal("failed to parse database config", zap.Error(err))
	}
	poolCfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	pool, err := pgxpool.NewWithConfig(ctxSetup, poolCfg)
	cancelSetup()
	if err != nil {
		log.Fatal("failed to connect to database", zap.Error(err))
	}
	defer pool.Close()

	// 4. Repositories
	userRepo := postgres.NewUserRepository(pool)
	enterpriseRepo := postgres.NewEnterpriseRepository(pool)
	auditRepo := postgres.NewAuditRepository(pool)
	passwordResetRepo := postgres.NewPasswordResetRepository(pool)

	// 5. Kafka Producer
	kafkaProducer, err := kafka.NewProducer(kafka.Config{Brokers: cfg.KafkaBrokers})
	if err != nil {
		log.Fatal("failed to initialize kafka producer", zap.Error(err))
	}
	defer kafkaProducer.Close()
	eventPublisher := messaging.NewKafkaPublisher(kafkaProducer)

	// 6. HTTP Clients & Storage
	payClient := client.NewPaymentClient(cfg.PaymentServiceURL)
	examClient := client.NewExamClient(cfg.ExamServiceURL)
	candidateClient := client.NewCandidateClient(cfg.CandidateServiceURL)
	logoStorage, err := cloudinary.NewCloudinaryStorage(
		cfg.CloudinaryCloudName,
		cfg.CloudinaryAPIKey,
		cfg.CloudinaryAPISecret,
		cfg.CloudinaryLogosFolder,
	)
	if err != nil {
		log.Fatal("failed to initialize cloudinary storage", zap.Error(err))
	}

	// 7. Usecases
	enterpriseUC := usecase.NewEnterpriseUsecase(pool, userRepo, enterpriseRepo, auditRepo, eventPublisher, payClient, examClient, candidateClient, logoStorage)
	userUC := usecase.NewUserUsecase(pool, userRepo, enterpriseRepo, auditRepo, eventPublisher, passwordResetRepo, cfg.FrontendBaseURL)
	maintenanceUC := usecase.NewMaintenanceUseCase(userRepo, passwordResetRepo, enterpriseRepo, enterpriseUC, auditRepo, log)

	// 8. Scheduler
	cronScheduler := cronjob.NewScheduler(log)
	scheduler.RegisterEnterpriseJobs(cronScheduler, maintenanceUC)
	cronScheduler.Start(context.Background())
	defer cronScheduler.Stop()

	// 9. Handlers & Router
	enterpriseHandler := handler.NewEnterpriseHandler(enterpriseUC)
	userHandler := handler.NewUserHandler(userUC)
	r := router.NewRouter(enterpriseHandler, userHandler)

	// Shared shutdown primitives
	consumerCtx, consumerCancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	// 10. HTTP Server
	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}
	serverErr := make(chan error, 1)
	go func() {
		log.Info("enterprise-service starting", zap.String("port", cfg.Port))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	// 11. Kafka Consumer
	kafkaSubscriber, err := kafka.NewSubscriber(kafka.Config{
		Brokers:       cfg.KafkaBrokers,
		ConsumerGroup: "enterprise-service-group",
	})
	if err != nil {
		log.Error("failed to initialize kafka subscriber — payment suspension events will not be processed", zap.Error(err))
	} else {
		subRouter := messaging.NewSubscriptionRouter(enterpriseUC, log)
		wg.Go(func() {
			log.Info("kafka consumer starting", zap.Strings("topics", subRouter.Topics()))
			for {
				select {
				case <-consumerCtx.Done():
					log.Info("kafka consumer stopping")
					return
				default:
				}
				if err := kafkaSubscriber.Subscribe(consumerCtx, subRouter.Topics(), subRouter.Handle); err != nil {
					if consumerCtx.Err() != nil {
						log.Info("kafka consumer stopped cleanly")
						return
					}
					log.Error("kafka subscribe error, retrying in 5s", zap.Error(err))
					select {
					case <-time.After(5 * time.Second):
					case <-consumerCtx.Done():
						return
					}
				}
			}
		})
	}

	// Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		log.Info("shutdown signal received", zap.String("signal", sig.String()))
	case err := <-serverErr:
		log.Error("server error, initiating shutdown", zap.Error(err))
	}

	// 1. Stop accepting new HTTP requests
	httpCtx, httpCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer httpCancel()
	if err := server.Shutdown(httpCtx); err != nil {
		log.Error("http server forced shutdown", zap.Error(err))
	}

	// 2. Stop Kafka consumer
	consumerCancel()
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
		log.Info("kafka consumer stopped cleanly")
	case <-time.After(15 * time.Second):
		log.Warn("kafka consumer shutdown timed out")
	}

	// 3. Close Kafka subscriber connection 
	if kafkaSubscriber != nil {
		kafkaSubscriber.Close()
	}

	log.Info("shutdown complete")
}