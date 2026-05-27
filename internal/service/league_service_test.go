package service_test

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/busrakoyun/premier-league-simulation/internal/domain"
	"github.com/busrakoyun/premier-league-simulation/internal/prediction"
	"github.com/busrakoyun/premier-league-simulation/internal/service"
	"github.com/busrakoyun/premier-league-simulation/internal/simulation"
	"github.com/busrakoyun/premier-league-simulation/internal/testutil"
)

// setup builds a service wired to an in-memory repo with the four PDF teams.
// All randomness is seeded so tests are deterministic — no flake tolerance.
func setup(t *testing.T) (*service.LeagueService, *testutil.FakeLeagueRepo) {
	t.Helper()
	teams := []domain.Team{
		{ID: 1, Name: "Chelsea", ShortName: "CHE", AttackStrength: 1.30, DefenseStrength: 1.20, HomeAdvantage: 1.30},
		{ID: 2, Name: "Arsenal", ShortName: "ARS", AttackStrength: 1.05, DefenseStrength: 1.05, HomeAdvantage: 1.30},
		{ID: 3, Name: "Manchester City", ShortName: "MCI", AttackStrength: 1.00, DefenseStrength: 1.00, HomeAdvantage: 1.30},
		{ID: 4, Name: "Liverpool", ShortName: "LIV", AttackStrength: 0.85, DefenseStrength: 0.85, HomeAdvantage: 1.30},
	}
	repo := testutil.NewFakeLeagueRepo(teams)
	sim := simulation.NewPoissonSimulator()
	rng := simulation.NewRNG(42)
	predictor := prediction.NewMonteCarlo(sim, 200, 42)
	svc := service.New(repo, sim, predictor, rng)
	return svc, repo
}

func TestEnsureSeason_CreatesSeasonAndFixtures(t *testing.T) {
	svc, repo := setup(t)
	ctx := context.Background()

	if err := svc.EnsureSeason(ctx, "2025-26"); err != nil {
		t.Fatalf("EnsureSeason: %v", err)
	}

	_, seasons, matches := repo.Snapshot()
	if len(seasons) != 1 {
		t.Fatalf("seasons: got %d, want 1", len(seasons))
	}
	if seasons[0].TotalWeeks != 6 {
		t.Errorf("TotalWeeks = %d, want 6 (double round-robin for 4 teams)", seasons[0].TotalWeeks)
	}
	if len(matches) != 12 {
		t.Errorf("matches: got %d, want 12", len(matches))
	}
	for _, m := range matches {
		if m.Played() {
			t.Errorf("freshly created fixture %d should be unplayed", m.ID)
		}
	}
}

func TestEnsureSeason_IsIdempotent(t *testing.T) {
	svc, repo := setup(t)
	ctx := context.Background()

	if err := svc.EnsureSeason(ctx, "first"); err != nil {
		t.Fatalf("EnsureSeason 1: %v", err)
	}
	if err := svc.EnsureSeason(ctx, "second"); err != nil {
		t.Fatalf("EnsureSeason 2: %v", err)
	}

	_, seasons, matches := repo.Snapshot()
	if len(seasons) != 1 || seasons[0].Name != "first" {
		t.Errorf("seasons should remain unchanged on re-call: %+v", seasons)
	}
	if len(matches) != 12 {
		t.Errorf("matches should remain at 12, got %d", len(matches))
	}
}

func TestNextWeek_PlaysBothMatchesAndAdvancesWeek(t *testing.T) {
	svc, _ := setup(t)
	ctx := context.Background()
	mustNoErr(t, svc.EnsureSeason(ctx, "test"))

	result, err := svc.NextWeek(ctx)
	if err != nil {
		t.Fatalf("NextWeek: %v", err)
	}
	if result.Week != 1 {
		t.Errorf("Week = %d, want 1", result.Week)
	}
	if len(result.Matches) != 2 {
		t.Errorf("matches: got %d, want 2", len(result.Matches))
	}
	for _, m := range result.Matches {
		if !m.Played() {
			t.Errorf("match %d should be played", m.ID)
		}
		if m.HomeGoals < 0 || m.AwayGoals < 0 {
			t.Errorf("match %d has negative score: %+v", m.ID, m)
		}
	}

	season, _ := svc.GetCurrentSeason(ctx)
	if season.CurrentWeek != 1 {
		t.Errorf("CurrentWeek = %d, want 1", season.CurrentWeek)
	}
}

