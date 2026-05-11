package config

import (
	"fmt"
	"os"
)

type Config struct {
	Port                   string
	DBUser                 string
	DBPass                 string
	DBHost                 string
	DBPort                 string
	DBName                 string
	DSN                    string
	ExamServiceURL         string
	EnrollmentTokenSecret  string
	CandidatePortalBaseURL string
	KafkaBrokers           string

	// Cloudinary configuration
	CloudinaryCloudName        string
	CloudinaryAPIKey           string
	CloudinaryAPISecret        string
	CloudinaryFaceUploadFolder string
}

func Load() *Config {
	cfg := &Config{
		Port:                   getEnv("GO_PORT", "8080"),
		DBUser:                 getEnv("PG_VERITAS_USER", "postgres"),
		DBPass:                 getEnv("PG_VERITAS_PASSWORD", "postgres"),
		DBHost:                 getEnv("PG_VERITAS_HOST", "localhost"),
		DBPort:                 getEnv("PG_VERITAS_PORT", "5432"),
		DBName:                 getEnv("PG_VERITAS_CORE_DB", "veritas_core"),
		ExamServiceURL:         getEnv("EXAM_SERVICE_URL", "http://localhost:8084"),
		EnrollmentTokenSecret:  getEnv("ENROLLMENT_TOKEN_SECRET", "super-secret-enrollment-key"),
		CandidatePortalBaseURL: getEnv("FRONTEND_BASE_URL", "http://localhost:3000"),
		KafkaBrokers:           getEnv("KAFKA_BROKERS", "localhost:9092"),

		CloudinaryCloudName:        getEnv("CLOUDINARY_CLOUD_NAME", ""),
		CloudinaryAPIKey:           getEnv("CLOUDINARY_API_KEY", ""),
		CloudinaryAPISecret:        getEnv("CLOUDINARY_API_SECRET", ""),
		CloudinaryFaceUploadFolder: getEnv("CLOUDINARY_FACE_UPLOADFOLDER", "veritas/face_registrations"),
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
