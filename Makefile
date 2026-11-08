# ThisIsYourYear — developer task runner.

COMPOSE := docker compose -f deployments/docker-compose.yml
PKG     := ./...

# golangci-lint is pinned here and in .github/workflows/ci.yml; bump both together.
# `make lint` builds exactly this version into ./bin on first use (no global install).
GOLANGCI_LINT_VERSION := v2.15.0
GOLANGCI_LINT         := bin/golangci-lint-$(GOLANGCI_LINT_VERSION)

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help.
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

.PHONY: run
run: ## Run the API locally (expects Mongo reachable, e.g. via `make compose-up`).
	go run ./cmd/api

.PHONY: build
build: ## Compile the API binary into ./bin (backend only, no embedded SPA).
	go build -o bin/api ./cmd/api

WEBUI_DIST := internal/transport/http/webui/dist

.PHONY: web-build
web-build: ## Build the frontend and stage it under the Go embed directory.
	cd web && npm ci && npm run build
	find $(WEBUI_DIST) -mindepth 1 ! -name .gitkeep -delete
	cp -r web/dist/. $(WEBUI_DIST)/

.PHONY: build-all
build-all: web-build ## Build the self-contained binary with the SPA embedded (-tags embedui).
	go build -tags embedui -o bin/api ./cmd/api

.PHONY: test
test: ## Run unit tests with the race detector.
	go test -race $(PKG)

.PHONY: test-int
test-int: ## Run integration tests (opt-in; requires Docker for testcontainers).
	go test -race -tags=integration $(PKG)

.PHONY: fmt
fmt: ## Format all Go source (skips web/node_modules via the go.mod ignore directive).
	go fmt $(PKG)

# Build tags: `integration` adds the integration tests to the default file set, and
# `embedui` swaps in the embedded-SPA file, so these two passes cover every Go file.
.PHONY: vet
vet: ## Run go vet over all build-tag combinations.
	go vet -tags=integration $(PKG)
	go vet -tags=embedui $(PKG)

$(GOLANGCI_LINT):
	@mkdir -p bin
	GOBIN=$(CURDIR)/bin go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	@mv bin/golangci-lint $@

.PHONY: lint
lint: $(GOLANGCI_LINT) ## Run golangci-lint (pinned version) over all build-tag combinations.
	$(GOLANGCI_LINT) run --build-tags=integration $(PKG)
	$(GOLANGCI_LINT) run --build-tags=embedui $(PKG)

.PHONY: tidy
tidy: ## Tidy and verify module dependencies.
	go mod tidy

.PHONY: compose-up
compose-up: ## Start MongoDB in the background.
	$(COMPOSE) up -d mongo

.PHONY: compose-down
compose-down: ## Stop and remove the dev containers.
	$(COMPOSE) down