func TestPlayAll_FinishesEntireSeasonInWeekOrder(t *testing.T) {
	svc, _ := setup(t)
	ctx := context.Background()
	mustNoErr(t, svc.EnsureSeason(ctx, "test"))

	weeks, err := svc.PlayAll(ctx)
	if err != nil {
		t.Fatalf("PlayAll: %v", err)
	}
	if len(weeks) != 6 {
		t.Fatalf("weeks: got %d, want 6", len(weeks))
	}
	for i, w := range weeks {
		if w.Week != i+1 {
			t.Errorf("weeks[%d].Week = %d, want %d", i, w.Week, i+1)
		}
		if len(w.Matches) != 2 {
			t.Errorf("week %d has %d matches, want 2", w.Week, len(w.Matches))
		}
	}

	season, _ := svc.GetCurrentSeason(ctx)
	if !season.Finished() {
		t.Errorf("season should be finished: %+v", season)
	}
}

func TestNextWeek_RejectsAfterSeasonComplete(t *testing.T) {
	svc, _ := setup(t)
	ctx := context.Background()
	mustNoErr(t, svc.EnsureSeason(ctx, "test"))
	if _, err := svc.PlayAll(ctx); err != nil {
		t.Fatal(err)
	}

	if _, err := svc.NextWeek(ctx); !errors.Is(err, service.ErrSeasonComplete) {
		t.Errorf("NextWeek after finish: got %v, want ErrSeasonComplete", err)
	}
	if _, err := svc.PlayAll(ctx); !errors.Is(err, service.ErrSeasonComplete) {
		t.Errorf("PlayAll after finish: got %v, want ErrSeasonComplete", err)
	}
}

func TestPredict_RejectsBeforeWeek4(t *testing.T) {
	svc, _ := setup(t)
	ctx := context.Background()
	mustNoErr(t, svc.EnsureSeason(ctx, "test"))

	// week 0
	if _, err := svc.Predict(ctx); !errors.Is(err, service.ErrPredictionsNotAvailable) {
		t.Errorf("Predict at week 0: got %v, want ErrPredictionsNotAvailable", err)
	}
	for w := 1; w <= 3; w++ {
		if _, err := svc.NextWeek(ctx); err != nil { t.Fatal(err) }
	}
	if _, err := svc.Predict(ctx); !errors.Is(err, service.ErrPredictionsNotAvailable) {
		t.Errorf("Predict at week 3: got %v, want ErrPredictionsNotAvailable", err)
	}

	if _, err := svc.NextWeek(ctx); err != nil { t.Fatal(err) } // week 4
	rows, err := svc.Predict(ctx)
	if err != nil {
		t.Fatalf("Predict at week 4: %v", err)
	}
	if len(rows) != 4 {
		t.Fatalf("predictions: got %d rows, want 4", len(rows))
	}

	sum := 0.0
	for _, r := range rows {
		if r.ChampionshipRate < 0 || r.ChampionshipRate > 1 {
			t.Errorf("invalid rate: %+v", r)
		}
		sum += r.ChampionshipRate
	}
	if math.Abs(sum-1.0) > 1e-9 {
		t.Errorf("rates sum to %v, want 1.0 exactly", sum)
	}
}

func TestEditMatch_UpdatesScoreAndStandingsReflectIt(t *testing.T) {
	svc, _ := setup(t)
	ctx := context.Background()
	mustNoErr(t, svc.EnsureSeason(ctx, "test"))
	week, err := svc.NextWeek(ctx)
	if err != nil {
		t.Fatalf("NextWeek: %v", err)
	}
	m := week.Matches[0]
	homeID := m.HomeTeamID

	// Force a 5-0 home win.
	updated, err := svc.EditMatch(ctx, m.ID, 5, 0)
	if err != nil {
		t.Fatalf("EditMatch: %v", err)
	}
	if updated.HomeGoals != 5 || updated.AwayGoals != 0 {
		t.Errorf("updated match: %+v", updated)
	}

	rows, err := svc.GetStandings(ctx)
	if err != nil {
		t.Fatalf("GetStandings: %v", err)
	}
	homeRow := findStanding(rows, homeID)
	if homeRow.GoalsFor < 5 {
		t.Errorf("home team should have at least 5 GoalsFor after edit: %+v", homeRow)
	}
	if homeRow.Won < 1 {
		t.Errorf("home team should have a win recorded: %+v", homeRow)
	}
}

