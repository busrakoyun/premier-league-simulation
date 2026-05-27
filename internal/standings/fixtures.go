package standings

import (
	"github.com/busrakoyun/premier-league-simulation/internal/domain"
)

// FixturePair is a single (home, away) match assignment for a given week,
// returned by GenerateFixtures before the matches exist in the database.
type FixturePair struct {
	Week       int
	HomeTeamID int64
	AwayTeamID int64
}

// GenerateFixtures produces a double round-robin schedule for the given
// teams. Each pair of teams meets twice — once at each venue — for a total
// of N×(N−1) matches over 2×(N−1) weeks. For the PDF case (N=4): 12 matches
// across 6 weeks, 2 matches per week.
//
// Algorithm: the "circle method" (a.k.a. Berger table). Fix team[0], rotate
// team[1:] one position per round; pair up positions from the ends inward.
// First half goes (home=a, away=b); the second half repeats the same
// pairings but flips sides, giving every team an equal home/away split.
//
// Returns nil for inputs that don't fit a clean round-robin (fewer than two
// teams or an odd team count) — odd N would need a "bye" team and falls
// outside the PDF's scope.
func GenerateFixtures(teams []domain.Team) []FixturePair {
	n := len(teams)
	if n < 2 || n%2 != 0 {
		return nil
	}

	ids := make([]int64, n)
	for i, t := range teams {
		ids[i] = t.ID
	}

	weeksPerHalf := n - 1
	out := make([]FixturePair, 0, n*(n-1))

	for half := 0; half < 2; half++ {
		// Reset the rotation each half so the pairing pattern is identical;
		// the half flag decides which side hosts.
		positions := make([]int64, n)
		copy(positions, ids)

		for round := 0; round < weeksPerHalf; round++ {
			week := half*weeksPerHalf + round + 1
			for i := 0; i < n/2; i++ {
				a, b := positions[i], positions[n-1-i]
				home, away := a, b
				if half == 1 {
					home, away = b, a
				}
				out = append(out, FixturePair{
					Week:       week,
					HomeTeamID: home,
					AwayTeamID: away,
				})
			}
			// Rotate positions[1:] one step "right" while keeping positions[0]
			// fixed: take the last element, shift the middle right, drop the
			// taken element into slot 1.
			last := positions[n-1]
			copy(positions[2:], positions[1:n-1])
			positions[1] = last
		}
	}
	return out
}
