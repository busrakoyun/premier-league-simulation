package domain

import "time"

// Season groups a set of teams playing a round-robin fixture over a fixed
// number of weeks. CurrentWeek is the number of fully completed weeks
// (0 = no matches played, TotalWeeks = season finished).
type Season struct {
	ID          int64
	Name        string
	CurrentWeek int
	TotalWeeks  int
	CreatedAt   time.Time
}

// Finished reports whether all weeks have been played.
func (s Season) Finished() bool {
	return s.CurrentWeek >= s.TotalWeeks
}
