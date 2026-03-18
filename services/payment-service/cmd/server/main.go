// Package main is the entry-point for the Veritas payment-service.
//
//	@title			Veritas Payment Service API
//	@version		1.0
//	@description	Subscription plans, upgrades, invoice retrieval, payment history, and Stripe webhook processing.
//
//	@contact.name	Veritas Platform Team
//
//	@tag.name		payment
//	@tag.description	Payment and billing operations.
//	@tag.name		subscription
//	@tag.description	Subscription plan and upgrade operations.
//	@tag.name		webhook
//	@tag.description	External payment provider webhook endpoints.
//	@tag.name		system
//	@tag.description	Operational and health endpoints.
//
//	@schemes		http https
//	@BasePath	/api/v1

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
	_ "github.com/tamirat-dejene/veritas/services/payment-service/docs/swagger"
	"github.com/tamirat-dejene/veritas/services/payment-service/internal/config"
	"github.com/tamirat-dejene/veritas/services/payment-service/internal/handler"
	"github.com/tamirat-dejene/veritas/services/payment-service/internal/infrastructure/stripe"
	"github.com/tamirat-dejene/veritas/services/payment-service/internal/repository/postgres"
	"github.com/tamirat-dejene/veritas/services/payment-service/internal/router"
	"github.com/tamirat-dejene/veritas/services/payment-service/internal/usecase"
	"github.com/tamirat-dejene/veritas/shared/pkg/logger"
	"go.uber.org/zap"
)

func main() {
	// Initialize logger
	log, err := logger.NewLogger("payment-service")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	// Load configuration
	cfg := config.Load()

	// Database connection (using pgxpool directly)
	pool, err := pgxpool.New(context.Background(), cfg.DSN)
	if err != nil {
		log.Fatal("failed to connect to database", zap.Error(err))
	}
	defer pool.Close()
	log.Info("connected to postgres (via pgxpool)")

	// Wire dependencies
	subRepo := postgres.NewSubscriptionRepository(pool)
	billingRepo := postgres.NewBillingRepository(pool)
	payProvider := stripe.NewStripeProvider(cfg.StripeSecretKey, cfg.StripeWebhookSecret)

	payUsecase := usecase.NewPaymentUsecase(pool, subRepo, billingRepo, payProvider)
	payHandler := handler.NewPaymentHandler(payUsecase)

	r := router.NewRouter(payHandler)

	// Server setup
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	// Start server in a goroutine
	go func() {
		log.Info("starting payment-service", zap.String("port", cfg.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("failed to start server", zap.Error(err))
		}
	}()

	// Graceful shutdown
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
