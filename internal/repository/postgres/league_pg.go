// Package postgres implements repository.LeagueRepository against PostgreSQL,
// wrapping sqlc-generated query code. The rest of the codebase only sees
// domain types — DB nullability and pgx-specific types stop at this package.
package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/busrakoyun/premier-league-simulation/internal/domain"
	"github.com/busrakoyun/premier-league-simulation/internal/repository"
	"github.com/busrakoyun/premier-league-simulation/internal/repository/sqlc"
)

// Compile-time assertion: LeagueRepo must satisfy repository.LeagueRepository.
var _ repository.LeagueRepository = (*LeagueRepo)(nil)

// LeagueRepo is the Postgres-backed implementation.
//
// When pool != nil the repo is bound to the pool and WithTx can start a fresh
// transaction. When pool == nil the repo is the inner repo handed to a WithTx
// callback — already bound to a tx via queries — and WithTx becomes a no-op
// that reuses the current tx (see interface doc for rationale).
type LeagueRepo struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

// New creates a LeagueRepo backed by the given pgx connection pool.
func New(pool *pgxpool.Pool) *LeagueRepo {
	return &LeagueRepo{
		pool:    pool,
		queries: sqlc.New(pool),
	}
}

func (r *LeagueRepo) WithTx(ctx context.Context, fn func(ctx context.Context, repo repository.LeagueRepository) error) error {
	if r.pool == nil {
		return fn(ctx, r)
	}
	return runInTx(ctx, r.pool, func(tx pgx.Tx) error {
		txRepo := &LeagueRepo{queries: sqlc.New(tx)}
		return fn(ctx, txRepo)
	})
}

// ---- Teams -----------------------------------------------------------------

func (r *LeagueRepo) ListTeams(ctx context.Context) ([]domain.Team, error) {
	rows, err := r.queries.ListTeams(ctx)
	if err != nil {
		return nil, mapPgError(err)
	}
	out := make([]domain.Team, len(rows))
	for i, row := range rows {
		out[i] = toDomainTeam(row)
	}
	return out, nil
}

// ---- Seasons ---------------------------------------------------------------

func (r *LeagueRepo) CreateSeason(ctx context.Context, name string, totalWeeks int) (domain.Season, error) {
	row, err := r.queries.CreateSeason(ctx, sqlc.CreateSeasonParams{
		Name:       name,
		TotalWeeks: int32(totalWeeks),
	})
	if err != nil {
		return domain.Season{}, mapPgError(err)
	}
	return toDomainSeason(row), nil
}

func (r *LeagueRepo) GetCurrentSeason(ctx context.Context) (domain.Season, error) {
	row, err := r.queries.GetCurrentSeason(ctx)
	if err != nil {
		return domain.Season{}, mapPgError(err)
	}
	return toDomainSeason(row), nil
}

func (r *LeagueRepo) SetSeasonWeek(ctx context.Context, seasonID int64, currentWeek int) (domain.Season, error) {
	row, err := r.queries.SetSeasonWeek(ctx, sqlc.SetSeasonWeekParams{
		ID:          seasonID,
		CurrentWeek: int32(currentWeek),
	})
	if err != nil {
		return domain.Season{}, mapPgError(err)
	}
	return toDomainSeason(row), nil
}

func (r *LeagueRepo) DeleteSeason(ctx context.Context, seasonID int64) error {
	return mapPgError(r.queries.DeleteSeason(ctx, seasonID))
}

// ---- Matches ---------------------------------------------------------------

func (r *LeagueRepo) CreateMatch(ctx context.Context, seasonID int64, week int, homeTeamID, awayTeamID int64) (domain.Match, error) {
	row, err := r.queries.CreateMatch(ctx, sqlc.CreateMatchParams{
		SeasonID:   seasonID,
		Week:       int32(week),
		HomeTeamID: homeTeamID,
		AwayTeamID: awayTeamID,
	})
	if err != nil {
		return domain.Match{}, mapPgError(err)
	}
	return toDomainMatch(row), nil
}

func (r *LeagueRepo) ListMatchesBySeason(ctx context.Context, seasonID int64) ([]domain.Match, error) {
	rows, err := r.queries.ListMatchesBySeason(ctx, seasonID)
	if err != nil {
		return nil, mapPgError(err)
	}
	out := make([]domain.Match, len(rows))
	for i, row := range rows {
		out[i] = toDomainMatch(row)
	}
	return out, nil
}

