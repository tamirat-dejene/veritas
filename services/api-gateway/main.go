package main

import (
	"log"
	"net/http"

	"github.com/tamirat-dejene/veritas/services/api-gateway/internal/config"
	"github.com/tamirat-dejene/veritas/services/api-gateway/internal/router"
)

func main() {
	cfg := config.Load()

	// Initialize Router
	handler, err := router.NewRouter(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize router: %v", err)
	}

	log.Printf("Service api-gateway starting on port %s", cfg.Port)
	log.Printf("Routes configured for services: Auth=%s, Enterprise=%s, Payment=%s", cfg.AuthServiceURL, cfg.EnterpriseServiceURL, cfg.PaymentServiceURL)

	// Start Server
	if err := http.ListenAndServe(":"+cfg.Port, handler); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
