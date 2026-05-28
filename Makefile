.PHONY: help build run test sqlc migrate-up migrate-down db-up db-down clean tools web-install web-dev web-build web-typecheck

DATABASE_URL ?= postgres://postgres:postgres@localhost:5432/premier_league?sslmode=disable
MIGRATE      := migrate -path db/migrations -database "$(DATABASE_URL)"

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  %-14s %s\n", $$1, $$2}'

tools: ## Install dev tools (sqlc, migrate)
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

build: ## Build the Go binary
	go build -o bin/server ./cmd/api

run: build ## Build and run the API
	./bin/server

test: ## Run all Go tests
	go test ./...

sqlc: ## Regenerate sqlc code
	sqlc generate

migrate-up: ## Apply database migrations
	$(MIGRATE) up

migrate-down: ## Roll back the last migration
	$(MIGRATE) down 1

db-up: ## Start a local Postgres in Docker
	docker run --name pl-postgres -d \
		-e POSTGRES_USER=postgres -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=premier_league \
		-p 5432:5432 postgres:16-alpine

db-down: ## Stop and remove the local Postgres container
	docker stop pl-postgres && docker rm pl-postgres

web-install: ## Install Vue / Vite dependencies
	cd web && npm install

web-dev: ## Run the Vite dev server (proxies /api to :8080)
	cd web && npm run dev

web-build: ## Build the Vue dist and copy it into the Go embed directory
	cd web && npm run build
	rm -rf internal/http/dist/assets
	cp -r web/dist/* internal/http/dist/

web-typecheck: ## Run vue-tsc against the SPA
	cd web && npm run typecheck

clean: ## Remove build artifacts
	rm -rf bin/ web/dist/
