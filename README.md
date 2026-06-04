# Premier League Simulation

A football league simulator: generate a round-robin fixture list, simulate match
results from team strengths using a Poisson goal model, track the live standings,
and forecast each team's title chances with a Monte Carlo prediction engine. A Go
API serves both the JSON endpoints and an embedded Vue 3 single-page app.

### 🔗 Live demo: **[pl-sim.onrender.com](https://pl-sim.onrender.com)**

> **Heads up:** the demo runs on Render's free tier, which puts the instance to
> sleep after ~15 minutes of inactivity. The **first request after a sleep can
> take ~50 seconds** while the container cold-starts (and runs migrations); every
> request after that is fast. If the page seems to hang, give it a moment.

---

## Tech stack

| Layer | Technology |
|-------|------------|
| Language | **Go 1.24** |
| HTTP router | **chi v5** (lightweight, idiomatic `net/http`) |
| Database | **PostgreSQL 16** |
| DB access | **pgx v5** driver + **sqlc** (type-safe Go from SQL) |
| Migrations | **golang-migrate** |
| Frontend | **Vue 3** + **TypeScript** |
| State | **Pinia** |
| Styling | **Tailwind CSS** |
| Build | **Vite 6** (frontend), `go build` with `go:embed` (single binary) |
| Deploy | **Docker** multi-stage → **Render** (Blueprint) |

The production artifact is a **single static Go binary** with the Vue build
embedded via `go:embed`, so there's no separate frontend host or CDN to manage.

---

## How the simulation works

### Match model (Poisson)

Each match is simulated as two **independent Poisson draws** — one goal count per
side. A team's scoring rate `λ` is built from dimensionless strength multipliers
(centered on 1.0) plus a home-advantage bonus:

```
λ_home = R · attackₕ · home_advantageₕ / defenseₐ
λ_away = R · attackₐ                    / defenseₕ

R = 1.4   (DefaultBaseGoalRate ≈ Premier League goals per team per game)
```

- **Attack** raises your own λ; the opponent's **defense** lowers it (defense 1.3
  ⇒ concede ~23% fewer).
- **Home advantage** (default 1.3) multiplies the host's λ only.
- Goals are then sampled `HomeGoals ~ Poisson(λ_home)`, `AwayGoals ~ Poisson(λ_away)`
  using Knuth's algorithm.

**Worked example** — Chelsea (attack 1.30, home adv 1.30) at home to Liverpool
(attack 0.85, defense 0.85); Chelsea defense 1.20:

```
λ_chelsea  = 1.4 · 1.30 · 1.30 / 0.85 ≈ 2.78 expected goals
λ_liverpool = 1.4 · 0.85         / 1.20 ≈ 0.99 expected goals
```

Why Poisson: it's a well-established fit for football goal counts (Maher, 1982).
The attack/defense split keeps the model interpretable and directional; treating
the two sides as independent is a deliberate simplification.

### Championship prediction (Monte Carlo)

Title odds come from a **Monte Carlo** simulation. For each of **N = 10,000**
iterations:

1. Take the matches already played as fixed.
2. Simulate every remaining (unplayed) fixture with the Poisson model.
3. Compute the final table and award the title to the top team — **ties at the
   top split the championship equally** (so the reported rates always sum to 1).

Each team's `championship_rate` is `titles_won / N`.

**Single-threaded, on purpose.** At ≤12 remaining matches × 10k iterations
(~120k simulated matches), the sequential loop finishes well under 100 ms on a
free Render instance. Goroutine scheduling and per-iteration RNG coordination
would cost more than they'd save, so the simple loop is also the fast one — a
decision made by profiling, not assumption. Predictions are **gated until 4 weeks
have been played** (matching the case brief); before that the endpoint returns
`422 PREDICTIONS_NOT_YET_AVAILABLE`.

---

## Why Vue 3 (not Vue 2.7) and Pinia (not Vuex)

**Vue 3, not 2.7** — Vue 3 is the current default and Vue 2 reached end-of-life in
December 2023. Vue 3 brings the Composition API with `<script setup>`, first-class
TypeScript support (the components here are written in TS), and a smaller,
Proxy-based reactivity system. Starting a new project on 2.7 would mean adopting a
maintenance-mode framework.

