package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tamirat-dejene/veritas/services/exam-service/internal/config"
	"github.com/tamirat-dejene/veritas/services/exam-service/internal/handler"
	"github.com/tamirat-dejene/veritas/services/exam-service/internal/repository/postgres"
	"github.com/tamirat-dejene/veritas/services/exam-service/internal/router"
	"github.com/tamirat-dejene/veritas/services/exam-service/internal/usecase"
	pg "github.com/tamirat-dejene/veritas/shared/db/pg"
	"go.uber.org/zap"
)

func main() {
	// 1. Initialize Logger
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	zap.ReplaceGlobals(logger)

	// 2. Load Configuration
	cfg := config.Load()

	// 3. Initialize Database Client
	dbClient, err := pg.NewPostgresClient(cfg.DSN)
	if err != nil {
		logger.Fatal("failed to connect to database", zap.Error(err))
	}
	defer dbClient.Close()
	dbClient.LogConnectionInfo()

	// 4. Initialize Repositories
	questionRepo := postgres.NewQuestionRepository(dbClient)
	examRepo := postgres.NewExamRepository(dbClient)

	// 5. Initialize Usecases
	questionUC := usecase.NewQuestionUsecase(questionRepo)
	examUC := usecase.NewExamUsecase(examRepo, questionRepo)

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
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exiting")
}
