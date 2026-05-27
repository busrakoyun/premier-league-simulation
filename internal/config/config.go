package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all runtime configuration loaded from environment variables.
// DATABASE_URL is intentionally not validated here so that go build / go test
// don't require it to be set; the DB connection step in main.go validates it.
type Config struct {
	DatabaseURL          string
	Port                 string
	MonteCarloIterations int
	RandomSeed           int64
}

// Load reads the environment into a Config, applying defaults and rejecting
// nonsensical values (non-positive Monte Carlo iteration count).
func Load() (Config, error) {
	cfg := Config{
		Port:                 getEnv("PORT", "8080"),
		DatabaseURL:          os.Getenv("DATABASE_URL"),
		MonteCarloIterations: getEnvInt("MC_ITERATIONS", 10000),
		RandomSeed:           int64(getEnvInt("RANDOM_SEED", 0)),
	}
	if cfg.MonteCarloIterations <= 0 {
		return Config{}, fmt.Errorf("MC_ITERATIONS must be positive, got %d", cfg.MonteCarloIterations)
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}
