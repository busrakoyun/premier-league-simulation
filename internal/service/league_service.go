// Package service holds the league orchestration: it owns the repository,
// the live match simulator, the prediction engine, and a per-instance
// mutex that serialises writes. The HTTP layer talks to LeagueService;
// LeagueService talks to interfaces — repository, simulator, predictor —
// so unit tests can plug in mocks without touching DB or RNG state.
package service

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"sync"

	"github.com/busrakoyun/premier-league-simulation/internal/domain"
	"github.com/busrakoyun/premier-league-simulation/internal/prediction"
	"github.com/busrakoyun/premier-league-simulation/internal/repository"
	"github.com/busrakoyun/premier-league-simulation/internal/simulation"
	"github.com/busrakoyun/premier-league-simulation/internal/standings"
)

// PredictionsAvailableAfterWeek is the number of completed weeks required
// before championship predictions are exposed. The PDF's Figure 1.a is the
// first figure to show predictions and corresponds to "after the 4th week",
// so predictions become available once CurrentWeek >= 4.
const PredictionsAvailableAfterWeek = 4

// WeekResult is the per-week payload returned by NextWeek / PlayAll —
// what got played in that week.
type WeekResult struct {
	Week    int
	Matches []domain.Match
}

// LeagueService composes its collaborators: every field is an interface
// (repo / simulator / predictor) plus a clock-free RNG dedicated to the
// live next-week path. The single sync.Mutex serialises all write paths
// for this process — for a single Render instance that's the right level
// of complexity. Horizontal scaling would warrant Postgres row-version
// CAS instead; not this case.
type LeagueService struct {
	repo      repository.LeagueRepository
	simulator simulation.MatchSimulator
	predictor prediction.PredictionEngine
	rng       *rand.Rand

	mu sync.Mutex
}

// New wires the service. The RNG is used only by the live simulation path
// (NextWeek / PlayAll); the predictor manages its own RNG internally so
// concurrent predictions don't share state with live writes.
func New(
	repo repository.LeagueRepository,
	simulator simulation.MatchSimulator,
	predictor prediction.PredictionEngine,
	rng *rand.Rand,
) *LeagueService {
	return &LeagueService{
		repo:      repo,
		simulator: simulator,
		predictor: predictor,
		rng:       rng,
	}
}

// ---- Read-only queries -----------------------------------------------------

// GetTeams returns the league's teams.
func (s *LeagueService) GetTeams(ctx context.Context) ([]domain.Team, error) {
	return s.repo.ListTeams(ctx)
}

// GetCurrentSeason returns the active season, translating the repo's
// NotFound to the service's ErrNoCurrentSeason so the HTTP layer can map
// it to a 404 without importing repository.
func (s *LeagueService) GetCurrentSeason(ctx context.Context) (domain.Season, error) {
	season, err := s.repo.GetCurrentSeason(ctx)
	if errors.Is(err, repository.ErrNotFound) {
		return domain.Season{}, ErrNoCurrentSeason
	}
	return season, err
}

// GetMatches returns every match in the current season, played and unplayed.
func (s *LeagueService) GetMatches(ctx context.Context) ([]domain.Match, error) {
	season, err := s.GetCurrentSeason(ctx)
	if err != nil {
		return nil, err
	}
	return s.repo.ListMatchesBySeason(ctx, season.ID)
}

// GetStandings computes the league table from the current set of played
// matches. Pure function; no caching — see plan flag #1 for the rationale
// (twelve rows compute in microseconds, a cache buys nothing).
func (s *LeagueService) GetStandings(ctx context.Context) ([]domain.StandingRow, error) {
	_, teams, matches, err := s.snapshotData(ctx)
	if err != nil {
		return nil, err
	}
	return standings.Calculate(teams, matches), nil
}

// Predict returns championship probabilities, or ErrPredictionsNotAvailable
// before the required number of weeks have been played.
func (s *LeagueService) Predict(ctx context.Context) ([]domain.PredictionRow, error) {
	season, teams, matches, err := s.snapshotData(ctx)
	if err != nil {
		return nil, err
	}
	if season.CurrentWeek < PredictionsAvailableAfterWeek {
		return nil, ErrPredictionsNotAvailable
	}
	return s.predictor.Predict(domain.SeasonSnapshot{
		Teams:       teams,
		Matches:     matches,
		CurrentWeek: season.CurrentWeek,
		TotalWeeks:  season.TotalWeeks,
	})
}

