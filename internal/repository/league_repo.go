// Package repository defines the persistence boundary for the league domain.
// Implementations keep storage details out of the rest of the codebase —
// services depend on the LeagueRepository interface, never on a concrete DB.
package repository

import (
	"context"
	"errors"

	"github.com/busrakoyun/premier-league-simulation/internal/domain"
)

// Sentinel errors that callers match with errors.Is to react in domain terms
// rather than driver specifics.
var (
	ErrNotFound = errors.New("repository: not found")
	ErrConflict = errors.New("repository: constraint conflict")
)

// LeagueRepository captures every operation the league service needs against
// the underlying store. Keeping this surface narrow means the in-memory mock
// used in service tests stays small, and a future Postgres swap (or read
// replica split) doesn't ripple into business logic.
type LeagueRepository interface {
	// Teams
	ListTeams(ctx context.Context) ([]domain.Team, error)

	// Seasons
	CreateSeason(ctx context.Context, name string, totalWeeks int) (domain.Season, error)
	GetCurrentSeason(ctx context.Context) (domain.Season, error)
	SetSeasonWeek(ctx context.Context, seasonID int64, currentWeek int) (domain.Season, error)
	DeleteSeason(ctx context.Context, seasonID int64) error

	// Matches
	CreateMatch(ctx context.Context, seasonID int64, week int, homeTeamID, awayTeamID int64) (domain.Match, error)
	ListMatchesBySeason(ctx context.Context, seasonID int64) ([]domain.Match, error)
	ListMatchesByWeek(ctx context.Context, seasonID int64, week int) ([]domain.Match, error)
	ListUnplayedMatchesByWeek(ctx context.Context, seasonID int64, week int) ([]domain.Match, error)
	GetMatchByID(ctx context.Context, matchID int64) (domain.Match, error)
	RecordMatchResult(ctx context.Context, matchID int64, homeGoals, awayGoals int) (domain.Match, error)
	UpdateMatchScore(ctx context.Context, matchID int64, homeGoals, awayGoals int) (domain.Match, error)
	DeleteMatchesBySeason(ctx context.Context, seasonID int64) error

	// WithTx runs fn inside a single database transaction. The repo handed to
	// fn is bound to that tx; all writes commit together on a nil return or
	// roll back on error. Calling WithTx on an already-tx-bound repo reuses
	// the existing tx rather than nesting — we don't need savepoint semantics
	// at this scale, and reusing keeps callers from worrying about depth.
	WithTx(ctx context.Context, fn func(ctx context.Context, repo LeagueRepository) error) error
}
