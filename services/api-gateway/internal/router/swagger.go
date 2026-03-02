package router

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// RegisterSwaggerRoutes exposes downstream service Swagger UIs via the API gateway.
// Pattern:
//
//	/swagger/{service}             -> redirects to /swagger/{service}/index.html
//	/swagger/{service}/*filepath   -> proxies to {service}/swagger/*filepath
func (g *RouterGroup) RegisterSwaggerRoutes(serviceSwaggerProxies map[string]http.Handler) {
	for serviceName, serviceProxy := range serviceSwaggerProxies {
		if serviceName == "" || serviceProxy == nil {
			continue
		}

		basePath := "/swagger/" + serviceName

		// /swagger/{service}
		g.engine.GET(basePath, func(c *gin.Context) {
			c.Redirect(http.StatusFound, basePath+"/index.html")
		})

		// /swagger/{service}/*filepath
		g.engine.GET(basePath+"/*filepath", func(c *gin.Context) {
			filePath := strings.TrimPrefix(c.Param("filepath"), "/")
			if filePath == "" {
				filePath = "index.html"
			}

			// Rewrite to downstream swagger path.
			c.Request.URL.Path = "/swagger/" + filePath
			serviceProxy.ServeHTTP(c.Writer, c.Request)
		})
	}
}
