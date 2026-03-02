package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
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
	engine.GET("/health", healthCheck)

	// Auth routes
	auth := engine.Group("/auth")
	{
		auth.POST("/login", authHandler.Login)
		auth.POST("/refresh", authHandler.Refresh)
		auth.POST("/logout", authHandler.Logout)
	}

	// Swagger UI — available at /swagger/index.html
	engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	return engine
}

// healthCheck returns 200 OK if the service is running.
//
//	@Summary		Health check
//	@ID			healthCheck
//	@Description	Returns a simple JSON indicating the service is alive.
//	@Tags			system
//	@Produce		json
//	@Success		200	{object}	map[string]string	"Service is healthy"
//	@Header			200	{string}	X-Request-ID		"Request correlation ID"
//	@Router			/health [get]
func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
