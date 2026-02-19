package main

import (
	"database/sql"
	"net/http"

	_ "github.com/lib/pq"
	"github.com/tamirat-dejene/veritas/services/auth-service/internal/application"
	"github.com/tamirat-dejene/veritas/services/auth-service/internal/config"
	"github.com/tamirat-dejene/veritas/services/auth-service/internal/infrastructure/postgres"
	"github.com/tamirat-dejene/veritas/services/auth-service/internal/infrastructure/security"
	authHTTP "github.com/tamirat-dejene/veritas/services/auth-service/internal/interface/http"
	"github.com/tamirat-dejene/veritas/shared/pkg/logger"
	"go.uber.org/zap"
)

func main() {
	log, err := logger.NewLogger()
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = log.Sync()
	}()
	zap.ReplaceGlobals(log)

	cfg := config.Load()

	// 1. Initialize Database
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		zap.L().Fatal("failed to open db", zap.Error(err))
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		zap.L().Fatal("failed to ping db", zap.Error(err))
	}
	zap.L().Info("Connected to database")

	// 2. Initialize Infrastructure
	userRepo := postgres.NewUserRepository(db)
	tokenService := security.NewTokenService(cfg.JWTSecret)

	// 3. Initialize Application
	authService := application.NewAuthService(userRepo, tokenService)

	// 4. Initialize Interface
	handler := authHTTP.NewRouter(authService)

	// 5. Start Server
	zap.L().Info("Starting auth-service", zap.String("port", cfg.Port))
	if err := http.ListenAndServe(":"+cfg.Port, handler); err != nil {
		zap.L().Fatal("failed to start server", zap.Error(err))
	}
}
