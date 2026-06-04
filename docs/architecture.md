# Architecture

The Premier League Simulation is a single Go binary that serves a JSON API and
an embedded Vue SPA, backed by PostgreSQL. It follows a layered architecture
with one strict rule: **dependencies point inward, toward the domain.** The
transport layer knows about the service; the service knows about interfaces;
nothing in the core knows about HTTP or SQL.

## Layer diagram

```
                     ┌───────────────────────────────────────────┐
   HTTP request ──►  │  internal/http                            │
   embedded SPA      │  chi router · handlers · DTOs · errors    │
                     └─────────────────────┬─────────────────────┘
                                           │ calls (domain types in/out)
                     ┌─────────────────────▼─────────────────────┐
   orchestration ─►  │  internal/service — LeagueService         │
                     │  write mutex · transactions · gating      │
                     └──────┬──────────────┬──────────────┬──────┘
                            │ depends on interfaces, never concretes
            ┌───────────────▼──┐  ┌────────▼────────┐  ┌──▼──────────────┐
            │ repository (IF)  │  │ simulation (IF) │  │ prediction (IF) │
            │  Postgres impl   │  │ Poisson model   │  │ Monte Carlo     │
            │  (pgx + sqlc)    │  │                 │  │                 │
            └───────────────┬──┘  └────────┬────────┘  └──┬──────────────┘
                            │              │              │
                     ┌──────▼──────────────▼──────────────▼──────┐
   shared kernel ─►  │  internal/domain   Team · Match · Season  │
                     │                    StandingRow · …        │
                     │  internal/standings   pure table + fixture│
                     │                       generation          │
                     └───────────────────────────────────────────┘
```

## Layers

| Package | Responsibility |
|---------|----------------|
| `cmd/api` | Process entry point: load config, open the pgx pool, wire dependencies, `EnsureSeason`, start the HTTP server, handle graceful shutdown. |
| `internal/http` | Transport only. Thin handlers parse requests, call `LeagueService`, and map domain values/errors to JSON DTOs and status codes. Serves the embedded Vue build with SPA fallback. |
| `internal/service` | The orchestration core. `LeagueService` owns the repository, simulator, and predictor (all as interfaces), serialises writes behind a mutex, wraps multi-step writes in transactions, and enforces business rules (e.g. predictions gated until week 4). |
| `internal/repository` | The persistence boundary: the `LeagueRepository` interface plus a Postgres implementation built on pgx and sqlc-generated queries. |
| `internal/simulation` | The match model: the `MatchSimulator` interface, the Poisson implementation, and a seedable PCG RNG. |
| `internal/prediction` | The forecast model: the `PredictionEngine` interface and the Monte Carlo implementation. |
| `internal/standings` | Pure, no-I/O functions: compute a league table from matches, and generate a double round-robin fixture list. Reused by the service, the edit path, and the predictor. |
| `internal/domain` | The shared kernel — plain types (`Team`, `Match`, `Season`, `StandingRow`, `PredictionRow`, `SeasonSnapshot`) with no dependencies. Everything imports it; it imports nothing. |

## The interfaces

Three seams sit between the service and its collaborators. Each is small and
expressed in domain terms:

```go
// internal/simulation — how a single match resolves into a score.
type MatchSimulator interface {
    Simulate(rng *rand.Rand, home, away domain.Team) domain.MatchScore
}

// internal/prediction — how a season snapshot becomes championship odds.
type PredictionEngine interface {
    Predict(snapshot domain.SeasonSnapshot) ([]domain.PredictionRow, error)
}

// internal/repository — every persistence operation the service needs.
type LeagueRepository interface {
    ListTeams(ctx context.Context) ([]domain.Team, error)
    CreateSeason(ctx context.Context, name string, totalWeeks int) (domain.Season, error)
    GetCurrentSeason(ctx context.Context) (domain.Season, error)
    // … matches, scores, week pointer, deletes …
    WithTx(ctx context.Context, fn func(ctx context.Context, repo LeagueRepository) error) error
}
```

**Why interface-based design here:**

- **Testability.** Service tests run against an in-memory repository mock and a
  fixed-seed simulator — no database, no clock, no flakiness. The Postgres
  implementation is exercised separately by integration tests that skip when
  `TEST_DATABASE_URL` is unset, so `go test ./...` stays green anywhere.
- **Dependency inversion.** The business logic depends on abstractions it owns,
  not on pgx or a specific goal model. The arrows in the diagram point inward.
- **Swappability.** Each seam names a real future option: Poisson → a Dixon-Coles
  bivariate model; Monte Carlo → an analytic or learned predictor; Postgres → a
  read-replica split. None of those ripple into the service.
- **A narrow surface.** `LeagueRepository` exposes only what the service calls,
  which keeps the mock small and the contract obvious.

`MatchSimulator.Simulate` takes the RNG as a parameter rather than holding it,
so the simulator is stateless with respect to randomness — the live write path
and each Monte Carlo run scope their own RNG without sharing mutable state.

## Key decisions

**Compute-on-read standings.** The league table is a pure function of the played
matches (`standings.Calculate`), recomputed on every `GET /standings` rather than
stored or cached. Twelve rows compute in microseconds, so a cache would add
invalidation complexity and buy nothing — and an edited score is reflected on the
very next read with no rebuild step.

**Single-threaded Monte Carlo.** `Predict` runs N=10,000 iterations in a tight
sequential loop. At this scale (≤12 unplayed matches × 10k ≈ 120k simulated
matches) the whole forecast finishes well under 100 ms on a free Render instance.
Goroutine scheduling and per-iteration RNG coordination would dominate any
parallel speedup, so the simple loop is also the fast one. It reuses a single
scratch slice across iterations to avoid 10k allocations.

**Per-season write mutex.** `LeagueService` holds one `sync.Mutex` that serialises
every write path (next-week, play-all, edit, reset). For a single Render instance
this is exactly the right amount of concurrency control: it makes the
read-modify-write sequences atomic against each other without distributed
machinery. Horizontal scaling would instead call for optimistic concurrency
(a row-version compare-and-set in Postgres) — out of scope for this case.

**Poisson goal model.** Match scores are independent Poisson draws, a well-
established fit for football goal counts (Maher, 1982). Each side's rate λ
decomposes into an attack multiplier (raises my goals) and the opponent's
defense multiplier (lowers them), with a home-advantage multiplier on the host —
an interpretable, directional model the frontend can explain to the user.
Treating the two sides as independent is a deliberate simplification that trades
a little realism for a much simpler model.

**Idempotent recompute on edit.** Because standings and predictions are pure
functions over the match set, editing a played score (`PATCH /matches/{id}`)
requires no cache rebuild or recomputation cascade — the write just updates one
row, and the next read derives the new table and odds from scratch. The same
property makes `play-all` and `next-week` safe to wrap in a single transaction:
they commit together or roll back, and nothing downstream holds stale derived
state.
