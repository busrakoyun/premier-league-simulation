CREATE TABLE teams (
    id               BIGSERIAL PRIMARY KEY,
    name             TEXT NOT NULL UNIQUE,
    short_name       TEXT NOT NULL UNIQUE,
    attack_strength  DOUBLE PRECISION NOT NULL CHECK (attack_strength > 0),
    defense_strength DOUBLE PRECISION NOT NULL CHECK (defense_strength > 0),
    home_advantage   DOUBLE PRECISION NOT NULL DEFAULT 1.3 CHECK (home_advantage > 0)
);

CREATE TABLE seasons (
    id           BIGSERIAL PRIMARY KEY,
    name         TEXT NOT NULL,
    current_week INTEGER NOT NULL DEFAULT 0 CHECK (current_week >= 0),
    total_weeks  INTEGER NOT NULL CHECK (total_weeks > 0),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE matches (
    id           BIGSERIAL PRIMARY KEY,
    season_id    BIGINT NOT NULL REFERENCES seasons(id) ON DELETE CASCADE,
    week         INTEGER NOT NULL CHECK (week > 0),
    home_team_id BIGINT NOT NULL REFERENCES teams(id),
    away_team_id BIGINT NOT NULL REFERENCES teams(id),
    home_goals   INTEGER CHECK (home_goals IS NULL OR home_goals >= 0),
    away_goals   INTEGER CHECK (away_goals IS NULL OR away_goals >= 0),
    played_at    TIMESTAMPTZ,
    CHECK (home_team_id <> away_team_id),
    CHECK (
        (home_goals IS NULL AND away_goals IS NULL AND played_at IS NULL)
        OR
        (home_goals IS NOT NULL AND away_goals IS NOT NULL AND played_at IS NOT NULL)
    ),
    UNIQUE (season_id, week, home_team_id, away_team_id)
);

CREATE INDEX idx_matches_season_week ON matches (season_id, week);
