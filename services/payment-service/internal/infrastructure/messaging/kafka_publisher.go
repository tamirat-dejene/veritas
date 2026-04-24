package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/services/payment-service/internal/domain"
	"github.com/tamirat-dejene/veritas/shared/pkg/messaging"
	"github.com/tamirat-dejene/veritas/shared/pkg/messaging/topics"
)

type kafkaPublisher struct {
	publisher messaging.Publisher
}

// NewKafkaPublisher creates a PaymentEventPublisher backed by Kafka.
func NewKafkaPublisher(pub messaging.Publisher) domain.PaymentEventPublisher {
	return &kafkaPublisher{publisher: pub}
}

// paymentFailedEvent is the payload published on topics.SubscriptionPaymentFailed.
type paymentFailedEvent struct {
	EnterpriseID uuid.UUID `json:"enterprise_id"`
	Timestamp    int64     `json:"timestamp"`
}

// PublishPaymentFailed publishes a subscription.payment.failed event so that
// enterprise-service can suspend the enterprise asynchronously.
func (p *kafkaPublisher) PublishPaymentFailed(ctx context.Context, enterpriseID uuid.UUID) error {
	event := paymentFailedEvent{
		EnterpriseID: enterpriseID,
		Timestamp:    time.Now().Unix(),
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("kafka_publisher: marshal payment_failed event: %w", err)
	}

	msg := messaging.Message{
		Topic: topics.SubscriptionPaymentFailed,
		Key:   enterpriseID[:],
		Value: payload,
	}

	if err := p.publisher.Publish(ctx, msg); err != nil {
		return fmt.Errorf("kafka_publisher: publish payment_failed: %w", err)
	}

	return nil
}

// subscriptionEvent is the payload published on topics.SubscriptionUpdated and topics.SubscriptionCanceled.
type subscriptionEvent struct {
	EnterpriseID uuid.UUID `json:"enterprise_id"`
	Timestamp    int64     `json:"timestamp"`
}

func (p *kafkaPublisher) PublishSubscriptionUpdated(ctx context.Context, enterpriseID uuid.UUID) error {
	event := subscriptionEvent{
		EnterpriseID: enterpriseID,
		Timestamp:    time.Now().Unix(),
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("kafka_publisher: marshal subscription_updated event: %w", err)
	}

	msg := messaging.Message{
		Topic: topics.SubscriptionUpdated,
		Key:   enterpriseID[:],
		Value: payload,
	}

	if err := p.publisher.Publish(ctx, msg); err != nil {
		return fmt.Errorf("kafka_publisher: publish subscription_updated: %w", err)
	}

	return nil
}

func (p *kafkaPublisher) PublishSubscriptionCanceled(ctx context.Context, enterpriseID uuid.UUID) error {
	event := subscriptionEvent{
		EnterpriseID: enterpriseID,
		Timestamp:    time.Now().Unix(),
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("kafka_publisher: marshal subscription_canceled event: %w", err)
	}

	msg := messaging.Message{
		Topic: topics.SubscriptionCanceled,
		Key:   enterpriseID[:],
		Value: payload,
	}

	if err := p.publisher.Publish(ctx, msg); err != nil {
		return fmt.Errorf("kafka_publisher: publish subscription_canceled: %w", err)
	}

	return nil
}