**Pinia, not Vuex** — Pinia is the officially recommended store for Vue 3 (it
effectively *became* "Vuex 5"). It has a simpler API with no mutations boilerplate,
full TypeScript inference out of the box, and a modular store design that fits the
Composition API. Vuex 4 runs on Vue 3 but is in maintenance mode, so Pinia is the
forward-looking choice.

---

## Local setup

### Prerequisites

- **Go 1.24+**
- **Node 20+**
- **Docker** (for a local Postgres) — or your own Postgres 16
- Dev tools: `make tools` installs **sqlc** and **golang-migrate**

### Environment

Copy the example env file and adjust if needed:

```bash
cp .env.example .env
```

| Variable | Default | Purpose |
|----------|---------|---------|
| `DATABASE_URL` | `postgres://postgres:postgres@localhost:5432/premier_league?sslmode=disable` | Postgres connection string (required at runtime). |
| `PORT` | `8080` | HTTP listen port. |
| `MC_ITERATIONS` | `10000` | Monte Carlo iterations per prediction. |
| `RANDOM_SEED` | `0` | Simulator RNG seed. `0` = time-based (non-deterministic); any non-zero value makes runs reproducible. |

### Run it

```bash
make tools        # install sqlc + migrate (one time)
make db-up        # start a local Postgres in Docker
make migrate-up   # apply schema + seed the four teams
make web-build    # build the Vue SPA and embed it into the Go binary
make run          # build and start the API on :8080
```

Open <http://localhost:8080>. For frontend hot-reload during development, run the
API with `make run` and the Vite dev server with `make web-dev` (it proxies
`/api` and `/healthz` to `:8080`).

### Other useful commands

```bash
make test           # go test ./...   (Postgres integration tests skip without TEST_DATABASE_URL)
make web-typecheck  # vue-tsc type check
make sqlc           # regenerate type-safe query code from db/queries
make migrate-down   # roll back the last migration
make db-down        # stop and remove the local Postgres container
make help           # list all targets
```

---

## Docker

The included multi-stage `Dockerfile` builds the Vue dist (`node:20-alpine`),
compiles a static Go binary with the dist embedded (`golang:1.24-alpine`), and
ships a minimal `alpine:3.20` runtime that runs as a **non-root** user. The
container entrypoint applies database migrations (idempotent) and then starts the
server.

```bash
# Build
docker build -t pl-sim .

# Run (point it at a reachable Postgres)
docker run --rm -p 8080:8080 \
  -e DATABASE_URL="postgres://user:pass@host:5432/premier_league?sslmode=disable" \
  pl-sim
```

The resulting image is ~28 MB.

---

## Deployment (Render)

The repo ships a `render.yaml` **Blueprint** that provisions a free Docker web
service plus a free managed Postgres:

1. Push the repo to GitHub.
2. In Render: **New → Blueprint**, and select the repository.
3. Render builds the Dockerfile, provisions the database, and wires
   `DATABASE_URL` into the web service automatically.

**Migrations on Render.** The free tier has no pre-deploy hook, so the container
**entrypoint runs `migrate up` on startup** before launching the server (the
`migrate` CLI and SQL files are baked into the image). Migrations are idempotent —
golang-migrate tracks the applied version and takes an advisory lock — so repeated
cold starts are safe. On the very first deploy you may see a restart or two if the
web container boots before Postgres is reachable; Render retries until it is.

> The free Postgres instance and the sleeping web service are fine for a demo but
> not intended for production traffic.

---

## API reference

Base URL: `http://localhost:8080` locally, or `https://pl-sim.onrender.com` live.
All responses are JSON. Errors use a flat `{ "error": "...", "code": "..." }`
envelope with stable `code` strings.

