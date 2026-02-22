package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tamirat-dejene/veritas/services/enterprise-service/internal/handler"
)

func NewRouter(h *handler.EnterpriseHandler) *gin.Engine {
	engine := gin.New()

	engine.Use(gin.Recovery())
	engine.Use(gin.Logger())

	engine.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	enterprises := engine.Group("/enterprises")
	{
		enterprises.POST("", h.Register)
		enterprises.GET("/:enterpriseId", h.Get)
		enterprises.PATCH("/:enterpriseId", h.Update)
		enterprises.POST("/:enterpriseId/approve", h.Approve)
		enterprises.POST("/:enterpriseId/suspend", h.Suspend)
		enterprises.DELETE("/:enterpriseId", h.Delete)
	}

	return engine
}
