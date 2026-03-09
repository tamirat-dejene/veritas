package kafka

import (
	"context"
	"testing"

	"github.com/IBM/sarama"
	"github.com/IBM/sarama/mocks"
	"github.com/tamirat-dejene/veritas/shared/pkg/messaging"
)

func TestProducer_Publish(t *testing.T) {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	
	mockSyncProducer := mocks.NewSyncProducer(t, config)
	
	p := &producer{
		sp: mockSyncProducer,
	}
	defer p.Close()

	ctx := context.Background()
	msg := messaging.Message{
		Topic: "test-topic",
		Key:   []byte("test-key"),
		Value: []byte("test-value"),
		Headers: map[string][]byte{
			"header1": []byte("value1"),
		},
	}

	mockSyncProducer.ExpectSendMessageWithCheckerFunctionAndSucceed(func(val []byte) error {
		if string(val) != "test-value" {
			t.Errorf("expected value test-value, got %s", string(val))
		}
		return nil
	})

	err := p.Publish(ctx, msg)
	if err != nil {
		t.Fatalf("failed to publish message: %v", err)
	}
}