func (r *LeagueRepo) ListMatchesByWeek(ctx context.Context, seasonID int64, week int) ([]domain.Match, error) {
	rows, err := r.queries.ListMatchesByWeek(ctx, sqlc.ListMatchesByWeekParams{
		SeasonID: seasonID,
		Week:     int32(week),
	})
	if err != nil {
		return nil, mapPgError(err)
	}
	out := make([]domain.Match, len(rows))
	for i, row := range rows {
		out[i] = toDomainMatch(row)
	}
	return out, nil
}

func (r *LeagueRepo) ListUnplayedMatchesByWeek(ctx context.Context, seasonID int64, week int) ([]domain.Match, error) {
	rows, err := r.queries.ListUnplayedMatchesByWeek(ctx, sqlc.ListUnplayedMatchesByWeekParams{
		SeasonID: seasonID,
		Week:     int32(week),
	})
	if err != nil {
		return nil, mapPgError(err)
	}
	out := make([]domain.Match, len(rows))
	for i, row := range rows {
		out[i] = toDomainMatch(row)
	}
	return out, nil
}

func (r *LeagueRepo) GetMatchByID(ctx context.Context, matchID int64) (domain.Match, error) {
	row, err := r.queries.GetMatchByID(ctx, matchID)
	if err != nil {
		return domain.Match{}, mapPgError(err)
	}
	return toDomainMatch(row), nil
}

func (r *LeagueRepo) RecordMatchResult(ctx context.Context, matchID int64, homeGoals, awayGoals int) (domain.Match, error) {
	hg, ag := int32(homeGoals), int32(awayGoals)
	row, err := r.queries.RecordMatchResult(ctx, sqlc.RecordMatchResultParams{
		ID:        matchID,
		HomeGoals: &hg,
		AwayGoals: &ag,
	})
	if err != nil {
		return domain.Match{}, mapPgError(err)
	}
	return toDomainMatch(row), nil
}

func (r *LeagueRepo) UpdateMatchScore(ctx context.Context, matchID int64, homeGoals, awayGoals int) (domain.Match, error) {
	hg, ag := int32(homeGoals), int32(awayGoals)
	row, err := r.queries.UpdateMatchScore(ctx, sqlc.UpdateMatchScoreParams{
		ID:        matchID,
		HomeGoals: &hg,
		AwayGoals: &ag,
	})
	if err != nil {
		return domain.Match{}, mapPgError(err)
	}
	return toDomainMatch(row), nil
}

func (r *LeagueRepo) DeleteMatchesBySeason(ctx context.Context, seasonID int64) error {
	return mapPgError(r.queries.DeleteMatchesBySeason(ctx, seasonID))
}

// ---- Mappers (sqlc row -> domain) ------------------------------------------

func toDomainTeam(t sqlc.Team) domain.Team {
	return domain.Team{
		ID:              t.ID,
		Name:            t.Name,
		ShortName:       t.ShortName,
		AttackStrength:  t.AttackStrength,
		DefenseStrength: t.DefenseStrength,
		HomeAdvantage:   t.HomeAdvantage,
	}
}

func toDomainSeason(s sqlc.Season) domain.Season {
	return domain.Season{
		ID:          s.ID,
		Name:        s.Name,
		CurrentWeek: int(s.CurrentWeek),
		TotalWeeks:  int(s.TotalWeeks),
		CreatedAt:   s.CreatedAt.Time,
	}
}

func toDomainMatch(m sqlc.Match) domain.Match {
	match := domain.Match{
		ID:         m.ID,
		SeasonID:   m.SeasonID,
		Week:       int(m.Week),
		HomeTeamID: m.HomeTeamID,
		AwayTeamID: m.AwayTeamID,
	}
	// The DB CHECK constraint guarantees these three fields are all-or-nothing,
	// so a played match has all three set and an unplayed match has none.
	if m.HomeGoals != nil && m.AwayGoals != nil && m.PlayedAt.Valid {
		match.MatchResult = &domain.MatchResult{
			HomeGoals: int(*m.HomeGoals),
			AwayGoals: int(*m.AwayGoals),
			PlayedAt:  m.PlayedAt.Time,
		}
	}
	return match
}
