package standings_test

import (
	"testing"

	"github.com/busrakoyun/premier-league-simulation/internal/domain"
	"github.com/busrakoyun/premier-league-simulation/internal/standings"
)

func TestGenerateFixtures_FourTeamsHasExpectedShape(t *testing.T) {
	teams := []domain.Team{
		{ID: 1, Name: "A"},
		{ID: 2, Name: "B"},
		{ID: 3, Name: "C"},
		{ID: 4, Name: "D"},
	}
	fixtures := standings.GenerateFixtures(teams)

	if len(fixtures) != 12 {
		t.Fatalf("got %d matches, want 12 (4 teams × 3 opponents × 2 legs)", len(fixtures))
	}

	// 6 weeks, 2 matches each.
	weeks := make(map[int]int)
	for _, f := range fixtures {
		weeks[f.Week]++
	}
	if len(weeks) != 6 {
		t.Errorf("got %d distinct weeks, want 6", len(weeks))
	}
	for w := 1; w <= 6; w++ {
		if weeks[w] != 2 {
			t.Errorf("week %d has %d matches, want 2", w, weeks[w])
		}
	}

	// Each unordered pair meets exactly twice.
	type pair struct{ a, b int64 }
	norm := func(x, y int64) pair {
		if x < y {
			return pair{x, y}
		}
		return pair{y, x}
	}
	pairCounts := make(map[pair]int)
	for _, f := range fixtures {
		pairCounts[norm(f.HomeTeamID, f.AwayTeamID)]++
	}
	if len(pairCounts) != 6 { // C(4,2)
		t.Errorf("got %d distinct pairs, want 6", len(pairCounts))
	}
	for p, c := range pairCounts {
		if c != 2 {
			t.Errorf("pair %v plays %d times, want 2", p, c)
		}
	}

	// Each directed pair (home, away) appears exactly once → balanced home/away.
	type directed struct{ home, away int64 }
	dirCounts := make(map[directed]int)
	for _, f := range fixtures {
		dirCounts[directed{f.HomeTeamID, f.AwayTeamID}]++
	}
	for d, c := range dirCounts {
		if c != 1 {
			t.Errorf("directed match %v plays %d times, want 1", d, c)
		}
	}

	// No self-matches.
	for _, f := range fixtures {
		if f.HomeTeamID == f.AwayTeamID {
			t.Errorf("self-match in fixture: %+v", f)
		}
	}

	// Each team plays at most once per week.
	for w := 1; w <= 6; w++ {
		seen := make(map[int64]bool)
		for _, f := range fixtures {
			if f.Week != w {
				continue
			}
			if seen[f.HomeTeamID] || seen[f.AwayTeamID] {
				t.Errorf("week %d has a team playing twice: %+v", w, f)
			}
			seen[f.HomeTeamID] = true
			seen[f.AwayTeamID] = true
		}
	}

	// Each team plays exactly 3 home games and 3 away games.
	homeCount := make(map[int64]int)
	awayCount := make(map[int64]int)
	for _, f := range fixtures {
		homeCount[f.HomeTeamID]++
		awayCount[f.AwayTeamID]++
	}
	for _, tm := range teams {
		if homeCount[tm.ID] != 3 || awayCount[tm.ID] != 3 {
			t.Errorf("team %d: home=%d away=%d, want 3/3",
				tm.ID, homeCount[tm.ID], awayCount[tm.ID])
		}
	}
}

func TestGenerateFixtures_RejectsBadShapes(t *testing.T) {
	if got := standings.GenerateFixtures(nil); got != nil {
		t.Errorf("nil teams: got %v, want nil", got)
	}
	if got := standings.GenerateFixtures([]domain.Team{{ID: 1}}); got != nil {
		t.Errorf("1 team: got %v, want nil", got)
	}
	if got := standings.GenerateFixtures([]domain.Team{{ID: 1}, {ID: 2}, {ID: 3}}); got != nil {
		t.Errorf("3 teams (odd): got %v, want nil (would need a bye)", got)
	}
}
