package router

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:embed docs
var docsFS embed.FS

// RegisterDocs sets up the static documentation route
func (g *RouterGroup) RegisterDocs() error {
	docsSub, err := fs.Sub(docsFS, "docs")
	if err != nil {
		return err
	}

	// Create file server for embedded static assets
	fileServer := http.FileServer(http.FS(docsSub))

	// Serving /docs/*filepath
	// We use http.StripPrefix("/docs/") because gin.WrapH receives the full URL path
	g.engine.GET("/docs/*filepath", gin.WrapH(http.StripPrefix("/docs/", fileServer)))

	// Redirect /docs (no slash) to /docs/ so it hits the /*filepath pattern correctly
	g.engine.GET("/docs", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/docs/")
	})

	// Redirect root to /docs/
	g.engine.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/docs/")
	})

	return nil
}
