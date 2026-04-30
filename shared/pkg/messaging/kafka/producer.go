package kafka

import (
	"context"
	"fmt"

	"github.com/IBM/sarama"
	"github.com/tamirat-dejene/veritas/shared/pkg/messaging"
)

type producer struct {
	sp sarama.SyncProducer
}

// NewProducer creates a new Publisher backed by a Kafka SyncProducer.
func NewProducer(cfg Config) (messaging.Publisher, error) {
	sc := saramaConfig(cfg)
	sp, err := sarama.NewSyncProducer(cfg.Brokers, sc)
	if err != nil {
		return nil, fmt.Errorf("kafka: create producer: %w", err)
	}
	return &producer{sp: sp}, nil
}

// Publish sends a message to the topic specified
func (p *producer) Publish(_ context.Context, msg messaging.Message) error {
	pm := p.toSaramaMessage(msg)
	_, _, err := p.sp.SendMessage(pm)
	if err != nil {
		return fmt.Errorf("kafka: publish to %s: %w", msg.Topic, err)
	}
	return nil
}

// PublishBatch sends multiple messages in a single batch.
func (p *producer) PublishBatch(_ context.Context, msgs []messaging.Message) error {
	if len(msgs) == 0 {
		return nil
	}

	pms := make([]*sarama.ProducerMessage, len(msgs))
	for i, msg := range msgs {
		pms[i] = p.toSaramaMessage(msg)
	}

	err := p.sp.SendMessages(pms)
	if err != nil {
		return fmt.Errorf("kafka: publish batch: %w", err)
	}
	return nil
}

func (p *producer) toSaramaMessage(msg messaging.Message) *sarama.ProducerMessage {
	pm := &sarama.ProducerMessage{
		Topic: msg.Topic,
		Value: sarama.ByteEncoder(msg.Value),
	}

	if len(msg.Key) > 0 {
		pm.Key = sarama.ByteEncoder(msg.Key)
	}

	for k, v := range msg.Headers {
		pm.Headers = append(pm.Headers, sarama.RecordHeader{
			Key:   []byte(k),
			Value: v,
		})
	}
	return pm
}

// Close flushes and closes the underlying SyncProducer.
func (p *producer) Close() error {
	return p.sp.Close()
}
