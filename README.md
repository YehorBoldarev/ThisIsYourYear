# ThisIsYourYear

A multi-user web application for tracking yearly goals. Users create big goals
(**Topics**) and concrete **Goals** beneath them; completing a Goal becomes an
achievement, and each Topic surfaces a motivational message based on progress. A
dashboard shows overall status and a countdown to the New Year.

This repository currently contains the **Go backend**. See [`plans/`](./plans)
for the system plan and per-step implementation plans.

## Tech stack

- **Backend:** Go 1.26, `net/http` + [chi](https://github.com/go-chi/chi) router
- **Database:** MongoDB (official driver)
- **API:** REST/JSON under `/api/v1`
- **Frontend (later):** React + TailwindCSS

## Requirements

- Go 1.26+
- Docker (for MongoDB via docker-compose; optional `golangci-lint` for linting)

## Getting started

```bash
# 1. Start MongoDB
make compose-up

# 2. (optional) configure local overrides
cp .env.example .env

# 3. Run the API
make run
```

The API listens on `http://localhost:8080`.

```bash
curl http://localhost:8080/healthz   # liveness  -> 200 {"status":"ok"}
curl http://localhost:8080/readyz    # readiness -> 200 when Mongo reachable, else 503
```

To run everything (API + Mongo) in Docker:

```bash
docker compose -f deployments/docker-compose.yml --profile app up --build
```

## Development

```bash
make help      # list all targets
make test      # unit tests (race detector)
make fmt       # format
make vet       # go vet
make lint      # golangci-lint
make tidy      # tidy go.mod
```

## Project layout

```
cmd/api                  process entrypoint and wiring
internal/config          environment configuration + validation
internal/platform/logging  structured logging (slog) setup
internal/platform/mongo  MongoDB connection lifecycle + index management
internal/transport/http  chi router, middleware, handlers, server lifecycle
deployments              docker-compose for local development
plans                    system plan and per-step implementation plans
```

Domain, service, and repository packages are introduced in later steps as the
features that need them are built.

## Configuration

All configuration is read from the environment (see [`.env.example`](./.env.example)).
In development a local `.env` file is loaded automatically; in production it is
ignored and values must come from the real environment.
