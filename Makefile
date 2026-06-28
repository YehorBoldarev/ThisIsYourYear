# ThisIsYourYear — developer task runner.

COMPOSE := docker compose -f deployments/docker-compose.yml
PKG     := ./...

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help.
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

.PHONY: run
run: ## Run the API locally (expects Mongo reachable, e.g. via `make compose-up`).
	go run ./cmd/api

.PHONY: build
build: ## Compile the API binary into ./bin.
	go build -o bin/api ./cmd/api

.PHONY: test
test: ## Run unit tests with the race detector.
	go test -race $(PKG)

.PHONY: test-int
test-int: ## Run integration tests (opt-in; requires Docker for testcontainers).
	go test -race -tags=integration $(PKG)

.PHONY: fmt
fmt: ## Format all Go source.
	gofmt -w .

.PHONY: vet
vet: ## Run go vet.
	go vet $(PKG)

.PHONY: lint
lint: ## Run golangci-lint (requires golangci-lint installed).
	golangci-lint run

.PHONY: tidy
tidy: ## Tidy and verify module dependencies.
	go mod tidy

.PHONY: compose-up
compose-up: ## Start MongoDB in the background.
	$(COMPOSE) up -d mongo

.PHONY: compose-down
compose-down: ## Stop and remove the dev containers.
	$(COMPOSE) down
