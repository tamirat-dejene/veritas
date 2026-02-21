package router

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:embed docs/index.html docs/styles.css docs/health.html docs/health.css
var docsFS embed.FS

// RegisterDocs sets up the static documentation route
func (g *RouterGroup) RegisterDocs() error {
	docsSub, err := fs.Sub(docsFS, "docs")
	if err != nil {
		return err
	}
	docsHandler := http.StripPrefix("/docs/", http.FileServer(http.FS(docsSub)))

	// Redirect homepage to docs
	g.engine.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/docs")
	})

	// Serve the static files using Gin wrapper
	// Setting up the /*filepath param helps gin pattern match over the http handler
	g.engine.GET("/docs/*filepath", gin.WrapH(http.StripPrefix("/docs", docsHandler)))

	// Ensure /docs without trailing slash redirects if hit explicitly, but gin usually handles this
	g.engine.GET("/docs", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/docs/")
	})

	return nil
}
