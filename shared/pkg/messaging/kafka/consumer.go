package kafka

import (
	"context"
	"fmt"

	"github.com/IBM/sarama"
	"github.com/tamirat-dejene/veritas/shared/pkg/messaging"
)

type subscriber struct {
	cg sarama.ConsumerGroup
}

// NewSubscriber creates a new Subscriber backed by a Kafka ConsumerGroup.
// cfg.ConsumerGroup must be non-empty.
func NewSubscriber(cfg Config) (messaging.Subscriber, error) {
	if cfg.ConsumerGroup == "" {
		return nil, fmt.Errorf("kafka: ConsumerGroup must not be empty")
	}
	sc := saramaConfig(cfg)
	cg, err := sarama.NewConsumerGroup(cfg.Brokers, cfg.ConsumerGroup, sc)
	if err != nil {
		return nil, fmt.Errorf("kafka: create consumer group %q: %w", cfg.ConsumerGroup, err)
	}
	return &subscriber{cg: cg}, nil
}

// Subscribe blocks until ctx is cancelled, dispatching each received message
// to handler. When handler returns nil the message offset is committed.
// When handler returns a non-nil error the offset is NOT committed, causing
// the message to be redelivered after the next rebalance.
func (s *subscriber) Subscribe(ctx context.Context, topics []string, handler messaging.Handler) error {
	h := &cgHandler{handler: handler}
	for {
		if err := s.cg.Consume(ctx, topics, h); err != nil {
			return fmt.Errorf("kafka: consume error: %w", err)
		}
		if ctx.Err() != nil {
			return nil // clean shutdown requested by caller
		}
	}
}

// Close shuts down the consumer group connection.
func (s *subscriber) Close() error {
	return s.cg.Close()
}

// ── sarama.ConsumerGroupHandler implementation ────────────────────────────────

// cgHandler bridges sarama's ConsumerGroupHandler interface to our Handler.
type cgHandler struct {
	handler messaging.Handler
}

// Setup is called at the beginning of a new session, before ConsumeClaim.
func (h *cgHandler) Setup(_ sarama.ConsumerGroupSession) error { return nil }

// Cleanup is called at the end of a session once all ConsumeClaim goroutines
// have exited.
func (h *cgHandler) Cleanup(_ sarama.ConsumerGroupSession) error { return nil }

// ConsumeClaim processes messages from a single partition claim.
// It runs in its own goroutine per partition.
func (h *cgHandler) ConsumeClaim(
	sess sarama.ConsumerGroupSession,
	claim sarama.ConsumerGroupClaim,
) error {
	for m := range claim.Messages() {
		headers := make(map[string][]byte, len(m.Headers))
		for _, hdr := range m.Headers {
			headers[string(hdr.Key)] = hdr.Value
		}

		msg := messaging.Message{
			Topic:   m.Topic,
			Key:     m.Key,
			Value:   m.Value,
			Headers: headers,
		}

		if err := h.handler(sess.Context(), msg); err == nil {
			// Only mark a message as processed when the handler succeeds.
			sess.MarkMessage(m, "")
		}
		// On error: offset not marked → message redelivered after rebalance.
	}
	return nil
}
