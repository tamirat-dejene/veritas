// Package main is the entry-point for the Veritas candidate-service.
//
//	@title			Veritas Candidate Service API
//	@version		1.0
//	@description	Candidate lifecycle, enrollment, exam access, and session management service.
//
//	@contact.name	Veritas Platform Team
//
//	@tag.name		candidate
//	@tag.description	Candidate, enrollment, access, session, and submission endpoints.
//	@tag.name		system
//	@tag.description	Operational and health endpoints.
//
//	@schemes		http https
//	@BasePath	/

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tamirat-dejene/veritas/services/candidate-service/internal/config"
	c_http "github.com/tamirat-dejene/veritas/services/candidate-service/internal/handler/http"
	"github.com/tamirat-dejene/veritas/services/candidate-service/internal/infrastructure/client"
	"github.com/tamirat-dejene/veritas/services/candidate-service/internal/repository/postgres"
	"github.com/tamirat-dejene/veritas/services/candidate-service/internal/router"
	"github.com/tamirat-dejene/veritas/services/candidate-service/internal/usecase"
	pg "github.com/tamirat-dejene/veritas/shared/db/pg"
	"github.com/tamirat-dejene/veritas/shared/pkg/logger"
	"go.uber.org/zap"

	// Import generated swagger docs so the spec is registered at startup.
	_ "github.com/tamirat-dejene/veritas/services/candidate-service/docs/swagger"
)

func main() {
	// 1. Initialize Logger from Shared Library matching the original implementation
	log, err := logger.NewLogger()
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = log.Sync()
	}()
	zap.ReplaceGlobals(log)

	// 2. Load Configuration
	cfg := config.Load()

	// 3. Initialize Shared Postgres DB Client
	dbClient, err := pg.NewPostgresClient(cfg.DSN)
	if err != nil {
		log.Fatal("failed to connect to database", zap.Error(err))
	}
	defer dbClient.Close()
	dbClient.LogConnectionInfo()

	// 4. Initialize Infrastructure Clients (HTTP Client for Exam Service)
	examClient := client.NewExamServiceClient(cfg.ExamServiceURL)

	// 5. Initialize Repositories
	candidateRepo := postgres.NewCandidateRepository(dbClient)
	enrollmentRepo := postgres.NewEnrollmentRepository(dbClient)
	sessionRepo := postgres.NewSessionRepository(dbClient)

	// 6. Initialize UseCases
	candidateUC := usecase.NewCandidateUseCase(candidateRepo, log)
	enrollmentUC := usecase.NewEnrollmentUseCase(enrollmentRepo, log)
	sessionUC := usecase.NewSessionUseCase(sessionRepo, enrollmentRepo, examClient, log)
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
		fmt.Printf("Candidate Service starting on port %s...\n", cfg.Port)
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
