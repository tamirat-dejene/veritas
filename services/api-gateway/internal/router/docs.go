package router

import (
	"embed"
	"io/fs"
	"net/http"
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
	g.mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/docs", http.StatusFound)
	})

	// Redirect /docs to /docs/
	g.register("GET /docs", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/docs/", http.StatusFound)
	}))

	// Serve the static files
	g.register("GET /docs/", docsHandler)

	return nil
}
