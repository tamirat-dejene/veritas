package handler

import (
	"github.com/gin-gonic/gin"
)

func writeError(c *gin.Context, code int, message string) {
	c.JSON(code, ErrorResponse{Error: message})
}

func writeJSON(c *gin.Context, code int, data any) {
	c.JSON(code, data)
}
