package domain

// StandingRow is a single team's row in the league table.
// Columns mirror Premier League standings: Played, Won, Drawn, Lost,
// GoalsFor, GoalsAgainst, GoalDifference, Points.
type StandingRow struct {
	TeamID         int64
	TeamName       string
	Played         int
	Won            int
	Drawn          int
	Lost           int
	GoalsFor       int
	GoalsAgainst   int
	GoalDifference int
	Points         int
}
