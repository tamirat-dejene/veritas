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
	CloudinaryCloudName   string
	CloudinaryAPIKey      string
	CloudinaryAPISecret   string
	CloudinaryQuestionsFolder string
}

func Load() *Config {
	cfg := &Config{
		Port:   getEnv("GO_PORT", "8080"),
		DBUser: getEnv("PG_VERITAS_USER", "postgres"),
		DBPass: getEnv("PG_VERITAS_PASSWORD", "postgres"),
		DBHost: getEnv("PG_VERITAS_HOST", "localhost"),
		DBPort: getEnv("PG_VERITAS_PORT", "5432"),
		DBName: getEnv("POSTGRES_EXAM_DB", getEnv("PG_VERITAS_CORE_DB", "veritas_exam_db")),
	}

	cfg.DSN = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.DBUser, cfg.DBPass, cfg.DBHost, cfg.DBPort, cfg.DBName)

	cfg.KafkaBrokers = []string{getEnv("KAFKA_BROKERS", "localhost:9092")}
	cfg.EnterpriseServiceURL = getEnv("ENTERPRISE_SERVICE_URL", "http://localhost:8081")
	cfg.CandidateServiceURL = getEnv("CANDIDATE_SERVICE_URL", "http://localhost:8082")

	cfg.CloudinaryCloudName = getEnv("CLOUDINARY_CLOUD_NAME", "")
	cfg.CloudinaryAPIKey = getEnv("CLOUDINARY_API_KEY", "")
	cfg.CloudinaryAPISecret = getEnv("CLOUDINARY_API_SECRET", "")
	cfg.CloudinaryQuestionsFolder = getEnv("CLOUDINARY_QUESTIONS_FOLDER", "veritas/questions")

	return cfg
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}
