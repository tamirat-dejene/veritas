package router

import (
	"net/http"
	"net/url"
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
		// Capture loop variables explicitly (safe in Go 1.22+ but good practice)
		localBase := basePath
		localProxy := serviceProxy

		// /swagger/{service}
		g.engine.GET(localBase, func(c *gin.Context) {
			c.Redirect(http.StatusFound, localBase+"/index.html")
		})

		// /swagger/{service}/*filepath
		g.engine.GET(localBase+"/*filepath", func(c *gin.Context) {
			filePath := strings.TrimPrefix(c.Param("filepath"), "/")
			if filePath == "" {
				filePath = "index.html"
			}

			req := c.Request.Clone(c.Request.Context())
			req.URL = &url.URL{}
			*req.URL = *c.Request.URL
			req.URL.Path = "/swagger/" + filePath

			localProxy.ServeHTTP(c.Writer, req)
		})
	}
}

// redactDatabaseURL strips the user:password@ portion from a Postgres DSN to prevent
// credential exposure in the health endpoint.
func redactDatabaseURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "[invalid-url]"
	}
	if u.User != nil {
		u.User = url.User(u.User.Username()) // keep username, drop password
	}
	return u.String()
}
