package simulation_test

import (
	"math"
	"testing"

	"github.com/busrakoyun/premier-league-simulation/internal/domain"
	"github.com/busrakoyun/premier-league-simulation/internal/simulation"
)

func TestSamplePoisson_NonPositiveLambdaReturnsZero(t *testing.T) {
	rng := simulation.NewRNG(1)
	for _, lambda := range []float64{0, -1, -0.0001} {
		if got := simulation.SamplePoisson(rng, lambda); got != 0 {
			t.Errorf("SamplePoisson(rng, %v) = %d, want 0", lambda, got)
		}
	}
}

func TestSamplePoisson_EmpiricalMeanMatchesLambda(t *testing.T) {
	const N = 5000
	rng := simulation.NewRNG(42)

	for _, lambda := range []float64{0.5, 1.4, 2.5, 4.0} {
		sum := 0
		for i := 0; i < N; i++ {
			sum += simulation.SamplePoisson(rng, lambda)
		}
		mean := float64(sum) / N
		// Sample-mean stddev for Poisson is sqrt(λ/N). 4σ keeps this test
		// stable across CI runs; flake rate ≈ 1 in 16000.
		tol := 4 * math.Sqrt(lambda/N)
		if math.Abs(mean-lambda) > tol {
			t.Errorf("lambda=%.2f: empirical mean %.4f outside %.2f±%.4f",
				lambda, mean, lambda, tol)
		}
	}
}

func TestPoissonSimulator_DeterministicForFixedSeed(t *testing.T) {
	home := domain.Team{Name: "H", AttackStrength: 1.0, DefenseStrength: 1.0, HomeAdvantage: 1.3}
	away := domain.Team{Name: "A", AttackStrength: 1.0, DefenseStrength: 1.0, HomeAdvantage: 1.3}

	sim := simulation.NewPoissonSimulator()
	rng1 := simulation.NewRNG(123)
	rng2 := simulation.NewRNG(123)

	for i := 0; i < 20; i++ {
		a := sim.Simulate(rng1, home, away)
		b := sim.Simulate(rng2, home, away)
		if a != b {
			t.Errorf("iter %d: simulators diverged: %+v vs %+v", i, a, b)
		}
	}
}

func TestPoissonSimulator_StrongerTeamDominates(t *testing.T) {
	strong := domain.Team{Name: "Strong", AttackStrength: 1.4, DefenseStrength: 1.4, HomeAdvantage: 1.3}
	weak := domain.Team{Name: "Weak", AttackStrength: 0.7, DefenseStrength: 0.7, HomeAdvantage: 1.3}

	sim := simulation.NewPoissonSimulator()
	rng := simulation.NewRNG(42)

	const N = 1000
	strongWins, weakWins := 0, 0
	for i := 0; i < N; i++ {
		score := sim.Simulate(rng, strong, weak)
		switch {
		case score.HomeGoals > score.AwayGoals:
			strongWins++
		case score.HomeGoals < score.AwayGoals:
			weakWins++
		}
	}
	// With λ_home ≈ 3.6 and λ_away ≈ 0.7 the strong side should win at least
	// 5× as often as the weak one. Generous bound: this test is about
	// directionality, not exact rates.
	if strongWins < 5*weakWins {
		t.Errorf("strong should dominate: strong=%d weak=%d", strongWins, weakWins)
	}
}

func TestPoissonSimulator_HomeAdvantageTilts(t *testing.T) {
	// Two identical teams; the only asymmetry is the home advantage applied
	// to the home side. Home should win more often than away.
	a := domain.Team{Name: "A", AttackStrength: 1.0, DefenseStrength: 1.0, HomeAdvantage: 1.3}
	b := domain.Team{Name: "B", AttackStrength: 1.0, DefenseStrength: 1.0, HomeAdvantage: 1.3}

	sim := simulation.NewPoissonSimulator()
	rng := simulation.NewRNG(7)

	const N = 2000
	homeWins, awayWins := 0, 0
	for i := 0; i < N; i++ {
		score := sim.Simulate(rng, a, b)
		switch {
		case score.HomeGoals > score.AwayGoals:
			homeWins++
		case score.HomeGoals < score.AwayGoals:
			awayWins++
		}
	}
	if homeWins <= awayWins {
		t.Errorf("home advantage should tilt: home=%d away=%d", homeWins, awayWins)
	}
}
