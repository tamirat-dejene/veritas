package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	SystemMode                 string
	Port                       string
	AuthServiceURL             string
	EnterpriseServiceURL       string
	PaymentServiceURL          string
	ExamServiceURL             string
	CandidateServiceURL        string
	ProctoringServiceURL       string
	GradingServiceURL          string
	JWTSecret                  string
	EnrollmentTokenSecret      string
	RedisHost                  string
	RedisPort                  int
	RedisPassword              string
	RedisDB                    int
	DatabaseURL                string
	KafkaBrokers               []string
	CORSAllowedOrigins         string
	CORSAllowedMethods         string
	CORSAllowedHeaders         string
}

const insecureDefaultSecret = "super-secret-key"

func Load() *Config {
	mode := getEnv("SYSTEM_MODE", "development")
	jwtSecret := getEnv("JWT_SECRET", insecureDefaultSecret)
	enrollmentSecret := getEnv("ENROLLMENT_TOKEN_SECRET", insecureDefaultSecret)

	if mode != "development" && (jwtSecret == "" || jwtSecret == insecureDefaultSecret) {
		panic("JWT_SECRET is not set or uses the insecure default — refusing to start in " + mode + " mode")
	}
	if mode != "development" && (enrollmentSecret == "" || enrollmentSecret == insecureDefaultSecret) {
		panic("ENROLLMENT_TOKEN_SECRET is not set or uses the insecure default — refusing to start in " + mode + " mode")
	}

	return &Config{
		SystemMode:                 mode,
		Port:                       getEnv("GO_PORT", "8080"),
		AuthServiceURL:             getEnv("AUTH_SERVICE_URL", "http://localhost:8081"),
		EnterpriseServiceURL:       getEnv("ENTERPRISE_SERVICE_URL", "http://localhost:8082"),
		PaymentServiceURL:          getEnv("PAYMENT_SERVICE_URL", "http://localhost:8083"),
		ExamServiceURL:             getEnv("EXAM_SERVICE_URL", "http://localhost:8084"),
		CandidateServiceURL:        getEnv("CANDIDATE_SERVICE_URL", "http://localhost:8085"),
		ProctoringServiceURL:       getEnv("PROCTORING_SERVICE_URL", "http://localhost:8086"),
		GradingServiceURL:          getEnv("GRADING_SERVICE_URL", "http://localhost:8088"),
		JWTSecret:                  jwtSecret,
		EnrollmentTokenSecret:      enrollmentSecret,
		RedisHost:                  getEnv("REDIS_HOST", "redis"),
		RedisPort:                  getEnvInt("REDIS_PORT", 6379),
		RedisPassword:              getEnv("REDIS_PASSWORD", ""),
		RedisDB:                    getEnvInt("REDIS_DB", 0),
		KafkaBrokers:               strings.Split(getEnv("KAFKA_BROKERS", "kafka:9092"), ","),
		CORSAllowedOrigins:         getEnv("CORS_ALLOWED_ORIGINS", "*"),
		CORSAllowedMethods:         getEnv("CORS_ALLOWED_METHODS", "GET,POST,PATCH,DELETE,OPTIONS"),
		CORSAllowedHeaders:         getEnv("CORS_ALLOWED_HEADERS", "Authorization,Content-Type,X-Request-ID"),
		DatabaseURL: fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
			getEnv("PG_VERITAS_USER", "postgres"),
			getEnv("PG_VERITAS_PASSWORD", "postgres"),
			getEnv("PG_VERITAS_HOST", "postgres"),
			getEnv("PG_VERITAS_PORT", "5432"),
			getEnv("POSTGRES_ENTERPRISE_DB", getEnv("PG_VERITAS_CORE_DB", "veritas_enterprise_db")),
		),
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
