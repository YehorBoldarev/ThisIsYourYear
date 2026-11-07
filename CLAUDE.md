# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

A multi-user yearly-goals tracker. Users create **Topics** (big yearly goals); create
**Goals/milestones** under them; a completed Goal flagged `isAchievement` becomes a ★ win;
each Topic surfaces a motivational message derived from its live progress; a **dashboard**
aggregates progress and counts down to the New Year. Go + MongoDB backend, React SPA frontend.

## Commands

### Backend (from repo root — see `make help`)

```bash
make compose-up   # start MongoDB (docker compose) — required before running/integration tests
make run          # go run ./cmd/api  → http://localhost:8080
make test         # unit tests, race detector (go test -race ./...)
make test-int     # integration tests — needs Docker (testcontainers spins up Mongo)
make fmt vet lint tidy   # gofmt -w · go vet · golangci-lint · go mod tidy
make build        # backend-only binary → ./bin/api (no embedded SPA)
make build-all    # self-contained binary with SPA embedded (go build -tags embedui)
```

- **Single unit test:** `go test -run TestName ./internal/service/`
- **Single integration test:** `go test -tags=integration -run TestName ./internal/repository/`
- **Integration tests are tag-gated** (`//go:build integration`) so the default `go test`
  stays fast and Docker-free. They need a Docker daemon.
- **Backend + Mongo fully in Docker:** `docker compose -f deployments/docker-compose.yml --profile app up --build`.
- **Lint gotchas** (`.golangci.yml`): revive's `exported` rule is on, so exported Go identifiers
  need doc comments; `goimports` groups local imports under `kibou/thisisyouryear`.

### Frontend (from `web/`)

```bash
npm run dev          # Vite dev server :5173, proxies /api and /openapi.yaml → :8080
npm run build        # tsc -b + vite build → dist/
npm run typecheck    # tsc -b (strict, verbatimModuleSyntax)
npm run lint         # ESLint (incl. jsx-a11y)
npm run format:check # Prettier verify (format = write)
npm test             # Vitest run once   ·   npm run test:watch
npm run gen:api      # regenerate src/api/schema.d.ts from ../api/openapi.yaml
```

- **Single test:** `npm test -- src/features/topics/TopicDetailModal.test.tsx` or `npm test -- -t "name"`.
- **`gen:api` is not automatic:** after editing `api/openapi.yaml`, re-run it — `schema.d.ts`
  is generated **and committed**, and the whole typed client depends on it staying in sync.

## Backend architecture

Strict inward-pointing layering; each layer depends only on the ones toward `domain`.
Everything is wired by hand in `cmd/api/main.go`.

- **`internal/domain`** — entities + business rules, **no transport/storage imports**. Holds the
  sentinel errors (`ErrNotFound`, `ErrConflict`, `ErrValidation`, `ErrUnauthenticated`,
  `ErrInvalidCredentials`) that the rest of the app speaks in. Adapters translate to/from these
  at the edges.
- **`internal/service`** — use-cases. Each service **defines its own `…Repository` interface**
  (the ports it needs) and receives an implementation via its constructor. **Every repository
  method is scoped by `userID`** — ownership is enforced at the data layer, not by ad-hoc checks.
  Services build "view" structs (e.g. `TopicView` = topic + computed `Progress` + `Motivation`)
  so progress and its message stay in lockstep everywhere.
