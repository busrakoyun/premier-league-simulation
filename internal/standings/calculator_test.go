package standings_test

import (
	"testing"
	"time"

	"github.com/busrakoyun/premier-league-simulation/internal/domain"
	"github.com/busrakoyun/premier-league-simulation/internal/standings"
)

var (
	chelsea = domain.Team{ID: 1, Name: "Chelsea"}
	arsenal = domain.Team{ID: 2, Name: "Arsenal"}
	mancity = domain.Team{ID: 3, Name: "Manchester City"}
	liverp  = domain.Team{ID: 4, Name: "Liverpool"}
	teams   = []domain.Team{chelsea, arsenal, mancity, liverp}
)

// played builds a played-match value. Test fixtures are intentionally terse —
// the table-driven cases below are easier to scan when each row is a single
// line.
func played(id int64, week int, home, away int64, hg, ag int) domain.Match {
	return domain.Match{
		ID: id, SeasonID: 1, Week: week,
		HomeTeamID: home, AwayTeamID: away,
		MatchResult: &domain.MatchResult{HomeGoals: hg, AwayGoals: ag, PlayedAt: time.Now()},
	}
}

func find(rows []domain.StandingRow, name string) domain.StandingRow {
	for _, r := range rows {
		if r.TeamName == name {
			return r
		}
	}
	return domain.StandingRow{}
}

func TestCalculate_EmptyMatchesProducesZeroRows(t *testing.T) {
	rows := standings.Calculate(teams, nil)
	if len(rows) != 4 {
		t.Fatalf("got %d rows, want 4", len(rows))
	}
	for _, r := range rows {
		if r.Played != 0 || r.Points != 0 || r.GoalsFor != 0 || r.GoalsAgainst != 0 {
			t.Errorf("team %q has non-zero stats with no matches: %+v", r.TeamName, r)
		}
	}
}

func TestCalculate_PointsAfterMixedResults(t *testing.T) {
	matches := []domain.Match{
		played(1, 1, chelsea.ID, arsenal.ID, 2, 1), // Chelsea wins
		played(2, 1, mancity.ID, liverp.ID, 1, 1),  // draw
		played(3, 2, arsenal.ID, mancity.ID, 0, 3), // Man City wins big
	}
	rows := standings.Calculate(teams, matches)

	// Expected:
	//   Manchester City: 1W 1D 0L, 4 pts, GD +3, GF 4
	//   Chelsea:         1W 0D 0L, 3 pts, GD +1, GF 2
	//   Liverpool:       0W 1D 0L, 1 pt,  GD 0,  GF 1
	//   Arsenal:         0W 0D 2L, 0 pts, GD -4, GF 1
	wantOrder := []string{"Manchester City", "Chelsea", "Liverpool", "Arsenal"}
	for i, want := range wantOrder {
		if rows[i].TeamName != want {
			t.Errorf("rank %d: got %q, want %q", i+1, rows[i].TeamName, want)
		}
	}

	mc := find(rows, "Manchester City")
	if mc.Points != 4 || mc.Won != 1 || mc.Drawn != 1 || mc.Lost != 0 || mc.GoalDifference != 3 || mc.GoalsFor != 4 {
		t.Errorf("Man City row: %+v", mc)
	}
	ch := find(rows, "Chelsea")
	if ch.Points != 3 || ch.Won != 1 || ch.GoalsFor != 2 || ch.GoalsAgainst != 1 {
		t.Errorf("Chelsea row: %+v", ch)
	}
	ars := find(rows, "Arsenal")
	if ars.Points != 0 || ars.Lost != 2 || ars.GoalDifference != -4 {
		t.Errorf("Arsenal row: %+v", ars)
	}
}

func TestCalculate_TiebreakerByGoalDifference(t *testing.T) {
	// Both Chelsea and Arsenal pick up 3 points; Chelsea has the better GD.
	matches := []domain.Match{
		played(1, 1, chelsea.ID, liverp.ID, 3, 0),
		played(2, 1, arsenal.ID, mancity.ID, 1, 0),
	}
	rows := standings.Calculate(teams, matches)
	if rows[0].TeamName != "Chelsea" || rows[1].TeamName != "Arsenal" {
		t.Errorf("expected Chelsea > Arsenal by GD, got %q > %q",
			rows[0].TeamName, rows[1].TeamName)
	}
}

func TestCalculate_TiebreakerByGoalsForWhenGDEqual(t *testing.T) {
	// Both winners on 3 pts with GD=+2; Chelsea scored more (3 vs 2).
	matches := []domain.Match{
		played(1, 1, chelsea.ID, arsenal.ID, 3, 1),
		played(2, 1, mancity.ID, liverp.ID, 2, 0),
	}
	rows := standings.Calculate(teams, matches)
	if rows[0].TeamName != "Chelsea" {
		t.Errorf("expected Chelsea first by GF, got %q (rows=%+v)", rows[0].TeamName, rows)
	}
}

func TestCalculate_NameAsFinalTiebreakerIsStable(t *testing.T) {
	// Identical stats across two teams; name asc is the deterministic tiebreaker.
	matches := []domain.Match{
		played(1, 1, chelsea.ID, liverp.ID, 1, 0),
		played(2, 1, arsenal.ID, mancity.ID, 1, 0),
	}
	rows := standings.Calculate(teams, matches)
	// Arsenal and Chelsea both 3 pts/+1 GD/GF 1 — Arsenal (alphabetical) comes first.
	if rows[0].TeamName != "Arsenal" || rows[1].TeamName != "Chelsea" {
		t.Errorf("expected Arsenal > Chelsea by name, got %q > %q",
			rows[0].TeamName, rows[1].TeamName)
	}
}

func TestCalculate_UnplayedMatchesIgnored(t *testing.T) {
	matches := []domain.Match{
		played(1, 1, chelsea.ID, arsenal.ID, 2, 0),
		// Unplayed: MatchResult zero value (nil pointer) — Played() = false.
		{ID: 2, SeasonID: 1, Week: 2, HomeTeamID: mancity.ID, AwayTeamID: liverp.ID},
	}
	rows := standings.Calculate(teams, matches)
	if mc := find(rows, "Manchester City"); mc.Played != 0 {
		t.Errorf("Manchester City should not have a played match: %+v", mc)
	}
	if ch := find(rows, "Chelsea"); ch.Played != 1 || ch.Won != 1 {
		t.Errorf("Chelsea should have 1 played + 1 win: %+v", ch)
	}
}

func TestCalculate_MatchWithUnknownTeamIsDropped(t *testing.T) {
	// Team 99 isn't in the teams slice — its match must not affect anything.
	matches := []domain.Match{
		played(1, 1, chelsea.ID, 99, 5, 0),
	}
	rows := standings.Calculate(teams, matches)
	for _, r := range rows {
		if r.Played != 0 {
			t.Errorf("match with unknown team leaked: %+v", r)
		}
	}
}
