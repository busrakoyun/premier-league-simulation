package simulation

import (
	"math"
	"math/rand/v2"

	"github.com/busrakoyun/premier-league-simulation/internal/domain"
)

// DefaultBaseGoalRate is the expected goals per side in a "neutral" match —
// both teams at strength 1.0, no home advantage. Roughly the Premier League
// long-run average of ~1.4 goals per team per game.
const DefaultBaseGoalRate = 1.4

// SamplePoisson draws a non-negative integer from Poisson(lambda) using
// Knuth's algorithm. Returns 0 for non-positive lambda.
//
// Knuth multiplies uniform draws until their product falls below exp(-λ).
// It's fine for our regime — λ stays under ~5 with realistic team strengths
// — but degrades for very large λ where exp(-λ) underflows. If we ever
// model leagues where λ > ~30, swap in PTRS or normal approximation.
func SamplePoisson(rng *rand.Rand, lambda float64) int {
	if lambda <= 0 {
		return 0
	}
	L := math.Exp(-lambda)
	k := 0
	p := 1.0
	for {
		k++
		p *= rng.Float64()
		if p <= L {
			return k - 1
		}
	}
}

// PoissonSimulator models match scores as independent Poisson draws.
//
// λ_home = BaseGoalRate × home.AttackStrength × home.HomeAdvantage / away.DefenseStrength
// λ_away = BaseGoalRate × away.AttackStrength                       / home.DefenseStrength
//
// Why Poisson: empirically a good fit for football goal counts (Maher 1982
// baseline); independence between sides is a known simplification that
// trades a small amount of realism for a much simpler model — fine at the
// scale of this case.
//
// Why the attack/defense decomposition: the PDF explicitly requires that
// match outcomes depend on team strengths. Splitting strength into an
// attack multiplier (raises my λ) and a defense multiplier (lowers my
// opponent's λ) gives an interpretable directional model — a team can be
// good in attack and bad in defense and the maths reflects that.
type PoissonSimulator struct {
	BaseGoalRate float64
}

// NewPoissonSimulator returns a simulator using the league-average base rate.
func NewPoissonSimulator() *PoissonSimulator {
	return &PoissonSimulator{BaseGoalRate: DefaultBaseGoalRate}
}

// Compile-time assertion that PoissonSimulator satisfies MatchSimulator.
var _ MatchSimulator = (*PoissonSimulator)(nil)

func (s *PoissonSimulator) Simulate(rng *rand.Rand, home, away domain.Team) domain.MatchScore {
	lambdaHome := s.BaseGoalRate * home.AttackStrength * home.HomeAdvantage / away.DefenseStrength
	lambdaAway := s.BaseGoalRate * away.AttackStrength / home.DefenseStrength
	return domain.MatchScore{
		HomeGoals: SamplePoisson(rng, lambdaHome),
		AwayGoals: SamplePoisson(rng, lambdaAway),
	}
}
