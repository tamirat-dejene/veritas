package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/tamirat-dejene/veritas/services/enterprise-service/internal/handler"
	smw "github.com/tamirat-dejene/veritas/shared/pkg/middleware"
)

// NewRouter creates and configures the Gin engine.
func NewRouter(
	eh *handler.EnterpriseHandler,
	uh *handler.UserHandler,
) *gin.Engine {
	engine := gin.New()
	engine.Use(
		smw.Recovery(),
		smw.RequestID(),
		smw.Logging(),
	)

	engine.GET("/health", healthCheck)

	// ── Public Auth Routes (no authentication required) ───────────────────
	engine.POST("/auth/forgot-password", uh.ForgotPassword)
	engine.POST("/auth/reset-password", uh.ResetPasswordViaToken)

	enterprises := engine.Group("/enterprises")
	{
		// ── Registration (public) ─────────────────────────────────────────────
		enterprises.POST("", eh.Register)

		// ── Discovery ─────────────────────────────────────────────────────────
		enterprises.GET("", eh.List)
		enterprises.GET("/me", eh.GetMe)
		enterprises.GET("/slug/:slug", eh.GetBySlug)

		// ── Single-enterprise read & general update ───────────────────────────
		enterprises.GET("/:enterpriseId", eh.Get)
		enterprises.PATCH("/:enterpriseId", eh.Update)

		// ── Admin lifecycle ───────────────────────────────────────────────────
		enterprises.POST("/:enterpriseId/approve", eh.Approve)
		enterprises.POST("/:enterpriseId/suspend", eh.Suspend)
		enterprises.DELETE("/:enterpriseId", eh.Delete)
		enterprises.POST("/:enterpriseId/reactivate", eh.Reactivate)
		enterprises.POST("/:enterpriseId/restore", eh.Restore)
		enterprises.DELETE("/:enterpriseId/permanent", eh.HardDelete)

		// ── Self-Service Branding & Settings ──────────────────────────────────
		enterprises.PATCH("/:enterpriseId/branding", eh.UpdateBranding)
		enterprises.PATCH("/:enterpriseId/settings", eh.UpdateSettings)

		// ── Status, Domain, Audit ─────────────────────────────────────────────
		enterprises.GET("/:enterpriseId/status", eh.GetStatus)
		enterprises.POST("/:enterpriseId/validate-domain", eh.ValidateDomain)
		enterprises.GET("/:enterpriseId/summary", eh.GetSummary)
		enterprises.GET("/:enterpriseId/audit-logs", eh.GetAuditLogs)

		// ── Enterprise User Management ────────────────────────────────────────
		enterprises.POST("/:enterpriseId/users", uh.CreateUser)
		enterprises.GET("/:enterpriseId/users", uh.ListUsers)
		enterprises.GET("/:enterpriseId/users/:userId", uh.GetUser)
		enterprises.PATCH("/:enterpriseId/users/:userId", uh.UpdateUser)
		enterprises.PATCH("/:enterpriseId/users/:userId/deactivate", uh.DeactivateUser)
		enterprises.PATCH("/:enterpriseId/users/:userId/activate", uh.ActivateUser)
		enterprises.DELETE("/:enterpriseId/users/:userId", uh.DeleteUser)
		enterprises.POST("/:enterpriseId/users/:userId/reset-password", uh.ResetPassword)
		enterprises.POST("/:enterpriseId/users/:userId/change-password", uh.ChangePassword)
	}

	// ── Internal Auth Lookups (Service-to-Service) ────────────────────────
	internal := engine.Group("/internal")
	{
		users := internal.Group("/users")
		{
			users.GET("", uh.GetByEmail) // GET /internal/users?email=...
			users.GET("/:id", uh.GetByID)
			users.POST("/:id/login-success", uh.RecordLoginSuccess)
			users.POST("/:id/login-failure", uh.RecordLoginFailure)
		}
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
//	@Router			/health [get]
func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "enterprise-service"})
}