func TestEditMatch_RejectsNegativeScore(t *testing.T) {
	svc, _ := setup(t)
	ctx := context.Background()
	mustNoErr(t, svc.EnsureSeason(ctx, "test"))
	week, _ := svc.NextWeek(ctx)
	m := week.Matches[0]

	if _, err := svc.EditMatch(ctx, m.ID, -1, 0); !errors.Is(err, service.ErrInvalidScore) {
		t.Errorf("got %v, want ErrInvalidScore", err)
	}
}

func TestEditMatch_RejectsUnplayedMatch(t *testing.T) {
	svc, repo := setup(t)
	ctx := context.Background()
	mustNoErr(t, svc.EnsureSeason(ctx, "test"))

	_, _, matches := repo.Snapshot()
	if len(matches) == 0 {
		t.Fatal("no matches after EnsureSeason")
	}
	// First match exists but hasn't been played.
	if _, err := svc.EditMatch(ctx, matches[0].ID, 1, 0); !errors.Is(err, service.ErrMatchNotPlayed) {
		t.Errorf("got %v, want ErrMatchNotPlayed", err)
	}
}

func TestResetSeason_WipesResultsAndRegeneratesFixtures(t *testing.T) {
	svc, repo := setup(t)
	ctx := context.Background()
	mustNoErr(t, svc.EnsureSeason(ctx, "test"))
	if _, err := svc.NextWeek(ctx); err != nil { t.Fatal(err) }
	if _, err := svc.NextWeek(ctx); err != nil { t.Fatal(err) }

	if err := svc.ResetSeason(ctx); err != nil {
		t.Fatalf("ResetSeason: %v", err)
	}

	season, _ := svc.GetCurrentSeason(ctx)
	if season.CurrentWeek != 0 {
		t.Errorf("CurrentWeek after reset = %d, want 0", season.CurrentWeek)
	}
	_, _, matches := repo.Snapshot()
	if len(matches) != 12 {
		t.Errorf("matches after reset: got %d, want 12", len(matches))
	}
	for _, m := range matches {
		if m.Played() {
			t.Errorf("match %d should be unplayed after reset", m.ID)
		}
	}
}

func TestGetStandings_AllTeamsCountedAfterPlay(t *testing.T) {
	svc, _ := setup(t)
	ctx := context.Background()
	mustNoErr(t, svc.EnsureSeason(ctx, "test"))
	if _, err := svc.NextWeek(ctx); err != nil { t.Fatal(err) }
	if _, err := svc.NextWeek(ctx); err != nil { t.Fatal(err) }

	rows, err := svc.GetStandings(ctx)
	if err != nil {
		t.Fatalf("GetStandings: %v", err)
	}
	if len(rows) != 4 {
		t.Fatalf("rows: got %d, want 4", len(rows))
	}
	for _, r := range rows {
		if r.Played != 2 {
			t.Errorf("team %q has Played=%d after 2 weeks, want 2", r.TeamName, r.Played)
		}
	}
}

func TestGetCurrentSeason_ReturnsNoCurrentSeasonError(t *testing.T) {
	svc, _ := setup(t)
	ctx := context.Background()
	// EnsureSeason not called.
	if _, err := svc.GetCurrentSeason(ctx); !errors.Is(err, service.ErrNoCurrentSeason) {
		t.Errorf("got %v, want ErrNoCurrentSeason", err)
	}
}

// ---- helpers ---------------------------------------------------------------

func mustNoErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func findStanding(rows []domain.StandingRow, teamID int64) domain.StandingRow {
	for _, r := range rows {
		if r.TeamID == teamID {
			return r
		}
	}
	return domain.StandingRow{}
}
