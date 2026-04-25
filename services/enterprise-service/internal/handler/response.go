package handler

import (
	"github.com/gin-gonic/gin"
)

type errorResponse struct {
	Error string `json:"error"`
}

type MessageResponse struct {
	Message string `json:"message"`
}

func writeError(c *gin.Context, code int, message string) {
	c.JSON(code, errorResponse{Error: message})
}

func writeJSON(c *gin.Context, code int, data any) {
	c.JSON(code, data)
}
