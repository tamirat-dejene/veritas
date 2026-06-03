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

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/tamirat-dejene/veritas/services/payment-service/docs/swagger"
	"github.com/tamirat-dejene/veritas/services/payment-service/internal/config"
	"github.com/tamirat-dejene/veritas/services/payment-service/internal/domain"
	"github.com/tamirat-dejene/veritas/services/payment-service/internal/handler"
	chapaprovider "github.com/tamirat-dejene/veritas/services/payment-service/internal/infrastructure/chapa"
	"github.com/tamirat-dejene/veritas/services/payment-service/internal/infrastructure/messaging"
	"github.com/tamirat-dejene/veritas/services/payment-service/internal/infrastructure/providerregistry"
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
	defer func() { _ = log.Sync() }()
	zap.ReplaceGlobals(log)

	// 2. Load configuration
	cfg := config.Load()

	// 3. Database connection
	pool, err := pgxpool.New(context.Background(), cfg.DSN)
	if err != nil {
		log.Fatal("failed to connect to database", zap.Error(err))
	}
	defer pool.Close()
	log.Info("connected to postgres (via pgxpool)")

	// 4. Wire repositories, Stripe & Chapa providers, and the registry
	subRepo := postgres.NewSubscriptionRepository(pool)
	billingRepo := postgres.NewBillingRepository(pool)

	stripeProv := stripeprovider.NewStripeProvider(cfg.StripeSecretKey, cfg.StripeWebhookSecret, cfg.StripeSuccessURL, cfg.StripeCancelURL)
	chapaProv := chapaprovider.NewChapaProvider(cfg.ChapaSecretKey, cfg.ChapaWebhookSecret, cfg.ChapaReturnURL, cfg.ChapaCallbackURL)
	provRegistry := providerregistry.NewProviderRegistry(stripeProv, chapaProv)

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

	// 5b. Kafka subscriber (consistency consumer — graceful degradation)
	kafkaSubscriber, subErr := kafka.NewSubscriber(kafka.Config{
		Brokers:       cfg.KafkaBrokers,
		ConsumerGroup: "payment-service-group",
	})
	if subErr != nil {
		log.Error("failed to initialize kafka subscriber — consistency events will not be processed", zap.Error(subErr))
	} else {
		defer kafkaSubscriber.Close()
	}

	// 6. Wire usecase and handler
	payUsecase := usecase.NewPaymentUsecase(pool, subRepo, billingRepo, provRegistry, eventPublisher)
	maintenanceUC := usecase.NewMaintenanceUseCase(subRepo, billingRepo, eventPublisher, log)
	consistencyUC := usecase.NewConsistencyUseCase(subRepo, billingRepo, log)
	payHandler := handler.NewPaymentHandler(payUsecase)

	// 7. Initialize Scheduler & Register Background Jobs
	cronScheduler := cronjob.NewScheduler(log.Named("scheduler"))
	scheduler.RegisterPaymentJobs(cronScheduler, maintenanceUC)
	cronScheduler.Start(context.Background())
	defer cronScheduler.Stop()

	// 7b. Start Kafka consistency consumer goroutine
	if kafkaSubscriber != nil {
		eventConsumer := messaging.NewEventConsumer(consistencyUC, log)
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
				if err := kafkaSubscriber.Subscribe(consumerCtx, eventConsumer.Topics(), eventConsumer.Handle); err != nil {
					if consumerCtx.Err() != nil {
						return
					}
					log.Error("kafka subscriber exited, retrying in 5s...", zap.Error(err))
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
