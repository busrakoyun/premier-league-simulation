package domain

// PredictionRow is one team's estimated championship probability.
// ChampionshipRate is in [0, 1]; rates across all teams sum to 1.0
// (ties for first share the championship equally per Monte Carlo iteration).
type PredictionRow struct {
	TeamID           int64
	TeamName         string
	ChampionshipRate float64
}

// SeasonSnapshot is the immutable input handed to a PredictionEngine —
// enough information to simulate the remaining season from scratch.
type SeasonSnapshot struct {
	Teams       []Team
	Matches     []Match
	CurrentWeek int
	TotalWeeks  int
}
