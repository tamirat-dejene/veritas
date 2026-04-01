package handler

import (

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GetCallerID extracts the authenticated user's ID from the X-User-ID header.
func GetCallerID(c *gin.Context) (uuid.UUID, bool) {
	raw := c.GetHeader("X-User-ID")
	if raw == "" {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(raw)
	return id, err == nil
}

// GetCallerRole extracts the authenticated user's role from X-User-Role header.
func GetCallerRole(c *gin.Context) string {
	return c.GetHeader("X-User-Role")
}

// GetCallerEnterpriseID extracts enterprise ID from X-Enterprise-ID header.
func GetCallerEnterpriseID(c *gin.Context) (uuid.UUID, bool) {
	raw := c.GetHeader("X-Enterprise-ID")
	if raw == "" {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(raw)
	return id, err == nil
}

// ParseEnterpriseID parses :enterpriseId from the URL path.
func ParseEnterpriseID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("enterpriseId"))
	return id, err == nil
}

// ParseUserID parses :userId from the URL path.
func ParseUserID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("userId"))
	return id, err == nil
}
