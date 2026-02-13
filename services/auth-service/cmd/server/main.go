package main

import (
	"database/sql"
	"log"
	"net/http"

	_ "github.com/lib/pq"
	"github.com/tamirat-dejene/veritas/services/auth-service/internal/application"
	"github.com/tamirat-dejene/veritas/services/auth-service/internal/config"
	"github.com/tamirat-dejene/veritas/services/auth-service/internal/infrastructure/postgres"
	"github.com/tamirat-dejene/veritas/services/auth-service/internal/infrastructure/security"
	authHTTP "github.com/tamirat-dejene/veritas/services/auth-service/internal/interface/http"
)

func main() {
	cfg := config.Load()

	// 1. Initialize Database
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("failed to ping db: %v", err)
	}
	log.Println("Connected to database")

	// 2. Initialize Infrastructure
	userRepo := postgres.NewUserRepository(db)
	tokenService := security.NewTokenService(cfg.JWTSecret)

	// 3. Initialize Application
	authService := application.NewAuthService(userRepo, tokenService)

	// 4. Initialize Interface
	handler := authHTTP.NewRouter(authService)

	// 5. Start Server
	log.Printf("Starting auth-service on port %s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, handler); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
