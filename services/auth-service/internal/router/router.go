package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tamirat-dejene/veritas/services/auth-service/internal/handler"
	"github.com/tamirat-dejene/veritas/services/auth-service/internal/middleware"
	"go.uber.org/zap"
)

// NewRouter creates and configures the Gin engine with all routes and middleware.
func NewRouter(authHandler *handler.AuthHandler, log *zap.Logger) *gin.Engine {
	// Use production-mode Gin (no debug output to stdout).
	engine := gin.New()

	// Global middleware
	engine.Use(
		gin.Recovery(),         // recover from panics and return 500
		middleware.RequestID(), // attach X-Request-ID to every request
	)

	// Health check (no auth required).
	engine.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Auth routes
	auth := engine.Group("/auth")
	{
		auth.POST("/login", authHandler.Login)
		auth.POST("/refresh", authHandler.Refresh)
		auth.POST("/logout", authHandler.Logout)
	}

	return engine
}
