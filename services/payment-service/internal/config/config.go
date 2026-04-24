package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	Port                string
	Pg_Veritas_Host     string
	Pg_Veritas_Port     string
	Pg_Veritas_User     string
	Pg_Veritas_Password string
	Pg_Veritas_Core_DB  string
	StripeSecretKey     string
	StripeWebhookSecret string
	StripeSuccessURL    string
	StripeCancelURL     string
	DSN                 string
	KafkaBrokers        []string
}

func Load() *Config {
	cfg := &Config{
		Port:                getEnv("GO_PORT", "8080"),
		Pg_Veritas_Host:     getEnv("PG_VERITAS_HOST", "localhost"),
		Pg_Veritas_Port:     getEnv("PG_VERITAS_PORT", "5432"),
		Pg_Veritas_User:     getEnv("PG_VERITAS_USER", "postgres"),
		Pg_Veritas_Password: getEnv("PG_VERITAS_PASSWORD", "postgres"),
		Pg_Veritas_Core_DB:  getEnv("PG_VERITAS_CORE_DB", "veritas_core"),
		StripeSecretKey:     getEnv("STRIPE_SECRET_KEY", ""),
		StripeWebhookSecret: getEnv("STRIPE_WEBHOOK_SECRET", ""),
		StripeSuccessURL:    getEnv("STRIPE_SUCCESS_URL", "https://veritas.com/payment/success?session_id={CHECKOUT_SESSION_ID}"),
		StripeCancelURL:     getEnv("STRIPE_CANCEL_URL", "https://veritas.com/payment/cancel"),
	}

	cfg.DSN = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.Pg_Veritas_User, cfg.Pg_Veritas_Password, cfg.Pg_Veritas_Host, cfg.Pg_Veritas_Port, cfg.Pg_Veritas_Core_DB)

	if brokers := os.Getenv("KAFKA_BROKERS"); brokers != "" {
		cfg.KafkaBrokers = strings.Split(brokers, ",")
	}

	return cfg
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
