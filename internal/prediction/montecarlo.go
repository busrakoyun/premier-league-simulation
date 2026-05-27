package prediction

import (
	"errors"
	"sort"

	"github.com/busrakoyun/premier-league-simulation/internal/domain"
	"github.com/busrakoyun/premier-league-simulation/internal/simulation"
	"github.com/busrakoyun/premier-league-simulation/internal/standings"
)

// MonteCarlo runs N independent simulations of the remaining matches and
// counts how often each team finishes top of the table. Ties at the top
// share the championship equally per iteration (1/k each among k tied
// teams), so the returned rates always sum to exactly 1.
//
// Why single-threaded: at the scale of this case — N=10000 iterations × ≤12
// matches per iteration ≈ 120k simulated matches per Predict — a tight
// sequential loop finishes well under 100ms on a free Render dyno.
// Goroutine overhead and the per-iteration RNG-coordination cost would
// dominate any benefit. Profiled, not assumed.
type MonteCarlo struct {
	sim        simulation.MatchSimulator
	iterations int
	seed       int64 // 0 ⇒ time-based, distinct across calls
}

// Compile-time assertion.
var _ PredictionEngine = (*MonteCarlo)(nil)

// NewMonteCarlo wires up the predictor. iterations must be positive.
// Seed 0 means each Predict call samples a fresh, time-seeded RNG (the
// production default — successive predictions are independent draws).
// A non-zero seed makes Predict deterministic — handy in tests.
func NewMonteCarlo(sim simulation.MatchSimulator, iterations int, seed int64) *MonteCarlo {
	return &MonteCarlo{sim: sim, iterations: iterations, seed: seed}
}

func (mc *MonteCarlo) Predict(snapshot domain.SeasonSnapshot) ([]domain.PredictionRow, error) {
	if mc.iterations <= 0 {
		return nil, errors.New("monte carlo: iterations must be positive")
	}
	if len(snapshot.Teams) == 0 {
		return nil, errors.New("monte carlo: snapshot has no teams")
	}

	// Index teams once; classify matches into played vs. unplayed once.
	byID := make(map[int64]domain.Team, len(snapshot.Teams))
	for _, t := range snapshot.Teams {
		byID[t.ID] = t
	}
	played := make([]domain.Match, 0, len(snapshot.Matches))
	unplayed := make([]domain.Match, 0, len(snapshot.Matches))
	for _, m := range snapshot.Matches {
		if m.Played() {
			played = append(played, m)
		} else {
			unplayed = append(unplayed, m)
		}
	}

	rng := simulation.NewRNG(mc.seed)
	wins := make(map[int64]float64, len(snapshot.Teams))

	// Reuse a single scratch slice across iterations to avoid 10000 fresh
	// allocations. The slice gets re-truncated to len(played) each loop.
	scratch := make([]domain.Match, 0, len(played)+len(unplayed))

	for iter := 0; iter < mc.iterations; iter++ {
		scratch = append(scratch[:0], played...)
		for _, m := range unplayed {
			score := mc.sim.Simulate(rng, byID[m.HomeTeamID], byID[m.AwayTeamID])
			simulated := m
			simulated.MatchResult = &domain.MatchResult{
				HomeGoals: score.HomeGoals,
				AwayGoals: score.AwayGoals,
			}
			scratch = append(scratch, simulated)
		}

		rows := standings.Calculate(snapshot.Teams, scratch)
		topCount := countTopTied(rows)
		share := 1.0 / float64(topCount)
		for i := 0; i < topCount; i++ {
			wins[rows[i].TeamID] += share
		}
	}

	out := make([]domain.PredictionRow, 0, len(snapshot.Teams))
	for _, t := range snapshot.Teams {
		out = append(out, domain.PredictionRow{
			TeamID:           t.ID,
			TeamName:         t.Name,
			ChampionshipRate: wins[t.ID] / float64(mc.iterations),
		})
	}
	sortPredictions(out)
	return out, nil
}

// countTopTied returns the number of teams sharing the top of the standings
// (same points, goal difference, and goals-for as rows[0]). The Calculator's
// final tiebreaker (team name) is intentionally ignored here so that a real
// tie at the top splits the championship rather than being broken
// alphabetically.
func countTopTied(rows []domain.StandingRow) int {
	if len(rows) == 0 {
		return 0
	}
	top := rows[0]
	n := 1
	for i := 1; i < len(rows); i++ {
		r := rows[i]
		if r.Points == top.Points && r.GoalDifference == top.GoalDifference && r.GoalsFor == top.GoalsFor {
			n++
		} else {
			break
		}
	}
	return n
}

func sortPredictions(rows []domain.PredictionRow) {
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].ChampionshipRate != rows[j].ChampionshipRate {
			return rows[i].ChampionshipRate > rows[j].ChampionshipRate
		}
		return rows[i].TeamName < rows[j].TeamName
	})
}
