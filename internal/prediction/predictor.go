// Package prediction estimates championship probabilities from the current
// state of a season. The PredictionEngine interface decouples the service
// layer from any specific algorithm — Monte Carlo is the implementation
// here, but the seam would also fit an analytic Poisson method or a
// learned model later.
package prediction

import "github.com/busrakoyun/premier-league-simulation/internal/domain"

// PredictionEngine estimates each team's championship probability given a
// snapshot of the season so far. Implementations must be safe to call from
// multiple goroutines — the Monte Carlo implementation achieves this by
// scoping its RNG to each Predict call rather than holding shared state.
type PredictionEngine interface {
	Predict(snapshot domain.SeasonSnapshot) ([]domain.PredictionRow, error)
}
