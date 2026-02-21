package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const requestIDHeader = "X-Request-ID"

// RequestID attaches a unique UUID to every incoming request.
// The ID is stored in the Gin context and echoed back in the response header.
// Downstream logs should include this ID using zap.String("requestId", ...).
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader(requestIDHeader)
		if requestID == "" {
			requestID = uuid.New().String()
		}

		c.Set("requestId", requestID)
		c.Header(requestIDHeader, requestID)
		c.Next()
	}
}
