package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/busrakoyun/premier-league-simulation/internal/config"
	httpapi "github.com/busrakoyun/premier-league-simulation/internal/http"
	"github.com/busrakoyun/premier-league-simulation/internal/prediction"
	"github.com/busrakoyun/premier-league-simulation/internal/repository/postgres"
	"github.com/busrakoyun/premier-league-simulation/internal/service"
	"github.com/busrakoyun/premier-league-simulation/internal/simulation"
)

// defaultSeasonName is used by EnsureSeason if no season has been created.
const defaultSeasonName = "2025-26 Season"

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("ping db: %v", err)
	}

	repo := postgres.New(pool)
	sim := simulation.NewPoissonSimulator()
	predictor := prediction.NewMonteCarlo(sim, cfg.MonteCarloIterations, cfg.RandomSeed)
	rng := simulation.NewRNG(cfg.RandomSeed)
	svc := service.New(repo, sim, predictor, rng)

	if err := svc.EnsureSeason(ctx, defaultSeasonName); err != nil {
		log.Fatalf("ensure season: %v", err)
	}

	h := httpapi.NewHandler(svc)
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           h.Router(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		// PlayAll + a fresh Monte Carlo finish under ~100ms even with MC_ITERATIONS=10000,
		// but Render free-tier cold starts mean we leave generous headroom.
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		log.Printf("listening on %s (mc_iterations=%d, default_season=%q)",
			srv.Addr, cfg.MonteCarloIterations, defaultSeasonName)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		log.Fatalf("server error: %v", err)
	case sig := <-quit:
		log.Printf("received %v, shutting down...", sig)
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("shutdown: %v", err)
	}
	log.Println("server stopped")
}
