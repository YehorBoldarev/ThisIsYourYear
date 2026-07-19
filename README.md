# ThisIsYourYear

A multi-user web application for tracking yearly goals. Users create big goals
(**Topics**) and concrete **Goals** beneath them; completing a Goal flagged as an
achievement becomes a ★ win, and each Topic surfaces a motivational message based on
its live progress. A dashboard shows overall status and a countdown to the New Year,
and reaching 100% on a Topic triggers a celebration.

The repository is a full-stack app: a **Go + MongoDB** backend and a **React** SPA that
the backend can serve from the same origin. See [`plans/`](./plans) for the system plan
and per-step implementation plans, and [`docs/dev_run.md`](./docs/dev_run.md) for the
detailed run/verify guide.

## Tech stack

- **Backend:** Go 1.26, `net/http` + [chi](https://github.com/go-chi/chi) router
- **Database:** MongoDB (official driver)
- **API:** REST/JSON under `/api/v1`; OpenAPI 3.1 contract at `GET /openapi.yaml`
- **Frontend:** React 19 + TypeScript + Vite + Tailwind CSS v4, TanStack Query,
  react-router v7 (in [`web/`](./web))

## Requirements

- Go 1.26+
- Node 20+ and npm 10+ (for the frontend)
- Docker (MongoDB via docker-compose; also required for integration tests; optional
  `golangci-lint` for linting)

## Getting started (development)

The backend and the SPA run as two processes in dev; the Vite dev server proxies `/api`
to the Go API.

```bash
# 1. Start MongoDB
make compose-up

# 2. (optional) configure local overrides
cp .env.example .env

# 3. Run the API  (http://localhost:8080)
make run

# 4. In another terminal, run the frontend  (http://localhost:5173, proxies /api → :8080)
cd web && npm install && npm run dev
```

Verify the API is up:

```bash
curl http://localhost:8080/healthz   # liveness  -> 200 {"status":"ok"}
curl http://localhost:8080/readyz    # readiness -> 200 when Mongo reachable, else 503
```

To run the backend + Mongo together in Docker (no local Go needed):

```bash
docker compose -f deployments/docker-compose.yml --profile app up --build
```

## Production (single origin)

In production one Go binary serves **both** the API and the built SPA from the same origin
(no CORS). The frontend build is embedded via `//go:embed` behind the `embedui` build tag.

```bash
make build-all        # build web/, stage it under the Go embed dir, compile the binary
./bin/api             # serves SPA + API on :8080
```

The multi-stage [`Dockerfile`](./Dockerfile) produces the same self-contained image. See
[`docs/dev_run.md`](./docs/dev_run.md) §6 for details.

### One-click local demo (non-technical Windows users)

Root-level `START-HERE.txt` and `start/stop/uninstall-website.bat` (backed by
`deployments/docker-compose.customer.yml`) let someone run the whole site by double-clicking
— the launcher installs Docker Desktop if needed. See `docs/dev_run.md` §6.1.

## Development

Backend (from the repo root — `make help` lists all targets):

```bash
make test      # unit tests (race detector)
make test-int  # integration tests (requires Docker; //go:build integration)
make fmt       # gofmt -w
make vet       # go vet
make lint      # golangci-lint
make tidy      # tidy go.mod
```

Frontend (from `web/`):

```bash
npm run typecheck     # tsc -b
npm run lint          # ESLint (incl. jsx-a11y)
npm run format:check  # Prettier verify
npm test              # Vitest
npm run build         # production build
npm run gen:api       # regenerate src/api/schema.d.ts from ../api/openapi.yaml (checked in)
```

After editing `api/openapi.yaml`, re-run `npm run gen:api` so the generated frontend types
stay in sync with the contract.

## Project layout

```
cmd/api                      process entrypoint and dependency wiring
internal/domain              entities + business rules + sentinel errors (no transport/storage deps)
internal/service             use-cases; each defines its own repository ports, all scoped by user
internal/repository          MongoDB implementations of the service ports
internal/config              environment configuration + validation
internal/platform/logging    structured logging (slog) setup
internal/platform/mongo      MongoDB connection lifecycle + index management
internal/transport/http      chi router, middleware, handlers, DTOs, server lifecycle
internal/transport/http/webui  embedded SPA assets (embedui build tag)
web                          React + TypeScript single-page app
api                          OpenAPI 3.1 contract (embedded and served)
deployments                  docker-compose files
docs                         run/verify guide
plans                        system plan and per-step implementation plans
```

## Configuration

All configuration is read from the environment (see [`.env.example`](./.env.example)).
In development a local `.env` file is loaded automatically; in production it is ignored and
values must come from the real environment. The insecure default `JWT_SECRET` is rejected
when `APP_ENV=production`.
