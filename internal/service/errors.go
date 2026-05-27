package service

import "errors"

// Service-level errors that the HTTP layer maps to specific status codes.
// Callers should match with errors.Is.
var (
	// ErrNoCurrentSeason is returned when an operation needs an active
	// season but none exists yet (startup state before EnsureSeason runs).
	ErrNoCurrentSeason = errors.New("service: no current season")

	// ErrSeasonComplete is returned when next-week / play-all are invoked
	// after every fixture has been played.
	ErrSeasonComplete = errors.New("service: season is complete")

	// ErrPredictionsNotAvailable is returned by Predict before the
	// configured number of weeks have been played — the PDF's Figure 1.a
	// is the first figure to show predictions, after 4 completed weeks.
	ErrPredictionsNotAvailable = errors.New("service: predictions require more completed weeks")

	// ErrMatchNotPlayed is returned by EditMatch when targeting a fixture
	// that hasn't been played yet (there's no result to edit).
	ErrMatchNotPlayed = errors.New("service: cannot edit a match that has not been played")

	// ErrInvalidScore is returned by EditMatch for negative goal counts.
	ErrInvalidScore = errors.New("service: goal counts must be non-negative")
)
