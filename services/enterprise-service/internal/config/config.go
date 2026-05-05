package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	Port              string
	DBUser            string
	DBPass            string
	DBHost            string
	DBPort            string
	DBName            string
	DSN               string
	KafkaBrokers      []string
	PaymentServiceURL   string
	ExamServiceURL      string
	CandidateServiceURL string
	FrontendBaseURL     string
	CloudinaryCloudName string
	CloudinaryAPIKey    string
	CloudinaryAPISecret string
	CloudinaryLogosFolder    string
}


func Load() *Config {
	cfg := &Config{
		Port:         getEnv("GO_PORT", "8080"),
		DBUser:       getEnv("PG_VERITAS_USER", "postgres"),
		DBPass:       getEnv("PG_VERITAS_PASSWORD", "postgres"),
		DBHost:       getEnv("PG_VERITAS_HOST", "localhost"),
		DBPort:       getEnv("PG_VERITAS_PORT", "5432"),
		DBName:       getEnv("PG_VERITAS_CORE_DB", "veritas_core"),
		KafkaBrokers:      getEnvList("KAFKA_BROKERS", ","),
		PaymentServiceURL: getEnv("PAYMENT_SERVICE_URL", "http://payment-service:8080"),
		ExamServiceURL:      getEnv("EXAM_SERVICE_URL", "http://exam-service:8080"),
		CandidateServiceURL: getEnv("CANDIDATE_SERVICE_URL", "http://candidate-service:8080"),
		FrontendBaseURL:   getEnv("FRONTEND_BASE_URL", "https://app.veritas.io"),
		CloudinaryCloudName: getEnv("CLOUDINARY_CLOUD_NAME", ""),
		CloudinaryAPIKey:    getEnv("CLOUDINARY_API_KEY", ""),
		CloudinaryAPISecret: getEnv("CLOUDINARY_API_SECRET", ""),
		CloudinaryLogosFolder:    getEnv("CLOUDINARY_LOGOS_FOLDER", "veritas/logos"),
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

func getEnvList(key, delimiter string) []string {
	value := os.Getenv(key)
	if value == "" {
		return nil
	}
	return strings.Split(value, delimiter)
}
