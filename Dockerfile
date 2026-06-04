# syntax=docker/dockerfile:1

# ---------------------------------------------------------------------------
# Stage 1 — build the Vue SPA. The dist is embedded into the Go binary in the
# next stage, so it must be produced before `go build` runs.
# ---------------------------------------------------------------------------
FROM node:20-alpine AS web
WORKDIR /web

# Install against the lockfile first so this layer caches across source edits.
COPY web/package.json web/package-lock.json ./
RUN npm ci

# `npm run build` = vue-tsc typecheck + vite build -> /web/dist.
COPY web/ ./
RUN npm run build

# ---------------------------------------------------------------------------
# Stage 2 — build the Go binary (with the SPA embedded) plus the migrate CLI.
# CGO is off so both binaries are static and run on a bare alpine runtime.
# ---------------------------------------------------------------------------
FROM golang:1.24-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# go:embed reads internal/http/dist at compile time; drop the freshly built
# SPA in before building so the binary serves the real frontend, not the
# in-code fallback page.
COPY --from=web /web/dist/ ./internal/http/dist/

ENV CGO_ENABLED=0 GOOS=linux
RUN go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/api

# golang-migrate CLI (postgres driver only, pure Go). The entrypoint runs it
# against DATABASE_URL on startup, since Render's free tier has no pre-deploy
# hook. Pinned to match the version used by `make tools`.
RUN go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.18.1 \
    && cp "$(go env GOPATH)/bin/migrate" /out/migrate

# ---------------------------------------------------------------------------
# Stage 3 — minimal runtime. Non-root, TLS roots for the Postgres connection.
# ---------------------------------------------------------------------------
FROM alpine:3.20 AS runtime

RUN apk add --no-cache ca-certificates \
    && adduser -D -H -u 10001 appuser

WORKDIR /app

COPY --from=build /out/server          /app/server
COPY --from=build /out/migrate         /usr/local/bin/migrate
COPY --from=build /src/db/migrations   /app/db/migrations

USER appuser
EXPOSE 8080

# Apply migrations (idempotent — golang-migrate tracks the version and takes an
# advisory lock), then hand off to the server. `exec` makes the server PID 1 so
# it receives SIGTERM for the graceful shutdown main.go implements.
ENTRYPOINT ["/bin/sh", "-c", "migrate -path /app/db/migrations -database \"$DATABASE_URL\" up && exec /app/server"]
