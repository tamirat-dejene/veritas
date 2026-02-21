package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func CORS(allowedOrigins, allowedMethods, allowedHeaders []string) gin.HandlerFunc {
	allowAllOrigins := len(allowedOrigins) == 1 && allowedOrigins[0] == "*"
	allowedOriginSet := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		if origin == "" || origin == "*" {
			continue
		}
		allowedOriginSet[origin] = struct{}{}
	}

	allowMethods := strings.Join(allowedMethods, ", ")
	allowHeaders := strings.Join(allowedHeaders, ", ")

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" {
			if allowAllOrigins {
				c.Header("Access-Control-Allow-Origin", "*")
			} else if _, ok := allowedOriginSet[origin]; ok {
				c.Header("Access-Control-Allow-Origin", origin)
			}
			c.Header("Vary", "Origin")
		}

		if c.Request.Method == http.MethodOptions && c.GetHeader("Access-Control-Request-Method") != "" {
			if allowMethods != "" {
				c.Header("Access-Control-Allow-Methods", allowMethods)
			}
			if allowHeaders != "" {
				c.Header("Access-Control-Allow-Headers", allowHeaders)
			}
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
