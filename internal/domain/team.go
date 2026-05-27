package domain

// Team is a club competing in the league.
//
// AttackStrength and DefenseStrength are dimensionless multipliers centered
// around 1.0: a team with attack 1.3 scores ~30% more than the league baseline
// in attack-equivalent conditions; defense 1.3 concedes ~23% less (1/1.3).
// HomeAdvantage is the multiplicative bonus applied to the home side's lambda.
type Team struct {
	ID              int64
	Name            string
	ShortName       string
	AttackStrength  float64
	DefenseStrength float64
	HomeAdvantage   float64
}
