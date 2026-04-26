package config

import (
	"fmt"
	"os"
)

type Config struct {
	Port   string
	DBUser string
	DBPass string
	DBHost string
	DBPort string
	DBName string
	DSN                 string
	KafkaBrokers        []string
	EnterpriseServiceURL string
	CandidateServiceURL  string
}

func Load() *Config {
	cfg := &Config{
		Port:   getEnv("GO_PORT", "8080"),
		DBUser: getEnv("PG_VERITAS_USER", "postgres"),
		DBPass: getEnv("PG_VERITAS_PASSWORD", "postgres"),
		DBHost: getEnv("PG_VERITAS_HOST", "localhost"),
		DBPort: getEnv("PG_VERITAS_PORT", "5432"),
		DBName: getEnv("PG_VERITAS_CORE_DB", "veritas_core"),
	}

	cfg.DSN = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.DBUser, cfg.DBPass, cfg.DBHost, cfg.DBPort, cfg.DBName)

	cfg.KafkaBrokers = []string{getEnv("KAFKA_BROKERS", "localhost:9092")}
	cfg.EnterpriseServiceURL = getEnv("ENTERPRISE_SERVICE_URL", "http://localhost:8081")
	cfg.CandidateServiceURL = getEnv("CANDIDATE_SERVICE_URL", "http://localhost:8082")

	return cfg
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}
