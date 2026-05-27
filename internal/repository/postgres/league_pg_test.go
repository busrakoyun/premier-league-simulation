package postgres_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/busrakoyun/premier-league-simulation/internal/repository"
	"github.com/busrakoyun/premier-league-simulation/internal/repository/postgres"
)

// These tests round-trip the repo against a real Postgres. They skip when
// TEST_DATABASE_URL is unset so `go test ./...` stays green on CI without a DB.
//
// To run locally:
//   make db-up && make migrate-up
//   export TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5432/premier_league?sslmode=disable"
//   go test ./internal/repository/postgres/...

func mustPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping Postgres integration test")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// truncateSeasons wipes seasons (cascading to matches) so each test starts
// from a known state. Teams are kept since they're part of the seed migration.
func truncateSeasons(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), "TRUNCATE seasons RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("truncate: %v", err)
	}
}

func TestLeagueRepo_TeamsRoundTrip(t *testing.T) {
	pool := mustPool(t)
	repo := postgres.New(pool)

	teams, err := repo.ListTeams(context.Background())
	if err != nil {
		t.Fatalf("ListTeams: %v", err)
	}
	if len(teams) != 4 {
		t.Fatalf("expected 4 seeded teams, got %d", len(teams))
	}
	for _, tm := range teams {
		if tm.AttackStrength <= 0 || tm.DefenseStrength <= 0 || tm.HomeAdvantage <= 0 {
			t.Errorf("team %q has non-positive strength: %+v", tm.Name, tm)
		}
	}
}

func TestLeagueRepo_MatchRoundTrip(t *testing.T) {
	pool := mustPool(t)
	truncateSeasons(t, pool)
	defer truncateSeasons(t, pool)

	ctx := context.Background()
	repo := postgres.New(pool)

	teams, err := repo.ListTeams(ctx)
	if err != nil {
		t.Fatalf("ListTeams: %v", err)
	}

	season, err := repo.CreateSeason(ctx, "test-season", 6)
	if err != nil {
		t.Fatalf("CreateSeason: %v", err)
	}
	if season.TotalWeeks != 6 || season.CurrentWeek != 0 {
		t.Errorf("season = %+v, want TotalWeeks=6, CurrentWeek=0", season)
	}

	current, err := repo.GetCurrentSeason(ctx)
	if err != nil {
		t.Fatalf("GetCurrentSeason: %v", err)
	}
	if current.ID != season.ID {
		t.Errorf("GetCurrentSeason.ID = %d, want %d", current.ID, season.ID)
	}

	m, err := repo.CreateMatch(ctx, season.ID, 1, teams[0].ID, teams[1].ID)
	if err != nil {
		t.Fatalf("CreateMatch: %v", err)
	}
	if m.Played() {
		t.Fatalf("new match should be unplayed")
	}

	played, err := repo.RecordMatchResult(ctx, m.ID, 2, 1)
	if err != nil {
		t.Fatalf("RecordMatchResult: %v", err)
	}
	if !played.Played() {
		t.Fatalf("match should be played after RecordMatchResult")
	}
	if played.HomeGoals != 2 || played.AwayGoals != 1 {
		t.Errorf("score = %d-%d, want 2-1", played.HomeGoals, played.AwayGoals)
	}

	edited, err := repo.UpdateMatchScore(ctx, m.ID, 3, 2)
	if err != nil {
		t.Fatalf("UpdateMatchScore: %v", err)
	}
	if edited.HomeGoals != 3 || edited.AwayGoals != 2 {
		t.Errorf("edited score = %d-%d, want 3-2", edited.HomeGoals, edited.AwayGoals)
	}

	unplayed, err := repo.CreateMatch(ctx, season.ID, 2, teams[2].ID, teams[3].ID)
	if err != nil {
		t.Fatalf("CreateMatch 2: %v", err)
	}
	// UpdateMatchScore requires played_at IS NOT NULL, so an unplayed match
	// produces a not-found from the empty UPDATE...RETURNING result.
	if _, err := repo.UpdateMatchScore(ctx, unplayed.ID, 1, 1); !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("UpdateMatchScore on unplayed: got %v, want ErrNotFound", err)
	}

	if _, err := repo.GetMatchByID(ctx, 999_999_999); !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("GetMatchByID missing: got %v, want ErrNotFound", err)
	}
}

func TestLeagueRepo_WithTx_RollsBackOnError(t *testing.T) {
	pool := mustPool(t)
	truncateSeasons(t, pool)
	defer truncateSeasons(t, pool)

	ctx := context.Background()
	repo := postgres.New(pool)
	teams, _ := repo.ListTeams(ctx)
	season, _ := repo.CreateSeason(ctx, "tx-rollback", 6)

	boom := errors.New("boom")
	err := repo.WithTx(ctx, func(ctx context.Context, txRepo repository.LeagueRepository) error {
		if _, err := txRepo.CreateMatch(ctx, season.ID, 1, teams[0].ID, teams[1].ID); err != nil {
			return err
		}
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("WithTx error: got %v, want boom", err)
	}

	matches, err := repo.ListMatchesBySeason(ctx, season.ID)
	if err != nil {
		t.Fatalf("ListMatchesBySeason: %v", err)
	}
	if len(matches) != 0 {
		t.Errorf("rolled-back tx left %d matches behind", len(matches))
	}
}

func TestLeagueRepo_WithTx_Commits(t *testing.T) {
	pool := mustPool(t)
	truncateSeasons(t, pool)
	defer truncateSeasons(t, pool)

	ctx := context.Background()
	repo := postgres.New(pool)
	teams, _ := repo.ListTeams(ctx)
	season, _ := repo.CreateSeason(ctx, "tx-commit", 6)

	err := repo.WithTx(ctx, func(ctx context.Context, txRepo repository.LeagueRepository) error {
		_, err := txRepo.CreateMatch(ctx, season.ID, 1, teams[0].ID, teams[1].ID)
		return err
	})
	if err != nil {
		t.Fatalf("WithTx: %v", err)
	}

	matches, err := repo.ListMatchesBySeason(ctx, season.ID)
	if err != nil {
		t.Fatalf("ListMatchesBySeason: %v", err)
	}
	if len(matches) != 1 {
		t.Errorf("after commit, expected 1 match, got %d", len(matches))
	}
}
