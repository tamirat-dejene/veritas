package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tamirat-dejene/veritas/services/api-gateway/internal/domain"
	"github.com/tamirat-dejene/veritas/services/api-gateway/internal/middleware"
)

// RouterGroup encapsulates routing logic and common middleware builders
type RouterGroup struct {
	engine    *gin.Engine
	jwtSecret string
}

// NewRouterGroup creates a new router group
func NewRouterGroup(engine *gin.Engine, jwtSecret string) *RouterGroup {
	return &RouterGroup{
		engine:    engine,
		jwtSecret: jwtSecret,
	}
}

// register handles attaching HTTP handlers wrapped as Gin routes
// Because proxies are standard http.Handler, we wrap them to gin.HandlerFunc.
func (g *RouterGroup) register(method, path string, h http.Handler, mws ...gin.HandlerFunc) {
	handlers := make([]gin.HandlerFunc, len(mws)+1)
	copy(handlers, mws)
	handlers[len(mws)] = gin.WrapH(h)
	g.engine.Handle(method, path, handlers...)
}

// defaultAuthChain builds the required base authentication and tenant resolution middlewares
func (g *RouterGroup) defaultAuthChain() []gin.HandlerFunc {
	return []gin.HandlerFunc{
		middleware.JWTAuth(g.jwtSecret),
		middleware.TenantResolver(),
		middleware.InjectUserHeaders(), // forward X-User-ID, X-User-Role, X-Enterprise-ID to downstream
	}
}

// authWithRoles wraps routes with auth and domain role checks
func (g *RouterGroup) authWithRoles(roles ...domain.Role) []gin.HandlerFunc {
	return append(g.defaultAuthChain(), middleware.RequireRole(roles...))
}
