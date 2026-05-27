// Package standings holds the pure functions that turn match data into a
// league table and a round-robin fixture list. Both are no-I/O, no-state
// — same inputs always produce the same output — so they're trivially
// testable and reusable across the live next-week path, the edit-match
// path, and the Monte Carlo predictor.
package standings

import (
	"sort"

	"github.com/busrakoyun/premier-league-simulation/internal/domain"
)

// Calculate produces a league table from a set of teams and matches.
//
// Premier League rules: 3 points for a win, 1 for a draw, 0 for a loss.
// Tiebreakers in order: points desc, goal difference desc, goals-for desc,
// team name asc (stable display fallback so the API never returns a
// nondeterministic ordering for two teams that are genuinely tied).
//
// Unplayed matches are skipped. Every team appears in the output (with
// zeros if they haven't played) so the table always has one row per club —
// the frontend can render the table from day zero.
func Calculate(teams []domain.Team, matches []domain.Match) []domain.StandingRow {
	rows := make(map[int64]*domain.StandingRow, len(teams))
	for _, t := range teams {
		rows[t.ID] = &domain.StandingRow{
			TeamID:   t.ID,
			TeamName: t.Name,
		}
	}

	for _, m := range matches {
		if !m.Played() {
			continue
		}
		home, okH := rows[m.HomeTeamID]
		away, okA := rows[m.AwayTeamID]
		if !okH || !okA {
			// A match referencing a team we weren't given is silently
			// dropped — the caller asked for the table over `teams`,
			// not over whoever happens to be in `matches`.
			continue
		}
		applyResult(home, away, m.HomeGoals, m.AwayGoals)
	}

	out := make([]domain.StandingRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, *r)
	}
	sortStandings(out)
	return out
}

func applyResult(home, away *domain.StandingRow, homeGoals, awayGoals int) {
	home.Played++
	away.Played++
	home.GoalsFor += homeGoals
	home.GoalsAgainst += awayGoals
	away.GoalsFor += awayGoals
	away.GoalsAgainst += homeGoals
	home.GoalDifference = home.GoalsFor - home.GoalsAgainst
	away.GoalDifference = away.GoalsFor - away.GoalsAgainst

	switch {
	case homeGoals > awayGoals:
		home.Won++
		home.Points += 3
		away.Lost++
	case homeGoals < awayGoals:
		away.Won++
		away.Points += 3
		home.Lost++
	default:
		home.Drawn++
		away.Drawn++
		home.Points++
		away.Points++
	}
}

func sortStandings(rows []domain.StandingRow) {
	sort.SliceStable(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		switch {
		case a.Points != b.Points:
			return a.Points > b.Points
		case a.GoalDifference != b.GoalDifference:
			return a.GoalDifference > b.GoalDifference
		case a.GoalsFor != b.GoalsFor:
			return a.GoalsFor > b.GoalsFor
		default:
			return a.TeamName < b.TeamName
		}
	})
}
