// Package main is the entry-point for the Veritas exam-service.
//
//	@title			Veritas Exam Service API
//	@version		1.0
//	@description	Exam authoring, scheduling, publishing, and question bank management service.
//
//	@contact.name	Veritas Platform Team
//
//	@tag.name		exam
//	@tag.description	Exam lifecycle and assembly endpoints.
//	@tag.name		question
//	@tag.description	Question bank management endpoints.
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

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tamirat-dejene/veritas/services/exam-service/internal/config"
	"github.com/tamirat-dejene/veritas/services/exam-service/internal/handler"
	"github.com/tamirat-dejene/veritas/services/exam-service/internal/repository/postgres"
	"github.com/tamirat-dejene/veritas/services/exam-service/internal/router"
	"github.com/tamirat-dejene/veritas/services/exam-service/internal/usecase"
	"github.com/tamirat-dejene/veritas/shared/pkg/logger"
	"go.uber.org/zap"

	// Import generated swagger docs so the spec is registered at startup.
	_ "github.com/tamirat-dejene/veritas/services/exam-service/docs/swagger"
)

func main() {
	// 1. Initialize Logger
	log, err := logger.NewLogger("exam-service")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer log.Sync()

	// 2. Load Configuration
	cfg := config.Load()

	// 3. Initialize pgxpool directly
	pool, err := pgxpool.New(context.Background(), cfg.DSN)
	if err != nil {
		log.Fatal("failed to connect to database", zap.Error(err))
	}
	defer pool.Close()
	log.Info("connected to postgres (via pgxpool)")

	// 4. Initialize Repositories (passing pool as DBTX)
	questionRepo := postgres.NewQuestionRepository(pool)
	examRepo := postgres.NewExamRepository(pool)

	// 5. Initialize Usecases
	questionUC := usecase.NewQuestionUsecase(pool, questionRepo)
	examUC := usecase.NewExamUsecase(pool, examRepo, questionRepo)

	// 6. Initialize Handlers
	questionHandler := handler.NewQuestionHandler(questionUC)
	examHandler := handler.NewExamHandler(examUC)

	// 7. Initialize Router
	r := router.NewRouter(questionHandler, examHandler)

	// 8. Start HTTP Server
	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	go func() {
		fmt.Printf("Exam Service starting on port %s...\n", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("listen error", zap.Error(err))
		}
	}()

	// Graceful Shutdown
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
