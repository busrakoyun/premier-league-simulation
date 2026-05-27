package simulation

import (
	"math/rand/v2"
	"time"
)

// pcgMix is the golden-ratio constant used by many PRNGs to derive a second
// 64-bit seed value from a single user-supplied seed without correlated bits.
const pcgMix uint64 = 0x9E3779B97F4A7C15

// NewRNG returns a PCG-based RNG. A non-zero seed is deterministic — tests
// always pass an explicit seed. Seed 0 means "use the wall clock", which is
// what production wants so two consecutive predictions are statistically
// independent samples rather than identical replays.
func NewRNG(seed int64) *rand.Rand {
	var s uint64
	if seed == 0 {
		s = uint64(time.Now().UnixNano())
	} else {
		s = uint64(seed)
	}
	return rand.New(rand.NewPCG(s, s^pcgMix))
}
