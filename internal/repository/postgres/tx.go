package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/busrakoyun/premier-league-simulation/internal/repository"
)

// runInTx begins a transaction on the pool, runs fn, commits on success, and
// rolls back on error. The deferred Rollback is a no-op after a successful
// Commit, so the happy path costs nothing extra.
func runInTx(ctx context.Context, pool *pgxpool.Pool, fn func(pgx.Tx) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// mapPgError translates pgx / Postgres errors into repository sentinels so
// the service layer can reason in domain terms. Returns nil for nil input.
func mapPgError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return repository.ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505", // unique_violation
			"23503", // foreign_key_violation
			"23514": // check_violation
			return repository.ErrConflict
		}
	}
	return err
}
