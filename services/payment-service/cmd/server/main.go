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
//	@BasePath	/

package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/tamirat-dejene/veritas/services/payment-service/docs/swagger"
	"github.com/tamirat-dejene/veritas/services/payment-service/internal/config"
	"github.com/tamirat-dejene/veritas/services/payment-service/internal/handler"
	"github.com/tamirat-dejene/veritas/services/payment-service/internal/infrastructure/stripe"
	"github.com/tamirat-dejene/veritas/services/payment-service/internal/repository/postgres"
	"github.com/tamirat-dejene/veritas/services/payment-service/internal/router"
	"github.com/tamirat-dejene/veritas/services/payment-service/internal/usecase"
	pg_client "github.com/tamirat-dejene/veritas/shared/db/pg"
	"go.uber.org/zap"
)

func main() {
	// Initialize logger
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// Load configuration
	cfg := config.Load()

	// Database connection
	db, err := pg_client.NewPostgresClient(cfg.DSN)
	if err != nil {
		logger.Fatal("failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// Wire dependencies
	subRepo := postgres.NewSubscriptionRepository(db)
	billingRepo := postgres.NewBillingRepository(db)
	payProvider := stripe.NewStripeProvider(cfg.StripeSecretKey, cfg.StripeWebhookSecret)

	payUsecase := usecase.NewPaymentUsecase(subRepo, billingRepo, payProvider)
	payHandler := handler.NewPaymentHandler(payUsecase)

	r := router.NewRouter(payHandler)

	// Server setup
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("starting payment-service", zap.String("port", cfg.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("failed to start server", zap.Error(err))
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down payment-service...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("server forced to shutdown", zap.Error(err))
	}

	logger.Info("payment-service exited gracefully")
}
