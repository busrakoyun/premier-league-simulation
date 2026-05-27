package main

import (
	"fmt"
	"log"

	"github.com/busrakoyun/premier-league-simulation/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	fmt.Printf("premier-league-simulation booting (port=%s, mc_iterations=%d)\n",
		cfg.Port, cfg.MonteCarloIterations)
}
