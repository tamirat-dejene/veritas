package config

import (
	"fmt"
	"os"
	"time"
)

// Config holds all runtime configuration for the auth-service.
type Config struct {
	Port            string
	JWTSecret       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration

	// PostgreSQL configuration
	Pg_Veritas_Host      string
	Pg_Veritas_Port      string
	Pg_Veritas_User      string
	Pg_Veritas_Password  string
	Pg_Veritas_Core_DB   string
	Pg_SSL_Mode          string
	DSN                  string
	KafkaBrokers         []string
	EnterpriseServiceURL string
}

// Load reads configuration from environment variables and returns a Config.
// All values have sensible, secure defaults for local development only.
func Load() *Config {
	cfg := &Config{
		Port:            getEnv("GO_PORT", "8080"),
		JWTSecret:       getEnv("JWT_SECRET", ""),
		AccessTokenTTL:  getDurationEnv("ACCESS_TOKEN_TTL", 15*time.Minute),
		RefreshTokenTTL: getDurationEnv("REFRESH_TOKEN_TTL", 7*24*time.Hour),
		EnterpriseServiceURL: getEnv("ENTERPRISE_SERVICE_URL", "http://enterprise-service:8080"),
		KafkaBrokers: []string{getEnv("KAFKA_BROKERS", "kafka:9092")},

		Pg_Veritas_Host:     getEnv("PG_VERITAS_HOST", "localhost"),
		Pg_Veritas_Port:     getEnv("PG_VERITAS_PORT", "5432"),
		Pg_Veritas_User:     getEnv("PG_VERITAS_USER", "postgres"),
		Pg_Veritas_Password: getEnv("PG_VERITAS_PASSWORD", ""),
		Pg_Veritas_Core_DB:  getEnv("POSTGRES_AUTH_DB", getEnv("PG_VERITAS_CORE_DB", "veritas_auth_db")),
		Pg_SSL_Mode:         getEnv("PG_SSL_MODE", "disable"),
	}
	cfg.DSN = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", cfg.Pg_Veritas_User, cfg.Pg_Veritas_Password, cfg.Pg_Veritas_Host, cfg.Pg_Veritas_Port, cfg.Pg_Veritas_Core_DB, cfg.Pg_SSL_Mode)
	return cfg
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}

// getDurationEnv parses a Go duration string (e.g. "15m", "168h").
func getDurationEnv(key string, fallback time.Duration) time.Duration {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return fallback
}
