package config

import (
	"os"
)

type Config struct {
	Port                       string
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
}

func Load() *Config {
	return &Config{
		Port:                       getEnv("PORT", "8080"),
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
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
