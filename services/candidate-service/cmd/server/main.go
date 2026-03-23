// Package main is the entry-point for the Veritas candidate-service.
//
//	@title			Veritas Candidate Service API
//	@version		1.0
//	@description	Candidate lifecycle, enrollment, exam access, and session management service.
//
//	@contact.name	Veritas Platform Team
//
//	@tag.name		candidate
//	@tag.description	Candidate profile management endpoints.
//	@tag.name		enrollment
//	@tag.description	Exam enrollment and enrollment token management endpoints.
//	@tag.name		session
//	@tag.description	Session lifecycle, access validation, answers, and submission endpoints.
//	@tag.name		monitoring
//	@tag.description	Session monitoring and submission/result retrieval endpoints.
//	@tag.name		system
//	@tag.description	Operational and health endpoints.
//
//	@schemes		http https
//	@BasePath	/api/v1

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tamirat-dejene/veritas/services/candidate-service/internal/config"
	c_http "github.com/tamirat-dejene/veritas/services/candidate-service/internal/handler"
	"github.com/tamirat-dejene/veritas/services/candidate-service/internal/infrastructure/client"
	"github.com/tamirat-dejene/veritas/services/candidate-service/internal/infrastructure/token"
	"github.com/tamirat-dejene/veritas/services/candidate-service/internal/repository/postgres"
	"github.com/tamirat-dejene/veritas/services/candidate-service/internal/router"
	"github.com/tamirat-dejene/veritas/services/candidate-service/internal/usecase"
	"github.com/tamirat-dejene/veritas/shared/pkg/logger"
	"go.uber.org/zap"

	// Import generated swagger docs so the spec is registered at startup.
	_ "github.com/tamirat-dejene/veritas/services/candidate-service/docs/swagger"
)

func main() {
	// 1. Initialize Logger from Shared Library matching the original implementation
	log, err := logger.NewLogger("candidate-service")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		_ = log.Sync()
	}()

	// 2. Load Configuration
	cfg := config.Load()

	// 3. Initialize pgxpool directly
	pool, err := pgxpool.New(context.Background(), cfg.DSN)
	if err != nil {
		log.Fatal("failed to connect to database", zap.Error(err))
	}
	defer pool.Close()
	log.Info("connected to postgres (via pgxpool)")

	// 4. Initialize Infrastructure Clients (HTTP Client for Exam Service)
	examClient := client.NewExamServiceClient(cfg.ExamServiceURL)

	// 5. Initialize Repositories (passing pool as DBTX)
	candidateRepo := postgres.NewCandidateRepository(pool)
	enrollmentRepo := postgres.NewEnrollmentRepository(pool)
	sessionRepo := postgres.NewSessionRepository(pool)

	// 6. Initialize UseCases
	tokenService := token.NewTokenService(cfg.EnrollmentTokenSecret)

	candidateUC := usecase.NewCandidateUseCase(pool, candidateRepo, log)
	enrollmentUC := usecase.NewEnrollmentUseCase(pool, enrollmentRepo, tokenService, log)
	sessionUC := usecase.NewSessionUseCase(pool, sessionRepo, enrollmentRepo, examClient, tokenService, log)
	monitoringUC := usecase.NewMonitoringUseCase(sessionRepo, log)

	// 7. Initialize Handlers
	candidateHandler := c_http.NewCandidateHandler(candidateUC, log)
	enrollmentHandler := c_http.NewEnrollmentHandler(enrollmentUC, log)
	sessionHandler := c_http.NewSessionHandler(sessionUC, log)
	monitoringHandler := c_http.NewMonitoringHandler(monitoringUC, log)

	// 8. Initialize Router
	r := router.NewRouter(candidateHandler, enrollmentHandler, sessionHandler, monitoringHandler)

	// 9. Start Server
	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	go func() {
		log.Info("Candidate Service starting", zap.String("port", cfg.Port))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("listen error", zap.Error(err))
		}
	}()

	// Graceful Shutdown block
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown", zap.Error(err))
	}

	log.Info("Server exiting")
}
