package middleware

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/shared/pkg/logger"
)

const HeaderXRequestID = "X-Request-ID"

// RequestID returns a middleware that injects a request ID into the context and header.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.GetHeader(HeaderXRequestID)
		if rid == "" {
			rid = uuid.New().String()
		}

		// 1. Set in Gin context (for other Gin middlewares)
		c.Set(string(logger.RequestIDKey), rid)
		
		// 2. Wrap the request context for standard library / usecase usage
		ctx := context.WithValue(c.Request.Context(), logger.RequestIDKey, rid)
		c.Request = c.Request.WithContext(ctx)
		
		// 3. Set header in response
		c.Header(HeaderXRequestID, rid)
		
		c.Next()
	}
}
