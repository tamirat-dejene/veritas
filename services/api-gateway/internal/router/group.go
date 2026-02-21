package router

import (
	"net/http"

	"github.com/tamirat-dejene/veritas/services/api-gateway/internal/domain"
	"github.com/tamirat-dejene/veritas/services/api-gateway/internal/middleware"
)

// RouterGroup encapsulates routing logic and common middleware builders
type RouterGroup struct {
	mux       *http.ServeMux
	jwtSecret string
}

// NewRouterGroup creates a new router group
func NewRouterGroup(mux *http.ServeMux, jwtSecret string) *RouterGroup {
	return &RouterGroup{
		mux:       mux,
		jwtSecret: jwtSecret,
	}
}

// chain merges a handler and middlewares into a single handler
func (g *RouterGroup) chain(h http.Handler, mws ...func(http.Handler) http.Handler) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

// register handles attaching routes to the mux
func (g *RouterGroup) register(pattern string, h http.Handler, mws ...func(http.Handler) http.Handler) {
	g.mux.Handle(pattern, g.chain(h, mws...))
}

// defaultAuthChain builds the required base authentication and tenant resolution middlewares
func (g *RouterGroup) defaultAuthChain() []func(http.Handler) http.Handler {
	return []func(http.Handler) http.Handler{
		middleware.JWTAuth(g.jwtSecret),
		middleware.TenantResolver,
	}
}

// authWithRoles wraps routes with auth and domain role checks
func (g *RouterGroup) authWithRoles(roles ...domain.Role) []func(http.Handler) http.Handler {
	return append(g.defaultAuthChain(), middleware.RequireRole(roles...))
}
