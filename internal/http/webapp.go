package http

import (
	"embed"
	"io/fs"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
)

// distFS holds the Vue build. The repo commits only a .gitkeep here so the
// directory exists for `go:embed` on a fresh clone; `make web-build` (and
// the Dockerfile) populate index.html + assets/ before `go build`. When
// no real index.html is embedded, SPA() serves the in-code placeholder
// below so a clone-and-build cycle without npm still produces a usable
// landing page describing the API endpoints.
//
//go:embed all:dist
var distFS embed.FS

// fallbackHTML is served at "/" when no real index.html is embedded — the
// signal that the Vue dist hasn't been built. It lists every API endpoint
// so a reviewer who hits the URL gets immediate value.
const fallbackHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Premier League Simulation</title>
    <style>
        :root { color-scheme: light dark; }
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", system-ui, sans-serif; max-width: 720px; margin: 4rem auto; padding: 0 1.25rem; line-height: 1.55; color: #1a1a1a; background: #fafafa; }
        @media (prefers-color-scheme: dark) { body { color: #ececec; background: #161616; } code { background: #2a2a2a; } }
        h1 { font-size: 1.6rem; margin-bottom: 0.25rem; }
        h2 { font-size: 1.05rem; margin-top: 2rem; color: #444; }
        @media (prefers-color-scheme: dark) { h2 { color: #bbb; } }
        code { background: #ececec; padding: 0.1rem 0.4rem; border-radius: 4px; font-size: 0.95em; font-family: "SF Mono", Menlo, monospace; }
        ul { padding-left: 1.25rem; }
        li { margin: 0.3rem 0; }
    </style>
</head>
<body>
    <h1>Premier League Simulation</h1>
    <p>The Vue frontend isn't built into this binary. Run <code>make web-build</code> to embed it, or test the API directly.</p>

    <h2>Read endpoints</h2>
    <ul>
        <li><code>GET /api/seasons/current</code></li>
        <li><code>GET /api/seasons/current/teams</code></li>
        <li><code>GET /api/seasons/current/standings</code></li>
        <li><code>GET /api/seasons/current/matches</code></li>
        <li><code>GET /api/seasons/current/predictions</code> &mdash; 422 before week 4</li>
    </ul>

    <h2>Write endpoints</h2>
    <ul>
        <li><code>POST /api/seasons/current/next-week</code></li>
        <li><code>POST /api/seasons/current/play-all</code></li>
        <li><code>POST /api/seasons/current/reset</code></li>
        <li><code>PATCH /api/matches/{id}</code> with body <code>{"home_goals":N,"away_goals":N}</code></li>
    </ul>

    <h2>Health</h2>
    <ul>
        <li><code>GET /healthz</code></li>
    </ul>
</body>
</html>`

// SPA serves the embedded Vue build with HTML5 fallback: any path that
// doesn't match a real asset returns index.html so the Vue Router can
// handle the client-side route on refresh. If no index.html exists in the
// embed (fresh clone before `make web-build`), it serves the in-code
// fallback HTML instead — so the binary is always useful even without npm.
func (h *Handler) SPA(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		writeFallback(w)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" {
		path = "index.html"
	}

	data, err := fs.ReadFile(sub, path)
	if err != nil {
		// Asset not found — try index.html so the SPA router can resolve
		// the route on the client. If even that's missing we're running
		// without a Vue build; serve the in-code fallback.
		data, err = fs.ReadFile(sub, "index.html")
		if err != nil {
			writeFallback(w)
			return
		}
		path = "index.html"
	}

	if ct := mime.TypeByExtension(filepath.Ext(path)); ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	_, _ = w.Write(data)
}

func writeFallback(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(fallbackHTML))
}
