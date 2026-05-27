package domain

import "time"

// MatchResult captures the outcome of a played match.
type MatchResult struct {
	HomeGoals int
	AwayGoals int
	PlayedAt  time.Time
}

// MatchScore is the goal count for a single match — the value the simulator
// produces before the service stamps a played-at timestamp on it. Kept
// distinct from MatchResult so the simulator stays stateless w.r.t. clocks
// (the small duplication of two int fields is the price; not worth embedding
// MatchScore inside MatchResult given how often each is constructed by hand).
type MatchScore struct {
	HomeGoals int
	AwayGoals int
}

// Match is a fixture between two teams in a given week of a season.
//
// Composition: embedding *MatchResult lets a played match expose HomeGoals
// and AwayGoals as if they were direct fields, while a nil pointer makes the
// "unplayed" state explicit. Use the Played() predicate at call sites rather
// than the bare nil check so the contract is readable.
type Match struct {
	ID         int64
	SeasonID   int64
	Week       int
	HomeTeamID int64
	AwayTeamID int64
	*MatchResult
}

// Played reports whether the match has been played.
func (m Match) Played() bool {
	return m.MatchResult != nil
}
