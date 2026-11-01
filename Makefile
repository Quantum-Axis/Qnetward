# Common development tasks. Requires GNU make (Linux, macOS, or Git Bash on Windows).
# Each target is a thin wrapper, so the commands also work when typed directly.

WEB := apps/web
BIN := .tmp/qnetward

.PHONY: help build run migrate migrate-status test test-db vet fmt lint web-install web-dev web-build web-lint web-e2e e2e-api check clean

help: ## List targets
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-12s %s\n", $$1, $$2}'

build: ## Build the qnetward binary into .tmp/
	go build -o $(BIN) ./cmd/qnetward

run: build ## Run the API server on 127.0.0.1:8080 using .env
	scripts/with-env.sh $(BIN) serve -listen 127.0.0.1:8080

migrate: build ## Apply database migrations to QNETWARD_DB_DSN from .env
	scripts/with-env.sh $(BIN) migrate

migrate-status: build ## List migrations and whether each is applied
	scripts/with-env.sh $(BIN) migrate -status

test: ## Go unit tests (Postgres-backed tests skip without QNETWARD_TEST_POSTGRES_DSN)
	go test ./...

test-db: ## Go tests including Postgres-backed ones (QNETWARD_TEST_POSTGRES_DSN from env or .env)
	scripts/with-env.sh sh -c 'test -n "$$QNETWARD_TEST_POSTGRES_DSN" || { echo "QNETWARD_TEST_POSTGRES_DSN is not set"; exit 1; }; go test -race -p 1 ./...'

vet: ## go vet
	go vet ./...

fmt: ## gofmt all Go files
	gofmt -w cmd internal

lint: vet web-lint ## All linters

web-install: ## Install frontend dependencies
	pnpm --dir $(WEB) install --frozen-lockfile

web-dev: ## Next.js dev server on 127.0.0.1:3000
	QNETWARD_BACKEND_URL=http://127.0.0.1:8080 NEXT_PUBLIC_QNETWARD_WS_URL=ws://127.0.0.1:8080/ws pnpm --dir $(WEB) dev --hostname 127.0.0.1 --port 3000

web-build: ## Production build of the frontend
	pnpm --dir $(WEB) build

web-lint: ## ESLint for the frontend
	pnpm --dir $(WEB) lint

web-e2e: ## Playwright end-to-end tests (needs a running stack; see e2e-api)
	pnpm --dir $(WEB) test:e2e

e2e-api: build ## API server for the e2e suite on :8090 (test database, email off, demo on)
	scripts/e2e-backend.sh

check: vet test web-lint ## What CI runs, minus the Postgres service

clean: ## Remove build output
	rm -rf .tmp $(WEB)/.next
