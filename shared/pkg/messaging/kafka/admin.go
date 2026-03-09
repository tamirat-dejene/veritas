package kafka

import (
	"context"
	"fmt"

	"github.com/IBM/sarama"
	"github.com/tamirat-dejene/veritas/shared/pkg/messaging"
)

type adminClient struct {
	admin sarama.ClusterAdmin
}

// NewAdminClient creates an AdminClient for managing topics and partitions.
func NewAdminClient(cfg Config) (messaging.AdminClient, error) {
	sc := saramaConfig(cfg)
	a, err := sarama.NewClusterAdmin(cfg.Brokers, sc)
	if err != nil {
		return nil, fmt.Errorf("kafka: create admin client: %w", err)
	}
	return &adminClient{admin: a}, nil
}

// CreateTopic creates a topic with the given configuration.
// If the topic already exists the call is a no-op (idempotent).
func (a *adminClient) CreateTopic(_ context.Context, topic string, cfg messaging.TopicConfig) error {
	detail := &sarama.TopicDetail{
		NumPartitions:     cfg.NumPartitions,
		ReplicationFactor: cfg.ReplicationFactor,
	}
	if cfg.RetentionMs != 0 {
		retentionStr := fmt.Sprintf("%d", cfg.RetentionMs)
		detail.ConfigEntries = map[string]*string{
			"retention.ms": &retentionStr,
		}
	}

	err := a.admin.CreateTopic(topic, detail, false)
	if err != nil {
		// ErrTopicAlreadyExists is benign — treat as success so services can
		// safely call CreateTopic at startup without extra existence checks.
		if te, ok := err.(*sarama.TopicError); ok && te.Err == sarama.ErrTopicAlreadyExists {
			return nil
		}
		return fmt.Errorf("kafka: create topic %q: %w", topic, err)
	}
	return nil
}

// DeleteTopic removes a topic from the cluster.
func (a *adminClient) DeleteTopic(_ context.Context, topic string) error {
	if err := a.admin.DeleteTopic(topic); err != nil {
		return fmt.Errorf("kafka: delete topic %q: %w", topic, err)
	}
	return nil
}

// ListTopics returns the names of all topics visible to the client.
func (a *adminClient) ListTopics(_ context.Context) ([]string, error) {
	topics, err := a.admin.ListTopics()
	if err != nil {
		return nil, fmt.Errorf("kafka: list topics: %w", err)
	}
	names := make([]string, 0, len(topics))
	for name := range topics {
		names = append(names, name)
	}
	return names, nil
}

// Close releases the admin client's broker connections.
func (a *adminClient) Close() error {
	return a.admin.Close()
}
