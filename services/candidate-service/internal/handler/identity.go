package handler

import (


	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/services/candidate-service/internal/domain"
)

// getEnterpriseID extracts the enterprise ID from the Gin context or "X-Enterprise-Id" header.
func getEnterpriseID(c *gin.Context) (uuid.UUID, error) {
	val, exists := c.Get("enterprise_id")
	if !exists {
		headerVal := c.GetHeader("X-Enterprise-Id")
		if headerVal != "" {
			return uuid.Parse(headerVal)
		}
		return uuid.Nil, domain.ErrUnauthorizedAccess
	}
	return uuid.Parse(val.(string))
}

// getCandidateID extracts the candidate ID from the Gin context or "X-Subject-Id" header.
func getCandidateID(c *gin.Context) (uuid.UUID, error) {
	val, exists := c.Get("subject_id")
	if !exists {
		headerVal := c.GetHeader("X-Subject-Id")
		if headerVal != "" {
			return uuid.Parse(headerVal)
		}
		return uuid.Nil, domain.ErrUnauthorizedAccess
	}
	return uuid.Parse(val.(string))
}

// getEnrollmentID extracts the enrollment ID from the Gin context or "X-Enrollment-Id" header.
func getEnrollmentID(c *gin.Context) (uuid.UUID, error) {
	val, exists := c.Get("enrollment_id")
	if !exists {
		headerVal := c.GetHeader("X-Enrollment-Id")
		if headerVal != "" {
			return uuid.Parse(headerVal)
		}
		return uuid.Nil, domain.ErrUnauthorizedAccess
	}
	return uuid.Parse(val.(string))
}