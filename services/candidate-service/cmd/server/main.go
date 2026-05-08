// Package main is the entry-point for the Veritas candidate-service.
//
//	@title			Veritas Candidate Service API
//	@version		1.0
//	@description	Candidate lifecycle, enrollment, exam access, and session management service.
//
//	@contact.name	Veritas Platform Team
//
//	@tag.name		candidate
//	@tag.description	Candidate profile management endpoints.
//	@tag.name		enrollment
//	@tag.description	Exam enrollment and invitation management endpoints.
//	@tag.name		session
//	@tag.description	Session lifecycle, access validation, answers, and submission endpoints.
//	@tag.name		monitoring
//	@tag.description	Session monitoring and submission/result retrieval endpoints.
//	@tag.name		system
//	@tag.description	Operational and health endpoints.
//
//	@schemes	http https
//	@BasePath	/

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tamirat-dejene/veritas/services/candidate-service/internal/config"
	c_http "github.com/tamirat-dejene/veritas/services/candidate-service/internal/handler"
	"github.com/tamirat-dejene/veritas/services/candidate-service/internal/infrastructure/client"
	"github.com/tamirat-dejene/veritas/services/candidate-service/internal/infrastructure/token"
	"github.com/tamirat-dejene/veritas/services/candidate-service/internal/repository/postgres"
	"github.com/tamirat-dejene/veritas/services/candidate-service/internal/router"
	"github.com/tamirat-dejene/veritas/services/candidate-service/internal/usecase"
	infrasched "github.com/tamirat-dejene/veritas/services/candidate-service/internal/infrastructure/scheduler"
	"github.com/tamirat-dejene/veritas/shared/pkg/cronjob"
	"github.com/tamirat-dejene/veritas/shared/pkg/logger"
	"github.com/tamirat-dejene/veritas/shared/pkg/messaging/kafka"
	"github.com/tamirat-dejene/veritas/services/candidate-service/internal/infrastructure/messaging"
	"go.uber.org/zap"

	// Import generated swagger docs so the spec is registered at startup.
	_ "github.com/tamirat-dejene/veritas/services/candidate-service/docs/swagger"
)

func main() {
	// 1. Initialize Logger
	log, err := logger.NewLogger("candidate-service")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = log.Sync() }()

	// 2. Load Configuration
	cfg := config.Load()

	// 3. Initialize pgxpool
	pool, err := pgxpool.New(context.Background(), cfg.DSN)
	if err != nil {
		log.Fatal("failed to connect to database", zap.Error(err))
	}
	defer pool.Close()
	log.Info("connected to postgres (via pgxpool)")

	// 4. Initialize Kafka producer for publishing enrollment events
	kafkaBrokers := strings.Split(cfg.KafkaBrokers, ",")
	publisher, err := kafka.NewProducer(kafka.Config{
		Brokers: kafkaBrokers,
	})
	if err != nil {
		log.Fatal("failed to create Kafka producer", zap.Error(err))
	}
	defer func() { _ = publisher.Close() }()
	log.Info("connected to Kafka", zap.Strings("brokers", kafkaBrokers))

	// 5. Initialize Infrastructure Clients
	examClient := client.NewExamServiceClient(cfg.ExamServiceURL, 10*time.Second)

	// 6. Initialize Repositories
	candidateRepo := postgres.NewCandidateRepository(pool)
	enrollmentRepo := postgres.NewEnrollmentRepository(pool)
	sessionRepo := postgres.NewSessionRepository(pool)

	// 7. Initialize Token Service
	tokenService := token.NewTokenService(cfg.EnrollmentTokenSecret)

	// 8. Initialize UseCases
	candidateUC := usecase.NewCandidateUseCase(pool, candidateRepo)
	enrollmentUC := usecase.NewEnrollmentUseCase(pool, enrollmentRepo, candidateRepo, tokenService, examClient, publisher, cfg.CandidatePortalBaseURL)
	sessionUC := usecase.NewSessionUseCase(pool, sessionRepo, enrollmentRepo, candidateRepo, examClient, tokenService, publisher)
	monitoringUC := usecase.NewMonitoringUseCase(sessionRepo)
	maintenanceUC := usecase.NewMaintenanceUseCase(sessionRepo, sessionUC, enrollmentRepo, log)
	consistencyUC := usecase.NewConsistencyUseCase(sessionRepo, enrollmentRepo, candidateRepo, log)

	// 8.5 Initialize Kafka Event Consumer
	eventConsumer := messaging.NewEventConsumer(consistencyUC, log.Named("event_consumer"))
	subscriber, err := kafka.NewSubscriber(kafka.Config{
		Brokers:       kafkaBrokers,
		ConsumerGroup: "candidate-service-group",
	})
	if err != nil {
		log.Fatal("failed to create Kafka consumer", zap.Error(err))
	}
	defer func() { _ = subscriber.Close() }()

	// 9. Initialize Handlers
	candidateHandler := c_http.NewCandidateHandler(candidateUC)
	enrollmentHandler := c_http.NewEnrollmentHandler(enrollmentUC)
	sessionHandler := c_http.NewSessionHandler(sessionUC)
	monitoringHandler := c_http.NewMonitoringHandler(monitoringUC)

	// 10. Initialize Router
	r := router.NewRouter(candidateHandler, enrollmentHandler, sessionHandler, monitoringHandler)

	// 11. Initialize Background Tasks
	scheduler := cronjob.NewScheduler(log.Named("scheduler"))
	infrasched.RegisterCandidateJobs(scheduler, maintenanceUC)
	scheduler.Start(context.Background())
	defer scheduler.Stop()

	// 11.5 Start Event Consumer
	if subscriber != nil {
		consumerCtx, consumerCancel := context.WithCancel(context.Background())
		defer consumerCancel()
		go func() {
			log.Info("kafka subscriber started", zap.Strings("topics", eventConsumer.Topics()))
			for {
				select {
				case <-consumerCtx.Done():
					return
				default:
				}

				if err := subscriber.Subscribe(consumerCtx, eventConsumer.Topics(), eventConsumer.Handle); err != nil {
					if consumerCtx.Err() != nil {
						return
					}
					log.Error("Kafka consumer stopped, retrying in 5s...", zap.Error(err))
					select {
					case <-time.After(5 * time.Second):
					case <-consumerCtx.Done():
						return
					}
					continue
				}
				break
			}
		}()
	}

	// 12. Start HTTP Server
	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	go func() {
		log.Info("Candidate Service starting", zap.String("port", cfg.Port))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("listen error", zap.Error(err))
		}
	}()

	// Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown", zap.Error(err))
	}

	log.Info("Server exiting")
}
