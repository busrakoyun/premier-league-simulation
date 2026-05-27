-- Teams ----------------------------------------------------------------------

-- name: ListTeams :many
SELECT * FROM teams ORDER BY id;

-- name: GetTeamByID :one
SELECT * FROM teams WHERE id = $1;

-- Seasons --------------------------------------------------------------------

-- name: CreateSeason :one
INSERT INTO seasons (name, total_weeks) VALUES ($1, $2) RETURNING *;

-- name: GetCurrentSeason :one
SELECT * FROM seasons ORDER BY created_at DESC LIMIT 1;

-- name: SetSeasonWeek :one
UPDATE seasons SET current_week = $2 WHERE id = $1 RETURNING *;

-- name: DeleteSeason :exec
DELETE FROM seasons WHERE id = $1;

-- Matches --------------------------------------------------------------------

-- name: CreateMatch :one
INSERT INTO matches (season_id, week, home_team_id, away_team_id)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListMatchesBySeason :many
SELECT * FROM matches WHERE season_id = $1 ORDER BY week, id;

-- name: ListMatchesByWeek :many
SELECT * FROM matches WHERE season_id = $1 AND week = $2 ORDER BY id;

-- name: ListUnplayedMatchesByWeek :many
SELECT * FROM matches
WHERE season_id = $1 AND week = $2 AND played_at IS NULL
ORDER BY id;

-- name: GetMatchByID :one
SELECT * FROM matches WHERE id = $1;

-- name: RecordMatchResult :one
UPDATE matches
SET home_goals = $2, away_goals = $3, played_at = NOW()
WHERE id = $1 AND played_at IS NULL
RETURNING *;

-- name: UpdateMatchScore :one
UPDATE matches
SET home_goals = $2, away_goals = $3
WHERE id = $1 AND played_at IS NOT NULL
RETURNING *;

-- name: DeleteMatchesBySeason :exec
DELETE FROM matches WHERE season_id = $1;
