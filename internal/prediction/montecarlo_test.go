package prediction_test

import (
	"math"
	"math/rand/v2"
	"testing"

	"github.com/busrakoyun/premier-league-simulation/internal/domain"
	"github.com/busrakoyun/premier-league-simulation/internal/prediction"
	"github.com/busrakoyun/premier-league-simulation/internal/simulation"
)

// stubSim always returns the same hard-coded score. Lets MC logic be tested
// in isolation from any particular goal model.
type stubSim struct {
	home, away int
}

func (s stubSim) Simulate(_ *rand.Rand, _, _ domain.Team) domain.MatchScore {
	return domain.MatchScore{HomeGoals: s.home, AwayGoals: s.away}
}

func TestMonteCarlo_RatesSumToOne(t *testing.T) {
	teams := []domain.Team{
		{ID: 1, Name: "A", AttackStrength: 1.0, DefenseStrength: 1.0, HomeAdvantage: 1.3},
		{ID: 2, Name: "B", AttackStrength: 1.0, DefenseStrength: 1.0, HomeAdvantage: 1.3},
		{ID: 3, Name: "C", AttackStrength: 1.0, DefenseStrength: 1.0, HomeAdvantage: 1.3},
		{ID: 4, Name: "D", AttackStrength: 1.0, DefenseStrength: 1.0, HomeAdvantage: 1.3},
	}
	matches := []domain.Match{
		{ID: 1, Week: 1, HomeTeamID: 1, AwayTeamID: 2},
		{ID: 2, Week: 1, HomeTeamID: 3, AwayTeamID: 4},
		{ID: 3, Week: 2, HomeTeamID: 1, AwayTeamID: 3},
		{ID: 4, Week: 2, HomeTeamID: 2, AwayTeamID: 4},
	}
	snap := domain.SeasonSnapshot{Teams: teams, Matches: matches, CurrentWeek: 0, TotalWeeks: 6}

	mc := prediction.NewMonteCarlo(simulation.NewPoissonSimulator(), 500, 42)
	rows, err := mc.Predict(snap)
	if err != nil {
		t.Fatalf("Predict: %v", err)
	}
	if len(rows) != 4 {
		t.Fatalf("got %d rows, want 4", len(rows))
	}

	total := 0.0
	for _, r := range rows {
		total += r.ChampionshipRate
	}
	if math.Abs(total-1.0) > 1e-9 {
		t.Errorf("rates sum to %v, want 1.0 exactly", total)
	}
}

func TestMonteCarlo_StrongTeamFavored(t *testing.T) {
	teams := []domain.Team{
		{ID: 1, Name: "Strong", AttackStrength: 1.6, DefenseStrength: 1.6, HomeAdvantage: 1.3},
		{ID: 2, Name: "Weak1", AttackStrength: 0.7, DefenseStrength: 0.7, HomeAdvantage: 1.3},
		{ID: 3, Name: "Weak2", AttackStrength: 0.7, DefenseStrength: 0.7, HomeAdvantage: 1.3},
		{ID: 4, Name: "Weak3", AttackStrength: 0.7, DefenseStrength: 0.7, HomeAdvantage: 1.3},
	}
	matches := genDoubleRoundRobin(teams)
	snap := domain.SeasonSnapshot{Teams: teams, Matches: matches, CurrentWeek: 0, TotalWeeks: 6}

	mc := prediction.NewMonteCarlo(simulation.NewPoissonSimulator(), 1000, 42)
	rows, err := mc.Predict(snap)
	if err != nil {
		t.Fatalf("Predict: %v", err)
	}

	if rows[0].TeamName != "Strong" {
		t.Errorf("expected Strong on top, got %q (rates=%+v)", rows[0].TeamName, rows)
	}
	if rows[0].ChampionshipRate < 0.6 {
		t.Errorf("Strong rate %.3f should be > 0.6 against three much weaker sides",
			rows[0].ChampionshipRate)
	}
}

func TestMonteCarlo_DeterministicForFixedSeed(t *testing.T) {
	teams := []domain.Team{
		{ID: 1, Name: "A", AttackStrength: 1.1, DefenseStrength: 1.1, HomeAdvantage: 1.3},
		{ID: 2, Name: "B", AttackStrength: 1.0, DefenseStrength: 1.0, HomeAdvantage: 1.3},
		{ID: 3, Name: "C", AttackStrength: 0.9, DefenseStrength: 0.9, HomeAdvantage: 1.3},
		{ID: 4, Name: "D", AttackStrength: 0.8, DefenseStrength: 0.8, HomeAdvantage: 1.3},
	}
	matches := genDoubleRoundRobin(teams)
	snap := domain.SeasonSnapshot{Teams: teams, Matches: matches, TotalWeeks: 6}

	mcA := prediction.NewMonteCarlo(simulation.NewPoissonSimulator(), 200, 123)
	mcB := prediction.NewMonteCarlo(simulation.NewPoissonSimulator(), 200, 123)

	rowsA, _ := mcA.Predict(snap)
	rowsB, _ := mcB.Predict(snap)
	for i := range rowsA {
		if rowsA[i].ChampionshipRate != rowsB[i].ChampionshipRate {
			t.Errorf("row %d diverged: %v vs %v",
				i, rowsA[i].ChampionshipRate, rowsB[i].ChampionshipRate)
		}
	}
}

func TestMonteCarlo_AlwaysHomeWinsGivesEverythingToHome(t *testing.T) {
	// One match remaining. A always-home-wins stub means team 1 always picks
	// up 3 pts and team 2 picks up 0. With no other matches, the champion is
	// always team 1.
	teams := []domain.Team{
		{ID: 1, Name: "Home"},
		{ID: 2, Name: "Away"},
	}
	matches := []domain.Match{
		{ID: 1, Week: 1, HomeTeamID: 1, AwayTeamID: 2},
	}
	snap := domain.SeasonSnapshot{Teams: teams, Matches: matches, TotalWeeks: 1}

	mc := prediction.NewMonteCarlo(stubSim{home: 2, away: 0}, 100, 1)
	rows, err := mc.Predict(snap)
	if err != nil {
		t.Fatalf("Predict: %v", err)
	}
	home := findPred(rows, "Home")
	if home.ChampionshipRate != 1.0 {
		t.Errorf("Home rate = %v, want 1.0", home.ChampionshipRate)
	}
}

func TestMonteCarlo_RejectsBadInputs(t *testing.T) {
	sim := simulation.NewPoissonSimulator()
	if _, err := prediction.NewMonteCarlo(sim, 0, 1).Predict(domain.SeasonSnapshot{Teams: []domain.Team{{ID: 1}}}); err == nil {
		t.Errorf("expected error for 0 iterations")
	}
	if _, err := prediction.NewMonteCarlo(sim, 100, 1).Predict(domain.SeasonSnapshot{}); err == nil {
		t.Errorf("expected error for empty teams")
	}
}

func genDoubleRoundRobin(teams []domain.Team) []domain.Match {
	var out []domain.Match
	var id int64
	for i, h := range teams {
		for j, a := range teams {
			if i == j {
				continue
			}
			id++
			out = append(out, domain.Match{
				ID: id, SeasonID: 1, Week: 1,
				HomeTeamID: h.ID, AwayTeamID: a.ID,
			})
		}
	}
	return out
}

func findPred(rows []domain.PredictionRow, name string) domain.PredictionRow {
	for _, r := range rows {
		if r.TeamName == name {
			return r
		}
	}
	return domain.PredictionRow{}
}
