package http

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Router builds the chi router with the API routes and the SPA fallback.
// Middlewares: request ID for log correlation, real IP behind Render's
// proxy, structured logging, panic recovery, a 60s ceiling on any single
// request, and permissive CORS for the Vite dev server during local dev
// (production serves the SPA from the same origin so CORS is a no-op).
func (h *Handler) Router() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(corsMiddleware)

	r.Get("/healthz", h.Healthz)

	r.Route("/api", func(r chi.Router) {
		r.Route("/seasons/current", func(r chi.Router) {
			r.Get("/", h.GetCurrentSeason)
			r.Get("/teams", h.GetTeams)
			r.Get("/standings", h.GetStandings)
			r.Get("/matches", h.GetMatches)
			r.Get("/predictions", h.GetPredictions)
			r.Post("/next-week", h.PostNextWeek)
			r.Post("/play-all", h.PostPlayAll)
			r.Post("/reset", h.PostReset)
		})
		r.Patch("/matches/{id}", h.PatchMatch)
	})

	// Anything else falls through to the SPA so client-side routes work
	// when the user refreshes the browser.
	r.NotFound(h.SPA)

	return r
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
