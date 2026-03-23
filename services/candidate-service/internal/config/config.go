package config

import (
	"fmt"
	"os"
)

type Config struct {
	Port           string
	DBUser         string
	DBPass         string
	DBHost         string
	DBPort         string
	DBName         string
	DSN            string
	ExamServiceURL        string
	EnrollmentTokenSecret string
}

func Load() *Config {
	cfg := &Config{
		Port:           getEnv("GO_PORT", "8080"),
		DBUser:         getEnv("PG_VERITAS_USER", "postgres"),
		DBPass:         getEnv("PG_VERITAS_PASSWORD", "postgres"),
		DBHost:         getEnv("PG_VERITAS_HOST", "localhost"),
		DBPort:         getEnv("PG_VERITAS_PORT", "5432"),
		DBName:         getEnv("PG_VERITAS_CORE_DB", "veritas_core"),
		ExamServiceURL:        getEnv("EXAM_SERVICE_URL", "http://localhost:8084"),
		EnrollmentTokenSecret: getEnv("ENROLLMENT_TOKEN_SECRET", "super-secret-enrollment-key"),
	}

	cfg.DSN = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.DBUser, cfg.DBPass, cfg.DBHost, cfg.DBPort, cfg.DBName)

	return cfg
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}
