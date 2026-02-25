package router

import (
	"github.com/gin-gonic/gin"
	"github.com/tamirat-dejene/veritas/services/payment-service/internal/handler"
)

func NewRouter(h *handler.PaymentHandler) *gin.Engine {
	r := gin.Default()

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	api := r.Group("/")
	{
		api.GET("/subscriptions/plans", h.ListPlans)
		api.POST("/subscriptions/:enterpriseId/upgrade", h.UpgradeSubscription)
		api.GET("/payments/history", h.ListPaymentHistory)
		api.GET("/invoices/:invoiceId", h.GetInvoice)
		api.POST("/webhooks/stripe", h.HandleWebhook)
	}

	return r
}
