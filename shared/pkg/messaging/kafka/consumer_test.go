package kafka

import (
	"context"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/tamirat-dejene/veritas/shared/pkg/messaging"
)

// mockConsumerGroup implements sarama.ConsumerGroup for testing.
type mockConsumerGroup struct {
	sarama.ConsumerGroup
	closed bool
}

func (m *mockConsumerGroup) Consume(ctx context.Context, topics []string, handler sarama.ConsumerGroupHandler) error {
	// Simulate receiving one message
	sess := &mockSession{ctx: ctx}
	claim := &mockClaim{
		messages: make(chan *sarama.ConsumerMessage, 1),
	}
	claim.messages <- &sarama.ConsumerMessage{
		Topic: "test-topic",
		Value: []byte("test-consumer-value"),
	}
	close(claim.messages)

	err := handler.Setup(sess)
	if err != nil {
		return err
	}

	err = handler.ConsumeClaim(sess, claim)
	if err != nil {
		return err
	}

	handler.Cleanup(sess)

	// Block until context is cancelled to avoid tight loop in Subscribe
	<-ctx.Done()
	return nil
}

func (m *mockConsumerGroup) Close() error {
	m.closed = true
	return nil
}

type mockSession struct {
	sarama.ConsumerGroupSession
	ctx context.Context
}

func (m *mockSession) Context() context.Context { return m.ctx }
func (m *mockSession) MarkMessage(msg *sarama.ConsumerMessage, metadata string) {}

type mockClaim struct {
	sarama.ConsumerGroupClaim
	messages chan *sarama.ConsumerMessage
}

func (m *mockClaim) Messages() <-chan *sarama.ConsumerMessage { return m.messages }

func TestSubscriber_Subscribe(t *testing.T) {
	mockCG := &mockConsumerGroup{}
	s := &subscriber{
		cg: mockCG,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	received := make(chan string, 1)
	handler := func(ctx context.Context, msg messaging.Message) error {
		received <- string(msg.Value)
		return nil
	}

	// We expect Subscribe to run until the mock claim finishes or context cancels
	err := s.Subscribe(ctx, []string{"test-topic"}, handler)
	if err != nil && err != context.DeadlineExceeded {
		t.Fatalf("Subscribe failed: %v", err)
	}

	select {
	case val := <-received:
		if val != "test-consumer-value" {
			t.Errorf("expected test-consumer-value, got %s", val)
		}
	case <-time.After(500 * time.Millisecond):
		t.Error("timeout waiting for message")
	}
}
