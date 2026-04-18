package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/services/enterprise-service/internal/domain"
	"github.com/tamirat-dejene/veritas/shared/pkg/messaging"
	"github.com/tamirat-dejene/veritas/shared/pkg/messaging/topics"
)

type kafkaPublisher struct {
	publisher messaging.Publisher
}

// NewKafkaPublisher creates a new EventPublisher backed by Kafka.
func NewKafkaPublisher(pub messaging.Publisher) domain.EventPublisher {
	return &kafkaPublisher{
		publisher: pub,
	}
}

// EnterpriseCreatedEvent is the payload for the enterprise.created event.
type EnterpriseCreatedEvent struct {
	EnterpriseID uuid.UUID `json:"enterprise_id"`
	Name         string    `json:"name"`
	OwnerEmail   string    `json:"owner_email"`
	Timestamp    int64     `json:"timestamp"`
}

// PublishEnterpriseCreated publishes an enterprise.created event.
func (p *kafkaPublisher) PublishEnterpriseCreated(ctx context.Context, enterpriseID uuid.UUID, legalName string, ownerEmail string) error {
	event := EnterpriseCreatedEvent{
		EnterpriseID: enterpriseID,
		Name:         legalName,
		OwnerEmail:   ownerEmail,
		Timestamp:    time.Now().Unix(),
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("kafka_publisher: marshal event: %w", err)
	}

	msg := messaging.Message{
		Topic: topics.EnterpriseCreated,
		Key:   enterpriseID[:],
		Value: payload,
	}

	if err := p.publisher.Publish(ctx, msg); err != nil {
		return fmt.Errorf("kafka_publisher: publish: %w", err)
	}

	return nil
}
