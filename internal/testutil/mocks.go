// Package testutil exposes test doubles that other internal packages can
// pull into their _test files. Living in a separate package keeps these
// helpers out of production binaries.
package testutil

import (
	"context"
	"sync"
	"time"

	"github.com/busrakoyun/premier-league-simulation/internal/domain"
	"github.com/busrakoyun/premier-league-simulation/internal/repository"
)

// Compile-time assertion: FakeLeagueRepo satisfies the repository interface.
// If the interface grows a method, this line breaks before the tests do.
var _ repository.LeagueRepository = (*FakeLeagueRepo)(nil)

// FakeLeagueRepo is an in-memory LeagueRepository for service tests. Tx
// semantics are not modelled — WithTx just calls fn with the same repo —
// because the service tests in this package don't depend on rollback
// behaviour; the Postgres impl covers that with its own integration test.
type FakeLeagueRepo struct {
	mu      sync.Mutex
	teams   []domain.Team
	seasons []domain.Season
	matches []domain.Match
	nextID  int64
}

// NewFakeLeagueRepo seeds the repo with the given teams. The teams slice
// is treated as read-only by the repo.
func NewFakeLeagueRepo(teams []domain.Team) *FakeLeagueRepo {
	cp := make([]domain.Team, len(teams))
	copy(cp, teams)
	return &FakeLeagueRepo{teams: cp, nextID: 1}
}

func (f *FakeLeagueRepo) allocID() int64 {
	id := f.nextID
	f.nextID++
	return id
}

// ---- Teams -----------------------------------------------------------------

func (f *FakeLeagueRepo) ListTeams(_ context.Context) ([]domain.Team, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]domain.Team, len(f.teams))
	copy(out, f.teams)
	return out, nil
}

// ---- Seasons ---------------------------------------------------------------

func (f *FakeLeagueRepo) CreateSeason(_ context.Context, name string, totalWeeks int) (domain.Season, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s := domain.Season{
		ID:         f.allocID(),
		Name:       name,
		TotalWeeks: totalWeeks,
		CreatedAt:  time.Now(),
	}
	f.seasons = append(f.seasons, s)
	return s, nil
}

func (f *FakeLeagueRepo) GetCurrentSeason(_ context.Context) (domain.Season, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.seasons) == 0 {
		return domain.Season{}, repository.ErrNotFound
	}
	return f.seasons[len(f.seasons)-1], nil
}

func (f *FakeLeagueRepo) SetSeasonWeek(_ context.Context, seasonID int64, currentWeek int) (domain.Season, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i, s := range f.seasons {
		if s.ID == seasonID {
			f.seasons[i].CurrentWeek = currentWeek
			return f.seasons[i], nil
		}
	}
	return domain.Season{}, repository.ErrNotFound
}

func (f *FakeLeagueRepo) DeleteSeason(_ context.Context, seasonID int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i, s := range f.seasons {
		if s.ID == seasonID {
			f.seasons = append(f.seasons[:i], f.seasons[i+1:]...)
			return nil
		}
	}
	return repository.ErrNotFound
}

// ---- Matches ---------------------------------------------------------------

func (f *FakeLeagueRepo) CreateMatch(_ context.Context, seasonID int64, week int, homeTeamID, awayTeamID int64) (domain.Match, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	m := domain.Match{
		ID: f.allocID(), SeasonID: seasonID, Week: week,
		HomeTeamID: homeTeamID, AwayTeamID: awayTeamID,
	}
	f.matches = append(f.matches, m)
	return m, nil
}

func (f *FakeLeagueRepo) ListMatchesBySeason(_ context.Context, seasonID int64) ([]domain.Match, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]domain.Match, 0)
	for _, m := range f.matches {
		if m.SeasonID == seasonID {
			out = append(out, m)
		}
	}
	return out, nil
}

func (f *FakeLeagueRepo) ListMatchesByWeek(_ context.Context, seasonID int64, week int) ([]domain.Match, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]domain.Match, 0)
	for _, m := range f.matches {
		if m.SeasonID == seasonID && m.Week == week {
			out = append(out, m)
		}
	}
	return out, nil
}

func (f *FakeLeagueRepo) ListUnplayedMatchesByWeek(_ context.Context, seasonID int64, week int) ([]domain.Match, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]domain.Match, 0)
	for _, m := range f.matches {
		if m.SeasonID == seasonID && m.Week == week && !m.Played() {
			out = append(out, m)
		}
	}
	return out, nil
}

func (f *FakeLeagueRepo) GetMatchByID(_ context.Context, matchID int64) (domain.Match, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, m := range f.matches {
		if m.ID == matchID {
			return m, nil
		}
	}
	return domain.Match{}, repository.ErrNotFound
}

func (f *FakeLeagueRepo) RecordMatchResult(_ context.Context, matchID int64, homeGoals, awayGoals int) (domain.Match, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i, m := range f.matches {
		if m.ID == matchID && !m.Played() {
			f.matches[i].MatchResult = &domain.MatchResult{
				HomeGoals: homeGoals,
				AwayGoals: awayGoals,
				PlayedAt:  time.Now(),
			}
			return f.matches[i], nil
		}
	}
	return domain.Match{}, repository.ErrNotFound
}

func (f *FakeLeagueRepo) UpdateMatchScore(_ context.Context, matchID int64, homeGoals, awayGoals int) (domain.Match, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i, m := range f.matches {
		if m.ID == matchID && m.Played() {
			f.matches[i].MatchResult.HomeGoals = homeGoals
			f.matches[i].MatchResult.AwayGoals = awayGoals
			return f.matches[i], nil
		}
	}
	return domain.Match{}, repository.ErrNotFound
}

func (f *FakeLeagueRepo) DeleteMatchesBySeason(_ context.Context, seasonID int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	kept := make([]domain.Match, 0, len(f.matches))
	for _, m := range f.matches {
		if m.SeasonID != seasonID {
			kept = append(kept, m)
		}
	}
	f.matches = kept
	return nil
}

// WithTx in the fake just invokes fn with the same repo. Tx atomicity isn't
// modelled — that's intentional. The Postgres integration test covers the
// real rollback/commit semantics; here we only care about orchestration.
func (f *FakeLeagueRepo) WithTx(ctx context.Context, fn func(ctx context.Context, tx repository.LeagueRepository) error) error {
	return fn(ctx, f)
}

// ---- Test helpers ----------------------------------------------------------

// Snapshot returns a defensive copy of the repo's current state. Useful for
// assertions in tests that want to inspect what the service persisted
// without poking at private fields.
func (f *FakeLeagueRepo) Snapshot() ([]domain.Team, []domain.Season, []domain.Match) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return cloneSlice(f.teams), cloneSlice(f.seasons), cloneSlice(f.matches)
}

func cloneSlice[T any](in []T) []T {
	out := make([]T, len(in))
	copy(out, in)
	return out
}
