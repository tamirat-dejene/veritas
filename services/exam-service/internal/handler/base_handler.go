package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func writeError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": message})
}

func writeJSON(c *gin.Context, status int, data interface{}) {
	c.JSON(status, data)
}

// Helper to extract EnterpriseID and UserID from context
// In a real scenario, these come from middleware set by the Gateway
func getEnterpriseID(c *gin.Context) (uuid.UUID, bool) {
	idStr := c.GetHeader("X-Enterprise-ID")
	if idStr == "" {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(idStr)
	return id, err == nil
}

func getUserID(c *gin.Context) (uuid.UUID, bool) {
	idStr := c.GetHeader("X-User-ID")
	if idStr == "" {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(idStr)
	return id, err == nil
}
