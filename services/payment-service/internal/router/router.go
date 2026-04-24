package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/tamirat-dejene/veritas/services/payment-service/internal/handler"
	smw "github.com/tamirat-dejene/veritas/shared/pkg/middleware"
)

func NewRouter(h *handler.PaymentHandler) *gin.Engine {
	r := gin.New()
	r.Use(
		smw.Recovery(),
		smw.RequestID(),
		smw.Logging(),
	)

	r.GET("/health", healthCheck)

	api := r.Group("/")
	{
		api.GET("/subscriptions/plans", h.ListPlans)
		api.GET("/subscriptions/:enterpriseId", h.GetActiveSubscription)
		api.POST("/subscriptions/:enterpriseId/upgrade", h.UpgradeSubscription)
		api.POST("/subscriptions/:enterpriseId/cancel", h.CancelSubscription)
		api.POST("/subscriptions/:enterpriseId/reactivate", h.ReactivateSubscription)
		api.GET("/payments/history", h.ListPaymentHistory)
		api.GET("/invoices", h.ListInvoices)
		api.GET("/invoices/:invoiceId", h.GetInvoice)
		api.GET("/billing/summary", h.GetBillingSummary)
		api.POST("/webhooks/stripe", h.HandleWebhook)
	}

	// Admin-only routes
	admin := r.Group("/admin")
	{
		admin.POST("/subscriptions/:enterpriseId", h.AdminSetSubscription)
		admin.POST("/subscriptions/:enterpriseId/trial", h.CreateTrialSubscription)
		admin.POST("/plans", h.CreatePlan)
		admin.GET("/plans", h.AdminListPlans)
		admin.PATCH("/plans/:planId", h.UpdatePlan)
		admin.DELETE("/plans/:planId", h.DeactivatePlan)
		admin.POST("/invoices/:invoiceId/refund", h.RefundInvoice)
	}

	// Internal routes
	internal := r.Group("/billing")
	{
		internal.GET("/usage/:enterpriseId", h.GetFeatureGate)
	}

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	return r
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
	c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "payment-service"})
}
