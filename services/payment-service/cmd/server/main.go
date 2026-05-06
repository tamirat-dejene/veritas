// Package main is the entry-point for the Veritas payment-service.
//
//	@title			Veritas Payment Service API
//	@version		1.0
//	@description	Subscription plans, upgrades, cancellations, invoice retrieval, payment history, and Stripe webhook processing.
//
//	@contact.name	Veritas Platform Team
//
//	@tag.name		payment
//	@tag.description	Payment and billing operations.
//	@tag.name		subscription
//	@tag.description	Subscription plan and management operations.
//	@tag.name		webhook
//	@tag.description	External payment provider webhook endpoints.
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

	_ "github.com/tamirat-dejene/veritas/services/payment-service/docs/swagger"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tamirat-dejene/veritas/services/payment-service/internal/config"
	"github.com/tamirat-dejene/veritas/services/payment-service/internal/domain"
	"github.com/tamirat-dejene/veritas/services/payment-service/internal/handler"
	"github.com/tamirat-dejene/veritas/services/payment-service/internal/infrastructure/messaging"
	"github.com/tamirat-dejene/veritas/services/payment-service/internal/infrastructure/scheduler"
	stripeprovider "github.com/tamirat-dejene/veritas/services/payment-service/internal/infrastructure/stripe"
	"github.com/tamirat-dejene/veritas/services/payment-service/internal/repository/postgres"
	"github.com/tamirat-dejene/veritas/services/payment-service/internal/router"
	"github.com/tamirat-dejene/veritas/services/payment-service/internal/usecase"
	"github.com/tamirat-dejene/veritas/shared/pkg/cronjob"
	"github.com/tamirat-dejene/veritas/shared/pkg/logger"
	"github.com/tamirat-dejene/veritas/shared/pkg/messaging/kafka"
	"go.uber.org/zap"
)

func main() {
	// 1. Initialize logger
	log, err := logger.NewLogger("payment-service")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	// 2. Load configuration
	cfg := config.Load()

	// 3. Database connection
	pool, err := pgxpool.New(context.Background(), cfg.DSN)
	if err != nil {
		log.Fatal("failed to connect to database", zap.Error(err))
	}
	defer pool.Close()
	log.Info("connected to postgres (via pgxpool)")

	// 4. Wire repositories and Stripe provider
	subRepo := postgres.NewSubscriptionRepository(pool)
	billingRepo := postgres.NewBillingRepository(pool)
	payProvider := stripeprovider.NewStripeProvider(cfg.StripeSecretKey, cfg.StripeWebhookSecret, cfg.StripeSuccessURL, cfg.StripeCancelURL)

	// 5. Wire Kafka event publisher (graceful degradation if Kafka is unavailable)
	var eventPublisher domain.PaymentEventPublisher
	kafkaProducer, err := kafka.NewProducer(kafka.Config{Brokers: cfg.KafkaBrokers})
	if err != nil {
		log.Error("kafka producer unavailable — payment.failed events will NOT reach enterprise-service", zap.Error(err))
	} else {
		defer kafkaProducer.Close()
		eventPublisher = messaging.NewKafkaPublisher(kafkaProducer)
		log.Info("kafka producer initialized")
	}

	// 6. Wire usecase and handler
	payUsecase := usecase.NewPaymentUsecase(pool, subRepo, billingRepo, payProvider, eventPublisher)
	maintenanceUC := usecase.NewMaintenanceUseCase(subRepo, billingRepo, eventPublisher, log)
	payHandler := handler.NewPaymentHandler(payUsecase)

	// 7. Initialize Scheduler & Register Background Jobs
	cronScheduler := cronjob.NewScheduler(log.Named("scheduler"))
	scheduler.RegisterPaymentJobs(cronScheduler, maintenanceUC)
	cronScheduler.Start(context.Background())
	defer cronScheduler.Stop()

	// 8. Build router and start HTTP server
	r := router.NewRouter(payHandler)
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	go func() {
		log.Info("starting payment-service", zap.String("port", cfg.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("failed to start server", zap.Error(err))
		}
	}()

	// 9. Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down payment-service...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("server forced to shutdown", zap.Error(err))
	}

	log.Info("payment-service exited gracefully")
}