- **`internal/repository`** — MongoDB implementations of the service ports, built on the generic
  `Repository[T, PT]` base in `internal/platform/mongo` (CRUD/count/aggregate over one collection;
  entities embed `Document`). They register indexes on construction (`EnsureIndexes` runs after
  they're built).
- **`internal/auth`** — primitives the auth service composes: bcrypt passwords, HS256 JWT access
  tokens, and opaque refresh tokens persisted only as a SHA-256 hash (the `sessions` collection).
  `AuthService.Refresh` **rotates** the session on every use; presenting an already-rotated token
  is treated as theft and revokes *all* of that user's sessions.
- **`internal/transport/http`** — chi router, DTOs, handlers, middleware, server lifecycle.

Key cross-cutting conventions (each requires reading several files to see):

- **One error-mapping point:** `render.go` `respondError` is the *only* place domain errors →
  HTTP status + the stable envelope `{"error":{"code","message","fields?}}`. Handlers/middleware
  funnel every failure here; unclassified errors log server-side and return a bare 500. Add a new
  failure mode by mapping a domain sentinel here, not by writing statuses in handlers.
- **Partial updates use `domain.FieldUpdate[T]`** (`update.go`) — a tri-state that a pointer
  can't express: *omitted* = leave unchanged, *present+nil* = clear, *present+value* = set.
  On the wire this is: field absent = unchanged, `""` = clear, value = set. Dates are date-only
  at **UTC midnight** and formatted `DD-MM-YYYY`.
- **Router wiring** (`router.go`): ops endpoints (`/healthz`, `/readyz`, `/openapi.yaml`) live
  *outside* `/api/v1`; the public `/auth` group carries per-IP rate limiting when configured;
  everything else sits behind the `authenticator` middleware inside `api.Group`. Adding a feature =
  add a service + its repository interface + mount a sub-router here; existing groups are untouched.
  Nested goal routes reuse `{id}` as the *topic* id to avoid a chi wildcard-name conflict.
- **Config** (`internal/config`) is 100% env-driven (`caarlos0/env`) with dev-friendly defaults.
  In non-prod a local `.env` is loaded; in prod it's ignored. The insecure default `JWT_SECRET`
  is *rejected* when `APP_ENV=production`.

## Frontend architecture (`web/`)

React 19 + Vite + TypeScript (strict) + Tailwind CSS v4 + TanStack Query + react-router v7.

- **Typed API client** (`src/api/`): `openapi-fetch` over the generated `schema.d.ts`. The
  **access token lives in memory only** (`tokenStore.ts`) — never `localStorage` (XSS). The durable
  credential is the **HttpOnly refresh cookie**; `client.ts` does **single-flight refresh** on a 401
  and once on boot via raw `fetch` to `POST /api/v1/auth/refresh` (raw so it never re-enters the 401
  middleware). Requests are **same-origin**, so the cookie is sent without `credentials: 'include'`.
- **Server state** is TanStack Query; mutations are optimistic (update cache → rollback →
  settle-invalidate). Feature folders under `src/features/` (dashboard, topics, celebration,
  achievements). There is **no achievements endpoint** — wins are derived client-side by fanning
  `GET /topics/{id}/goals` and filtering `done && isAchievement`.
- **Motivation exists twice, and they differ.** The backend (`internal/domain/motivation.go`,
  six bands) computes the `motivation` string returned on every topic. The UI does **not** render
  that string: it derives its own copy from the progress ratio via `src/lib/motivation.ts` (four
  design bands), the single UI source used by dashboard rows and the topic-detail callout. To
  change what users see, edit the frontend file.
- **Tests** use MSW: `src/test/msw.ts` holds the shared server, fixtures (`TEST_USER`,
  `tokenResponse`, `dashboardResponse`, …) and the `errorBody` envelope helper. Override handlers
  per test with `server.use(...)`. `src/test/a11y.test.tsx` runs axe checks via `vitest-axe`.
- **Design reference:** `design_handoff_this_is_your_year/` holds the HTML mockups (dark + light
  themes, desktop/tablet/mobile). Recreate them in React; ignore their templating runtime
  (`support.js`).

## Production serving (single origin)

`make build-all` (or the multi-stage `Dockerfile`) copies `web/dist` into
`internal/transport/http/webui/dist` and compiles it into the binary via `//go:embed` behind the
**`embedui`** build tag. The SPA is mounted as chi's `NotFound` handler, so it never shadows API/ops
routes; unmatched `/api/**` still returns the JSON 404 envelope. One binary serves SPA + API from
one origin (no CORS). Default `go build`/`go test` stay backend-only. Strict CSP requires the
pre-paint theme script to stay external (`web/public/theme-init.js`), not inline.

For a **local non-technical demo**, root-level `START-HERE.txt` + `*-website.bat` +
`deployments/docker-compose.customer.yml` launch the whole stack by double-click (see `docs/dev_run.md` §6.1).

## Working conventions in this repo

- Development follows an approval-gated, per-step plan flow: backend plans live in `plans/`
  (`00-system-plan.md` + numbered steps), frontend plans in `plans/frontend/` (their own numbered
  track); implementation proceeds one step at a time. **After every implementation step, update `docs/dev_run.md`.**
- The OpenAPI contract (`api/openapi.yaml`) is the source of truth for the HTTP API and is embedded
  and served at `GET /openapi.yaml`; keep it and the generated frontend types in sync.
