package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/services/auth-service/internal/domain"
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

// UserLoginEvent is the payload for the auth.login event.
type UserLoginEvent struct {
	UserID    uuid.UUID `json:"user_id"`
	Email     string    `json:"email"`
	Timestamp int64     `json:"timestamp"`
}

func (p *kafkaPublisher) PublishLogin(ctx context.Context, userID uuid.UUID, email string) error {
	event := UserLoginEvent{
		UserID:    userID,
		Email:     email,
		Timestamp: time.Now().Unix(),
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("kafka_publisher: marshal event: %w", err)
	}

	msg := messaging.Message{
		Topic: topics.AuthUserLogin,
		Key:   userID[:],
		Value: payload,
	}

	if err := p.publisher.Publish(ctx, msg); err != nil {
		return fmt.Errorf("kafka_publisher: publish: %w", err)
	}

	return nil
}
