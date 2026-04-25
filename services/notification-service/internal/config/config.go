package config

import (
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

// Config represents the notification-service configuration.
type Config struct {
	// General Configuration
	SystemMode string `envconfig:"SYSTEM_MODE" default:"development"`

	// Messaging (Kafka)
	KafkaBrokers []string `envconfig:"KAFKA_BROKERS" required:"true"`

	// SMTP Configuration
	SMTPHost string `envconfig:"SMTP_HOST" required:"true"`
	SMTPPort int    `envconfig:"SMTP_PORT" required:"true"`
	SMTPUser string `envconfig:"SMTP_USER" required:"false"`
	SMTPPass string `envconfig:"SMTP_PASS" required:"false"`
	SMTPFrom string `envconfig:"SMTP_FROM" required:"true"`
}

// Load loads the configuration from environment variables.
func Load() *Config {
	// Attempt to load .env file if it exists
	if _, err := os.Stat(".env"); err == nil {
		if err := godotenv.Load(); err != nil {
			log.Printf("Warning: Error loading .env file: %v", err)
		}
	} else if os.IsNotExist(err) && os.Getenv("SYSTEM_MODE") != "production" {
		// Fallback to parent directory
		if err := godotenv.Load("../../.env"); err != nil {
			log.Printf("Warning: Error loading ../../.env file: %v", err)
		}
	}

	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		log.Fatalf("Failed to parse configuration: %v", err)
	}

	// Clean up Kafka brokers list
	for i, broker := range cfg.KafkaBrokers {
		cfg.KafkaBrokers[i] = strings.TrimSpace(broker)
	}

	return &cfg
}
