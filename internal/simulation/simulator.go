// Package simulation produces match scores from team strengths.
// The MatchSimulator interface decouples the prediction and service
// layers from any particular goal model — Poisson here, but the seam
// would let us swap in (say) a Dixon-Coles bivariate model later.
package simulation

import (
	"math/rand/v2"

	"github.com/busrakoyun/premier-league-simulation/internal/domain"
)

// MatchSimulator produces a score for a single match given the two teams.
// Simulate takes the RNG as a parameter so the simulator itself stays
// stateless w.r.t. randomness: each Monte Carlo Predict call can scope its
// own RNG without a per-simulator mutex, and tests can produce reproducible
// sequences just by handing the same seed in.
type MatchSimulator interface {
	Simulate(rng *rand.Rand, home, away domain.Team) domain.MatchScore
}
