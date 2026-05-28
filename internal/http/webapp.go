package http

import (
	"embed"
	"io/fs"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
)

// distFS holds the Vue build. In development before `make web-build`
// runs, this directory contains only the placeholder index.html committed
// to the repo. The Dockerfile copies web/dist/* over the placeholder
// before `go build`, so the deployed binary serves the real SPA.
//
//go:embed all:dist
var distFS embed.FS

// SPA serves the embedded Vue build with HTML5 fallback: any path that
// doesn't match a real asset returns index.html so the Vue Router can
// handle the client-side route on refresh.
func (h *Handler) SPA(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		http.Error(w, "frontend not embedded", http.StatusInternalServerError)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" {
		path = "index.html"
	}

	data, err := fs.ReadFile(sub, path)
	if err != nil {
		// Asset not found — fall back to index.html so the SPA router
		// can resolve the route on the client.
		data, err = fs.ReadFile(sub, "index.html")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		path = "index.html"
	}

	if ct := mime.TypeByExtension(filepath.Ext(path)); ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	_, _ = w.Write(data)
}
