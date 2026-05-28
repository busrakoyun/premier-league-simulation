package http

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/busrakoyun/premier-league-simulation/internal/repository"
	"github.com/busrakoyun/premier-league-simulation/internal/service"
)

// ErrorResponse is the JSON envelope returned for any non-2xx HTTP response.
// Deliberately flat ({error, code}) rather than RFC 7807 — the simplicity
// matches the PDF's testing-via-Postman bar and the frontend's needs.
type ErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

// Codes returned in the `code` field. Stable strings the frontend can switch on.
const (
	CodeValidation              = "VALIDATION"
	CodeNotFound                = "NOT_FOUND"
	CodeConflict                = "CONFLICT"
	CodeNoCurrentSeason         = "NO_CURRENT_SEASON"
	CodeSeasonComplete          = "SEASON_COMPLETE"
	CodePredictionsNotAvailable = "PREDICTIONS_NOT_YET_AVAILABLE"
	CodeMatchNotPlayed          = "MATCH_NOT_PLAYED"
	CodeInvalidScore            = "INVALID_SCORE"
	CodeInternal                = "INTERNAL"
)

// writeError maps a service/repo error onto an HTTP status + envelope.
// Unknown errors become 500/INTERNAL and are logged — never leaked verbatim
// to clients (which would expose pgx internals or migration details).
func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrNoCurrentSeason):
		writeErrorEnvelope(w, http.StatusNotFound, CodeNoCurrentSeason, err.Error())
	case errors.Is(err, service.ErrSeasonComplete):
		writeErrorEnvelope(w, http.StatusUnprocessableEntity, CodeSeasonComplete, err.Error())
	case errors.Is(err, service.ErrPredictionsNotAvailable):
		writeErrorEnvelope(w, http.StatusUnprocessableEntity, CodePredictionsNotAvailable, err.Error())
	case errors.Is(err, service.ErrMatchNotPlayed):
		writeErrorEnvelope(w, http.StatusUnprocessableEntity, CodeMatchNotPlayed, err.Error())
	case errors.Is(err, service.ErrInvalidScore):
		writeErrorEnvelope(w, http.StatusBadRequest, CodeInvalidScore, err.Error())
	case errors.Is(err, repository.ErrNotFound):
		writeErrorEnvelope(w, http.StatusNotFound, CodeNotFound, "resource not found")
	case errors.Is(err, repository.ErrConflict):
		writeErrorEnvelope(w, http.StatusConflict, CodeConflict, "resource conflict")
	default:
		log.Printf("internal error: %v", err)
		writeErrorEnvelope(w, http.StatusInternalServerError, CodeInternal, "internal server error")
	}
}

func writeErrorEnvelope(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorResponse{Error: msg, Code: code})
}

// writeJSON encodes v and writes it with the given status. Encoding errors
// are logged but otherwise swallowed — the response status is already
// committed by the time we discover one.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("json encode: %v", err)
	}
}