// ---- Write paths (mutex-protected) -----------------------------------------

// EnsureSeason is the startup helper: if no season exists, it creates one
// and generates the round-robin fixtures inside a single transaction so
// callers never see a half-initialised state. Idempotent — subsequent
// calls are no-ops.
func (s *LeagueService) EnsureSeason(ctx context.Context, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.repo.GetCurrentSeason(ctx)
	if err == nil {
		return nil // season already exists
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return err
	}

	teams, err := s.repo.ListTeams(ctx)
	if err != nil {
		return err
	}
	fixtures := standings.GenerateFixtures(teams)
	if len(fixtures) == 0 {
		return fmt.Errorf("service: cannot generate fixtures for %d teams", len(teams))
	}
	totalWeeks := maxWeek(fixtures)

	return s.repo.WithTx(ctx, func(ctx context.Context, tx repository.LeagueRepository) error {
		season, err := tx.CreateSeason(ctx, name, totalWeeks)
		if err != nil {
			return err
		}
		for _, f := range fixtures {
			if _, err := tx.CreateMatch(ctx, season.ID, f.Week, f.HomeTeamID, f.AwayTeamID); err != nil {
				return err
			}
		}
		return nil
	})
}

// NextWeek simulates every unplayed fixture in the next week and advances
// the season pointer — all inside one transaction so a half-written week
// can't leak into the standings.
func (s *LeagueService) NextWeek(ctx context.Context) (WeekResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.playWeeksLocked(ctx, 1)
}

// PlayAll runs the simulation from the current week through to the end of
// the season, returning every week's results in order. Same transactional
// guarantee as NextWeek: the whole run commits together or rolls back.
func (s *LeagueService) PlayAll(ctx context.Context) ([]WeekResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	season, err := s.GetCurrentSeason(ctx)
	if err != nil {
		return nil, err
	}
	remaining := season.TotalWeeks - season.CurrentWeek
	if remaining <= 0 {
		return nil, ErrSeasonComplete
	}
	results, err := s.playWeeksLocked(ctx, remaining)
	if err != nil {
		return nil, err
	}
	// Re-fetch ordered weeks. playWeeksLocked returned a single WeekResult
	// for the simple NextWeek case; here we need the slice.
	return s.splitIntoWeeks(results), nil
}

// EditMatch updates a played match's score. Standings recompute on read
// (they're a pure function over matches) so no cache rebuild is needed —
// the next GET /standings reflects the edit immediately.
func (s *LeagueService) EditMatch(ctx context.Context, matchID int64, homeGoals, awayGoals int) (domain.Match, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if homeGoals < 0 || awayGoals < 0 {
		return domain.Match{}, ErrInvalidScore
	}

	existing, err := s.repo.GetMatchByID(ctx, matchID)
	if err != nil {
		return domain.Match{}, err
	}
	if !existing.Played() {
		return domain.Match{}, ErrMatchNotPlayed
	}
	return s.repo.UpdateMatchScore(ctx, matchID, homeGoals, awayGoals)
}

// ResetSeason wipes the current season's matches, resets the week counter
// to zero, and regenerates the fixtures. Useful as a demo affordance — not
// a PDF requirement, but cheap to expose.
func (s *LeagueService) ResetSeason(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	season, err := s.GetCurrentSeason(ctx)
	if err != nil {
		return err
	}
	teams, err := s.repo.ListTeams(ctx)
	if err != nil {
		return err
	}
	fixtures := standings.GenerateFixtures(teams)
	if len(fixtures) == 0 {
		return fmt.Errorf("service: cannot regenerate fixtures for %d teams", len(teams))
	}

	return s.repo.WithTx(ctx, func(ctx context.Context, tx repository.LeagueRepository) error {
		if err := tx.DeleteMatchesBySeason(ctx, season.ID); err != nil {
			return err
		}
		if _, err := tx.SetSeasonWeek(ctx, season.ID, 0); err != nil {
			return err
		}
		for _, f := range fixtures {
			if _, err := tx.CreateMatch(ctx, season.ID, f.Week, f.HomeTeamID, f.AwayTeamID); err != nil {
				return err
			}
		}
		return nil
	})
}

