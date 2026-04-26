package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/tamirat-dejene/veritas/services/exam-service/internal/domain"
	"github.com/tamirat-dejene/veritas/shared/pkg/messaging"
	"github.com/tamirat-dejene/veritas/shared/pkg/messaging/topics"
)

type kafkaPublisher struct {
	pub messaging.Publisher
}

func NewKafkaPublisher(pub messaging.Publisher) domain.EventPublisher {
	return &kafkaPublisher{pub: pub}
}

func (p *kafkaPublisher) PublishExamCreated(ctx context.Context, event domain.ExamCreatedEvent) error {
	if event.Timestamp == 0 {
		event.Timestamp = time.Now().Unix()
	}
	return p.publish(ctx, topics.ExamCreated, event.ExamID.String(), event)
}

func (p *kafkaPublisher) PublishExamScheduled(ctx context.Context, event domain.ExamLifecycleEvent) error {
	if event.Timestamp == 0 {
		event.Timestamp = time.Now().Unix()
	}
	return p.publish(ctx, topics.ExamScheduled, event.ExamID.String(), event)
}

func (p *kafkaPublisher) PublishExamPublished(ctx context.Context, event domain.ExamLifecycleEvent) error {
	if event.Timestamp == 0 {
		event.Timestamp = time.Now().Unix()
	}
	return p.publish(ctx, topics.ExamPublished, event.ExamID.String(), event)
}

func (p *kafkaPublisher) PublishExamClosed(ctx context.Context, event domain.ExamLifecycleEvent) error {
	if event.Timestamp == 0 {
		event.Timestamp = time.Now().Unix()
	}
	return p.publish(ctx, topics.ExamClosed, event.ExamID.String(), event)
}

func (p *kafkaPublisher) publish(ctx context.Context, topic, key string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("kafka_publisher: marshal: %w", err)
	}

	msg := messaging.Message{
		Topic: topic,
		Key:   []byte(key),
		Value: data,
	}

	return p.pub.Publish(ctx, msg)
}