| Method | Path | Description | Notable responses |
|--------|------|-------------|-------------------|
| `GET` | `/healthz` | Liveness probe. | `200 {"status":"ok"}` |
| `GET` | `/api/seasons/current` | Active season metadata. | `404 NO_CURRENT_SEASON` |
| `GET` | `/api/seasons/current/teams` | Teams + strength parameters. | `200` |
| `GET` | `/api/seasons/current/standings` | League table (computed on read). | `200` |
| `GET` | `/api/seasons/current/matches` | All fixtures, played and unplayed. | `200` |
| `GET` | `/api/seasons/current/predictions` | Monte Carlo title odds (sum to 1). | `422 PREDICTIONS_NOT_YET_AVAILABLE` before week 4 |
| `POST` | `/api/seasons/current/next-week` | Simulate the next week; advance the pointer. | `422 SEASON_COMPLETE` |
| `POST` | `/api/seasons/current/play-all` | Simulate through to the end of the season. | `422 SEASON_COMPLETE` |
| `POST` | `/api/seasons/current/reset` | Wipe results, reset to week 0, regenerate fixtures. | `204 No Content` |
| `PATCH` | `/api/matches/{id}` | Edit a played match's score. Body: `{"home_goals":2,"away_goals":1}`. | `422 MATCH_NOT_PLAYED`, `400 INVALID_SCORE`, `404 NOT_FOUND` |

A ready-to-import Postman collection lives at
[`docs/postman_collection.json`](docs/postman_collection.json) — switch the
`{{base_url}}` variable to the live URL to exercise the deployment.

---

## Design decisions

| Decision | Alternative considered | Why |
|----------|------------------------|-----|
| **Compute standings on read** (pure function over matches) | Persist/cache the table, invalidate on writes | 12 rows compute in microseconds; a cache adds invalidation bugs for zero gain, and edits show up on the next read for free. |
| **Single-threaded Monte Carlo** (N=10k) | Parallel workers across goroutines | ~120k sims finish <100 ms; goroutine + RNG coordination overhead would dominate. Faster *and* simpler. |
| **One per-instance write mutex** | Postgres row-version compare-and-set | Right level of concurrency control for a single instance; CAS only earns its complexity under horizontal scaling. |
| **Poisson goal model** with attack/defense split | Fixed win probabilities; bivariate (Dixon-Coles) | Established fit for football scores, interpretable and directional, behind a `MatchSimulator` seam so a richer model can drop in later. |
| **Embedded SPA via `go:embed`** | Separate static host / CDN | One deployable artifact, same-origin API (no CORS in prod), trivial Render setup. |
| **sqlc-generated queries** | Hand-written `database/sql`; a full ORM | Type-safe Go from plain SQL — no ORM magic, no runtime string-building, compile-time checks against the schema. |
| **Interfaces at every seam** (repo / simulator / predictor) | Concrete types wired directly | Unit tests use in-memory mocks (no DB/clock); implementations stay swappable; dependencies point inward. |
| **Migrations from the container entrypoint** | Render pre-deploy hook; manual step | Free tier has no pre-deploy hook; an idempotent `migrate up` on startup keeps the Blueprint one-click and self-contained. |

---

## Project structure

```
.
├── cmd/api/                  # main(): config, DB pool, wiring, server lifecycle
├── internal/
│   ├── config/               # env-var loading + validation
│   ├── domain/               # core types (Team, Match, Season, StandingRow, …)
│   ├── http/                 # chi router, handlers, DTOs, error mapping, SPA serving
│   │   └── dist/             # embedded Vue build (populated by `make web-build`)
│   ├── prediction/           # PredictionEngine interface + Monte Carlo
│   ├── repository/           # LeagueRepository interface
│   │   ├── postgres/         #   pgx-backed implementation + transactions
│   │   └── sqlc/             #   generated type-safe query code
│   ├── service/              # LeagueService: orchestration, write mutex, gating
│   ├── simulation/           # MatchSimulator interface + Poisson model + RNG
│   ├── standings/            # pure table calculator + round-robin fixtures
│   └── testutil/             # shared test helpers
├── db/
│   ├── migrations/           # golang-migrate SQL (schema + team seed)
│   └── queries/              # sqlc query source
├── web/                      # Vue 3 + Vite + Pinia + Tailwind SPA
├── docs/
│   ├── architecture.md       # layer diagram, interfaces, key decisions
│   └── postman_collection.json
├── Dockerfile                # multi-stage build → minimal non-root runtime
├── render.yaml               # Render Blueprint (web service + free Postgres)
└── Makefile                  # dev workflow targets
```

See [`docs/architecture.md`](docs/architecture.md) for the layer diagram, the
interface contracts, and the reasoning behind each design decision.