// ---- Helpers ---------------------------------------------------------------

// playWeeksLocked simulates the next `weeks` weeks (clamped to remaining).
// Caller must hold s.mu. Returns the flat list of every match played; the
// caller is responsible for slicing it into per-week buckets if needed.
func (s *LeagueService) playWeeksLocked(ctx context.Context, weeks int) (WeekResult, error) {
	season, err := s.GetCurrentSeason(ctx)
	if err != nil {
		return WeekResult{}, err
	}
	if season.Finished() {
		return WeekResult{}, ErrSeasonComplete
	}
	teams, err := s.repo.ListTeams(ctx)
	if err != nil {
		return WeekResult{}, err
	}
	teamByID := indexTeams(teams)

	endWeek := season.CurrentWeek + weeks
	if endWeek > season.TotalWeeks {
		endWeek = season.TotalWeeks
	}

	var played []domain.Match
	err = s.repo.WithTx(ctx, func(ctx context.Context, tx repository.LeagueRepository) error {
		for week := season.CurrentWeek + 1; week <= endWeek; week++ {
			unplayed, err := tx.ListUnplayedMatchesByWeek(ctx, season.ID, week)
			if err != nil {
				return err
			}
			for _, m := range unplayed {
				score := s.simulator.Simulate(s.rng, teamByID[m.HomeTeamID], teamByID[m.AwayTeamID])
				updated, err := tx.RecordMatchResult(ctx, m.ID, score.HomeGoals, score.AwayGoals)
				if err != nil {
					return err
				}
				played = append(played, updated)
			}
		}
		_, err := tx.SetSeasonWeek(ctx, season.ID, endWeek)
		return err
	})
	if err != nil {
		return WeekResult{}, err
	}
	// NextWeek (weeks=1) wants a single-week payload; PlayAll re-buckets.
	if weeks == 1 {
		return WeekResult{Week: season.CurrentWeek + 1, Matches: played}, nil
	}
	return WeekResult{Matches: played}, nil
}

// splitIntoWeeks turns a flat list of played matches into per-week buckets
// in week order. Used by PlayAll which needs the API-shaped grouped result.
func (s *LeagueService) splitIntoWeeks(flat WeekResult) []WeekResult {
	buckets := make(map[int][]domain.Match)
	for _, m := range flat.Matches {
		buckets[m.Week] = append(buckets[m.Week], m)
	}
	// Stable order by week ascending.
	weeks := make([]int, 0, len(buckets))
	for w := range buckets {
		weeks = append(weeks, w)
	}
	sortInts(weeks)
	out := make([]WeekResult, 0, len(weeks))
	for _, w := range weeks {
		out = append(out, WeekResult{Week: w, Matches: buckets[w]})
	}
	return out
}

// snapshotData reads the immutable bits the read endpoints need.
func (s *LeagueService) snapshotData(ctx context.Context) (domain.Season, []domain.Team, []domain.Match, error) {
	season, err := s.GetCurrentSeason(ctx)
	if err != nil {
		return domain.Season{}, nil, nil, err
	}
	teams, err := s.repo.ListTeams(ctx)
	if err != nil {
		return domain.Season{}, nil, nil, err
	}
	matches, err := s.repo.ListMatchesBySeason(ctx, season.ID)
	if err != nil {
		return domain.Season{}, nil, nil, err
	}
	return season, teams, matches, nil
}

func indexTeams(teams []domain.Team) map[int64]domain.Team {
	m := make(map[int64]domain.Team, len(teams))
	for _, t := range teams {
		m[t.ID] = t
	}
	return m
}

func maxWeek(fixtures []standings.FixturePair) int {
	n := 0
	for _, f := range fixtures {
		if f.Week > n {
			n = f.Week
		}
	}
	return n
}

func sortInts(a []int) {
	// Tiny tailored sort for ≤ tens of elements; avoids pulling sort just
	// for this one helper.
	for i := 1; i < len(a); i++ {
		for j := i; j > 0 && a[j-1] > a[j]; j-- {
			a[j-1], a[j] = a[j], a[j-1]
		}
	}
}
