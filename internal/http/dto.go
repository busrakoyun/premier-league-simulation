package http

import (
	"time"

	"github.com/busrakoyun/premier-league-simulation/internal/domain"
	"github.com/busrakoyun/premier-league-simulation/internal/service"
)

// ---- DTO types -------------------------------------------------------------

// TeamDTO exposes a team's identity and the strength parameters that drive
// the simulator. Strengths are included so the frontend can show "why" the
// simulator favours one side — full transparency.
type TeamDTO struct {
	ID              int64   `json:"id"`
	Name            string  `json:"name"`
	ShortName       string  `json:"short_name"`
	AttackStrength  float64 `json:"attack_strength"`
	DefenseStrength float64 `json:"defense_strength"`
	HomeAdvantage   float64 `json:"home_advantage"`
}

type SeasonDTO struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	CurrentWeek int    `json:"current_week"`
	TotalWeeks  int    `json:"total_weeks"`
	CreatedAt   string `json:"created_at"`
}

// ScoreDTO is omitted on unplayed matches (see MatchDTO.Score).
type ScoreDTO struct {
	HomeGoals int    `json:"home_goals"`
	AwayGoals int    `json:"away_goals"`
	PlayedAt  string `json:"played_at"`
}

type MatchDTO struct {
	ID           int64     `json:"id"`
	SeasonID     int64     `json:"season_id"`
	Week         int       `json:"week"`
	HomeTeamID   int64     `json:"home_team_id"`
	AwayTeamID   int64     `json:"away_team_id"`
	HomeTeamName string    `json:"home_team_name"`
	AwayTeamName string    `json:"away_team_name"`
	Played       bool      `json:"played"`
	Score        *ScoreDTO `json:"score,omitempty"`
}

type StandingDTO struct {
	TeamID         int64  `json:"team_id"`
	TeamName       string `json:"team_name"`
	Played         int    `json:"played"`
	Won            int    `json:"won"`
	Drawn          int    `json:"drawn"`
	Lost           int    `json:"lost"`
	GoalsFor       int    `json:"goals_for"`
	GoalsAgainst   int    `json:"goals_against"`
	GoalDifference int    `json:"goal_difference"`
	Points         int    `json:"points"`
}

type PredictionDTO struct {
	TeamID           int64   `json:"team_id"`
	TeamName         string  `json:"team_name"`
	ChampionshipRate float64 `json:"championship_rate"`
}

type WeekDTO struct {
	Week    int        `json:"week"`
	Matches []MatchDTO `json:"matches"`
}

// ---- Response envelopes ----------------------------------------------------

type TeamsResponse struct {
	Teams []TeamDTO `json:"teams"`
}
type MatchesResponse struct {
	Matches []MatchDTO `json:"matches"`
}
type StandingsResponse struct {
	Rows []StandingDTO `json:"rows"`
}
type PredictionsResponse struct {
	Rows []PredictionDTO `json:"rows"`
}
type NextWeekResponse struct {
	Week    int        `json:"week"`
	Matches []MatchDTO `json:"matches"`
}
type PlayAllResponse struct {
	Weeks []WeekDTO `json:"weeks"`
}

// ---- Request envelopes -----------------------------------------------------

type EditMatchRequest struct {
	HomeGoals int `json:"home_goals"`
	AwayGoals int `json:"away_goals"`
}

// ---- Converters ------------------------------------------------------------
//
// All domain → DTO mapping lives here so the handlers stay thin. Team names
// for matches are looked up against a caller-supplied map so the handler
// does the (cheap) ListTeams once per request and reuses it.

func toTeamDTOs(teams []domain.Team) []TeamDTO {
	out := make([]TeamDTO, len(teams))
	for i, t := range teams {
		out[i] = toTeamDTO(t)
	}
	return out
}

func toTeamDTO(t domain.Team) TeamDTO {
	return TeamDTO{
		ID:              t.ID,
		Name:            t.Name,
		ShortName:       t.ShortName,
		AttackStrength:  t.AttackStrength,
		DefenseStrength: t.DefenseStrength,
		HomeAdvantage:   t.HomeAdvantage,
	}
}

func toSeasonDTO(s domain.Season) SeasonDTO {
	return SeasonDTO{
		ID:          s.ID,
		Name:        s.Name,
		CurrentWeek: s.CurrentWeek,
		TotalWeeks:  s.TotalWeeks,
		CreatedAt:   s.CreatedAt.Format(time.RFC3339),
	}
}

func toMatchDTO(m domain.Match, teamByID map[int64]domain.Team) MatchDTO {
	dto := MatchDTO{
		ID:           m.ID,
		SeasonID:     m.SeasonID,
		Week:         m.Week,
		HomeTeamID:   m.HomeTeamID,
		AwayTeamID:   m.AwayTeamID,
		HomeTeamName: teamByID[m.HomeTeamID].Name,
		AwayTeamName: teamByID[m.AwayTeamID].Name,
		Played:       m.Played(),
	}
	if m.Played() {
		dto.Score = &ScoreDTO{
			HomeGoals: m.HomeGoals,
			AwayGoals: m.AwayGoals,
			PlayedAt:  m.PlayedAt.Format(time.RFC3339),
		}
	}
	return dto
}

func toMatchDTOs(matches []domain.Match, teams []domain.Team) []MatchDTO {
	teamByID := indexTeams(teams)
	out := make([]MatchDTO, len(matches))
	for i, m := range matches {
		out[i] = toMatchDTO(m, teamByID)
	}
	return out
}

func toStandingDTOs(rows []domain.StandingRow) []StandingDTO {
	out := make([]StandingDTO, len(rows))
	for i, r := range rows {
		out[i] = StandingDTO{
			TeamID:         r.TeamID,
			TeamName:       r.TeamName,
			Played:         r.Played,
			Won:            r.Won,
			Drawn:          r.Drawn,
			Lost:           r.Lost,
			GoalsFor:       r.GoalsFor,
			GoalsAgainst:   r.GoalsAgainst,
			GoalDifference: r.GoalDifference,
			Points:         r.Points,
		}
	}
	return out
}

func toPredictionDTOs(rows []domain.PredictionRow) []PredictionDTO {
	out := make([]PredictionDTO, len(rows))
	for i, r := range rows {
		out[i] = PredictionDTO{
			TeamID:           r.TeamID,
			TeamName:         r.TeamName,
			ChampionshipRate: r.ChampionshipRate,
		}
	}
	return out
}

func toWeekDTOs(weeks []service.WeekResult, teams []domain.Team) []WeekDTO {
	teamByID := indexTeams(teams)
	out := make([]WeekDTO, len(weeks))
	for i, w := range weeks {
		matches := make([]MatchDTO, len(w.Matches))
		for j, m := range w.Matches {
			matches[j] = toMatchDTO(m, teamByID)
		}
		out[i] = WeekDTO{Week: w.Week, Matches: matches}
	}
	return out
}

func indexTeams(teams []domain.Team) map[int64]domain.Team {
	m := make(map[int64]domain.Team, len(teams))
	for _, t := range teams {
		m[t.ID] = t
	}
	return m
}
