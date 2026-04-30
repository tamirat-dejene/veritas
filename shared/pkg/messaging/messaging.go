package messaging

import "context"

// Message is the canonical message envelope passed through the system.
type Message struct {
	Topic   string
	Key     []byte            // optional partition key
	Value   []byte            // serialized payload (JSON, protobuf, etc.)
	Headers map[string][]byte // metadata propagated alongside the payload
}

// Publisher publishes messages to a broker topic.
type Publisher interface {
	Publish(ctx context.Context, msg Message) error
	PublishBatch(ctx context.Context, msgs []Message) error
	Close() error
}

// Handler is the callback invoked for every received message.
// Returning a non-nil error signals that the message could not be processed;
// the broker-specific implementation decides whether to retry or dead-letter.
type Handler func(ctx context.Context, msg Message) error

// Subscriber subscribes to one or more topics and dispatches each
// received message to the provided Handler.
// Subscribe blocks until ctx is cancelled.
type Subscriber interface {
	Subscribe(ctx context.Context, topics []string, handler Handler) error
	Close() error
}

// AdminClient manages broker-level resources such as topics and partitions.
type AdminClient interface {
	CreateTopic(ctx context.Context, topic string, cfg TopicConfig) error
	DeleteTopic(ctx context.Context, topic string) error
	ListTopics(ctx context.Context) ([]string, error)
	Close() error
}

// TopicConfig holds parameters used when creating a topic.
type TopicConfig struct {
	NumPartitions     int32
	ReplicationFactor int16
	// RetentionMs configures log.retention.ms (-1 = infinite, 0 = broker default).
	RetentionMs int64
}
