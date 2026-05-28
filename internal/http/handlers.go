// Package http is the HTTP transport for the league API. Handlers are thin
// — they parse requests, call the service, and write DTOs. Everything
// HTTP-shaped lives here; everything domain-shaped lives behind the
// LeagueService boundary.
package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/busrakoyun/premier-league-simulation/internal/service"
)

// Handler holds the dependencies every endpoint needs. Constructed once at
// startup and shared across requests — the service is responsible for any
// per-request concurrency safety.
type Handler struct {
	svc *service.LeagueService
}

// NewHandler wires a Handler around the league service.
func NewHandler(svc *service.LeagueService) *Handler {
	return &Handler{svc: svc}
}

// Healthz returns 200 with a tiny JSON payload — used by Render's health
// probe and by curl-while-debugging.
func (h *Handler) Healthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// GET /api/seasons/current
func (h *Handler) GetCurrentSeason(w http.ResponseWriter, r *http.Request) {
	season, err := h.svc.GetCurrentSeason(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toSeasonDTO(season))
}

// GET /api/seasons/current/teams
func (h *Handler) GetTeams(w http.ResponseWriter, r *http.Request) {
	teams, err := h.svc.GetTeams(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, TeamsResponse{Teams: toTeamDTOs(teams)})
}

// GET /api/seasons/current/standings
func (h *Handler) GetStandings(w http.ResponseWriter, r *http.Request) {
	rows, err := h.svc.GetStandings(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, StandingsResponse{Rows: toStandingDTOs(rows)})
}

// GET /api/seasons/current/matches
func (h *Handler) GetMatches(w http.ResponseWriter, r *http.Request) {
	matches, err := h.svc.GetMatches(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	teams, err := h.svc.GetTeams(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, MatchesResponse{Matches: toMatchDTOs(matches, teams)})
}

// GET /api/seasons/current/predictions
func (h *Handler) GetPredictions(w http.ResponseWriter, r *http.Request) {
	rows, err := h.svc.Predict(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, PredictionsResponse{Rows: toPredictionDTOs(rows)})
}

// POST /api/seasons/current/next-week
func (h *Handler) PostNextWeek(w http.ResponseWriter, r *http.Request) {
	week, err := h.svc.NextWeek(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	teams, err := h.svc.GetTeams(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, NextWeekResponse{
		Week:    week.Week,
		Matches: toMatchDTOs(week.Matches, teams),
	})
}

// POST /api/seasons/current/play-all
func (h *Handler) PostPlayAll(w http.ResponseWriter, r *http.Request) {
	weeks, err := h.svc.PlayAll(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	teams, err := h.svc.GetTeams(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, PlayAllResponse{Weeks: toWeekDTOs(weeks, teams)})
}

// POST /api/seasons/current/reset
func (h *Handler) PostReset(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.ResetSeason(r.Context()); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// PATCH /api/matches/{id} — edit a played match's score.
//
// Validation order matters: we check the URL param and JSON body shape
// before calling the service so handler-level validation errors get the
// VALIDATION code, while service-level rules (unplayed, negative score)
// get their domain-specific codes downstream.
func (h *Handler) PatchMatch(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeErrorEnvelope(w, http.StatusBadRequest, CodeValidation, "match id must be an integer")
		return
	}
	var req EditMatchRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeErrorEnvelope(w, http.StatusBadRequest, CodeValidation, "invalid JSON body: "+err.Error())
		return
	}

	updated, err := h.svc.EditMatch(r.Context(), id, req.HomeGoals, req.AwayGoals)
	if err != nil {
		writeError(w, err)
		return
	}
	teams, err := h.svc.GetTeams(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toMatchDTO(updated, indexTeams(teams)))
}
