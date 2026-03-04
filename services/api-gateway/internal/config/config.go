package config

import (
	"os"
	"strconv"
)

type Config struct {
	API_Gateway_Port           string
	AuthServiceURL             string
	EnterpriseServiceURL       string
	PaymentServiceURL          string
	ExamServiceURL             string
	CandidateServiceURL        string
	ProctoringServiceURL       string
	FaceVerificationServiceURL string
	GradingServiceURL          string
	ReportingServiceURL        string
	JWTSecret                  string
	RedisHost                  string
	RedisPort                  int
	RedisPassword              string
	RedisDB                    int
	DatabaseURL                string
	CORSAllowedOrigins         string
	CORSAllowedMethods         string
	CORSAllowedHeaders         string
}

func Load() *Config {
	return &Config{
		API_Gateway_Port:           getEnv("API_GATEWAY_PORT", "8080"),
		AuthServiceURL:             getEnv("AUTH_SERVICE_URL", "http://localhost:8081"),
		EnterpriseServiceURL:       getEnv("ENTERPRISE_SERVICE_URL", "http://localhost:8082"),
		PaymentServiceURL:          getEnv("PAYMENT_SERVICE_URL", "http://localhost:8083"),
		ExamServiceURL:             getEnv("EXAM_SERVICE_URL", "http://localhost:8084"),
		CandidateServiceURL:        getEnv("CANDIDATE_SERVICE_URL", "http://localhost:8085"),
		ProctoringServiceURL:       getEnv("PROCTORING_SERVICE_URL", "http://localhost:8086"),
		FaceVerificationServiceURL: getEnv("FACE_VERIFICATION_SERVICE_URL", "http://localhost:8087"),
		GradingServiceURL:          getEnv("GRADING_SERVICE_URL", "http://localhost:8088"),
		ReportingServiceURL:        getEnv("REPORTING_SERVICE_URL", "http://localhost:8089"),
		JWTSecret:                  getEnv("JWT_SECRET", "super-secret-key"),
		RedisHost:                  getEnv("REDIS_HOST", "redis"),
		RedisPort:                  getEnvInt("REDIS_PORT", 6379),
		RedisPassword:              getEnv("REDIS_PASSWORD", ""),
		RedisDB:                    getEnvInt("REDIS_DB", 0),
		CORSAllowedOrigins:         getEnv("CORS_ALLOWED_ORIGINS", "*"),
		CORSAllowedMethods:         getEnv("CORS_ALLOWED_METHODS", "GET,POST,PATCH,DELETE,OPTIONS"),
		CORSAllowedHeaders:         getEnv("CORS_ALLOWED_HEADERS", "Authorization,Content-Type,X-Request-ID"),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value, ok := os.LookupEnv(key); ok {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return fallback
}
