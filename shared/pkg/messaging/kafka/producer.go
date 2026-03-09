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
// The SyncProducer blocks until the broker acknowledges delivery according
// to the RequireAcks setting in Config (default: WaitForAll).
func NewProducer(cfg Config) (messaging.Publisher, error) {
	sc := saramaConfig(cfg)
	sp, err := sarama.NewSyncProducer(cfg.Brokers, sc)
	if err != nil {
		return nil, fmt.Errorf("kafka: create producer: %w", err)
	}
	return &producer{sp: sp}, nil
}

// Publish sends a message to the topic specified in msg.Topic.
// It is safe to call from multiple goroutines concurrently.
func (p *producer) Publish(_ context.Context, msg messaging.Message) error {
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

	_, _, err := p.sp.SendMessage(pm)
	if err != nil {
		return fmt.Errorf("kafka: publish to %s: %w", msg.Topic, err)
	}
	return nil
}

// Close flushes and closes the underlying SyncProducer.
func (p *producer) Close() error {
	return p.sp.Close()
}
