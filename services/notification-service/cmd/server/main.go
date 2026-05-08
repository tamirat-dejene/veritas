package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tamirat-dejene/veritas/services/notification-service/internal/config"
	"github.com/tamirat-dejene/veritas/services/notification-service/internal/infrastructure/mailer"
	"github.com/tamirat-dejene/veritas/services/notification-service/internal/infrastructure/messaging"
	"github.com/tamirat-dejene/veritas/services/notification-service/internal/usecase"
	"github.com/tamirat-dejene/veritas/shared/pkg/logger"
	"github.com/tamirat-dejene/veritas/shared/pkg/messaging/kafka"
	"go.uber.org/zap"
)

func main() {
	// 1. Initialize Logger
	log, err := logger.NewLogger("notification-service")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	// 2. Load Configuration
	cfg := config.Load()

	// 3. Initialize Mailer
	smtpMailer := mailer.NewSMTPMailer(cfg)

	// 4. Initialize Usecase
	notificationUC, err := usecase.NewNotificationUsecase(smtpMailer, log)
	if err != nil {
		log.Fatal("failed to initialize notification usecase", zap.Error(err))
	}

	// 5. Initialize Kafka Subscriber
	kafkaSubscriber, err := kafka.NewSubscriber(kafka.Config{
		Brokers:       cfg.KafkaBrokers,
		ConsumerGroup: "notification-service-group",
	})
	if err != nil {
		log.Fatal("failed to initialize kafka subscriber", zap.Error(err))
	}
	defer kafkaSubscriber.Close()

	// 6. Setup Router & Start Consuming
	subRouter := messaging.NewNotificationRouter(notificationUC, log)

	consumerCtx, consumerCancel := context.WithCancel(context.Background())
	defer consumerCancel()

	go func() {
		log.Info("starting to consume Kafka messages...", zap.Strings("topics", subRouter.Topics()))
		for {
			select {
			case <-consumerCtx.Done():
				log.Info("consumer context cancelled, stopping retry loop")
				return
			default:
			}

			if err := kafkaSubscriber.Subscribe(consumerCtx, subRouter.Topics(), subRouter.Handle); err != nil {
				if consumerCtx.Err() != nil {
					log.Info("Kafka subscriber stopped due to context cancellation")
					return
				}
				log.Error("Kafka subscribe failed, retrying in 5s", zap.Error(err))
				select {
				case <-time.After(5 * time.Second):
				case <-consumerCtx.Done():
					return
				}
			}
		}
	}()

	// Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down notification-service...")
}
