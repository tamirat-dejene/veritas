package kafka

import (
	"crypto/tls"
	"time"

	"github.com/IBM/sarama"
)

// Config holds all settings for every Kafka component in this package.
// Zero values fall back to sensible production defaults.
type Config struct {
	// Brokers is the list of Kafka broker addresses (host:port).
	Brokers []string

	// TLS, when non-nil, enables TLS for all broker connections.
	TLS *tls.Config

	// --- Producer ---

	// ProducerMaxRetries is the number of retries on transient errors (default 3).
	ProducerMaxRetries int
	// ProducerTimeout is the per-message send timeout (default 10s).
	ProducerTimeout time.Duration
	// RequireAcks controls delivery guarantees:
	//   sarama.WaitForAll   – all in-sync replicas (safest, default)
	//   sarama.WaitForLocal – leader only (faster, less durable)
	//   sarama.NoResponse   – fire-and-forget
	RequireAcks sarama.RequiredAcks

	// --- Consumer ---

	// ConsumerGroup is the Kafka consumer-group name. Required for Subscriber.
	ConsumerGroup string
	// ConsumerOffsets controls where a new group starts consuming.
	// Accepted values: "oldest" (default) | "newest"
	ConsumerOffsets string
	// SessionTimeout is the consumer group session timeout (default 30s).
	SessionTimeout time.Duration
}

// saramaConfig converts a Config into a fully populated *sarama.Config.
// It is used internally by NewProducer, NewSubscriber, and NewAdminClient.
func saramaConfig(c Config) *sarama.Config {
	sc := sarama.NewConfig()
	sc.Version = sarama.V3_6_0_0 // minimum supported Kafka version

	// ── Producer ──────────────────────────────────────────────────────────────
	sc.Producer.Return.Successes = true
	sc.Producer.Return.Errors = true
	sc.Producer.Retry.Max = orInt(c.ProducerMaxRetries, 3)
	sc.Producer.Timeout = orDuration(c.ProducerTimeout, 10*time.Second)
	sc.Producer.RequiredAcks = orAcks(c.RequireAcks, sarama.WaitForAll)
	sc.Producer.Compression = sarama.CompressionSnappy // good balance of cpu/size

	// ── Consumer ──────────────────────────────────────────────────────────────
	sc.Consumer.Offsets.AutoCommit.Enable = true
	sc.Consumer.Offsets.AutoCommit.Interval = 3 * time.Second
	sc.Consumer.Group.Session.Timeout = orDuration(c.SessionTimeout, 30*time.Second)

	if c.ConsumerOffsets == "newest" {
		sc.Consumer.Offsets.Initial = sarama.OffsetNewest
	} else {
		sc.Consumer.Offsets.Initial = sarama.OffsetOldest
	}

	// ── TLS ───────────────────────────────────────────────────────────────────
	if c.TLS != nil {
		sc.Net.TLS.Enable = true
		sc.Net.TLS.Config = c.TLS
	}

	return sc
}

// ── helpers ───────────────────────────────────────────────────────────────────

func orInt(v, def int) int {
	if v == 0 {
		return def
	}
	return v
}

func orDuration(v, def time.Duration) time.Duration {
	if v == 0 {
		return def
	}
	return v
}

func orAcks(v, def sarama.RequiredAcks) sarama.RequiredAcks {
	if v == 0 {
		return def
	}
	return v
}
