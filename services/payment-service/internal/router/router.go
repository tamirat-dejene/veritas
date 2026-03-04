package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/tamirat-dejene/veritas/services/payment-service/internal/handler"
)

func NewRouter(h *handler.PaymentHandler) *gin.Engine {
	r := gin.Default()

	r.GET("/health", healthCheck)

	api := r.Group("/")
	{
		api.GET("/subscriptions/plans", h.ListPlans)
		api.POST("/subscriptions/:enterpriseId/upgrade", h.UpgradeSubscription)
		api.GET("/payments/history", h.ListPaymentHistory)
		api.GET("/invoices/:invoiceId", h.GetInvoice)
		api.POST("/webhooks/stripe", h.HandleWebhook)
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
