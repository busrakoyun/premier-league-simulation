// API response types. These mirror the Go DTOs in internal/http/dto.go
// one-to-one — keep them in sync if the backend shape changes.

export interface Team {
  id: number
  name: string
  short_name: string
  attack_strength: number
  defense_strength: number
  home_advantage: number
}

export interface Season {
  id: number
  name: string
  current_week: number
  total_weeks: number
  created_at: string
}

export interface Score {
  home_goals: number
  away_goals: number
  played_at: string
}

export interface Match {
  id: number
  season_id: number
  week: number
  home_team_id: number
  away_team_id: number
  home_team_name: string
  away_team_name: string
  played: boolean
  score?: Score
}

export interface StandingRow {
  team_id: number
  team_name: string
  played: number
  won: number
  drawn: number
  lost: number
  goals_for: number
  goals_against: number
  goal_difference: number
  points: number
}

export interface PredictionRow {
  team_id: number
  team_name: string
  championship_rate: number
}

export interface WeekResult {
  week: number
  matches: Match[]
}

export interface ApiError {
  error: string
  code: string
}

// Response envelopes ---------------------------------------------------------

export interface TeamsResponse {
  teams: Team[]
}
export interface MatchesResponse {
  matches: Match[]
}
export interface StandingsResponse {
  rows: StandingRow[]
}
export interface PredictionsResponse {
  rows: PredictionRow[]
}
export interface NextWeekResponse {
  week: number
  matches: Match[]
}
export interface PlayAllResponse {
  weeks: WeekResult[]
}
