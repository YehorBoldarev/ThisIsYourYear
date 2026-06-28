# Step 1 — Project Bootstrap & Tooling

> **Phase:** A (Backend) · **Status:** Draft for approval
> **Parent:** [`00-system-plan.md`](./00-system-plan.md) §8.1
> **Goal of this step:** A runnable, well‑structured Go service that loads config, connects
> to MongoDB, exposes `/healthz` + `/readyz`, shuts down gracefully, and has the full
> developer tooling (lint, format, test, Docker Compose) in place — so every later step
> drops into a clean, consistent foundation.
>
> **Explicitly *out* of scope here:** any domain logic, auth, or business endpoints.
> No `users`/`topics`/`goals`. Only the skeleton + plumbing + health checks.

---

## 1. Outcome / Definition of Done

- `docker compose up` starts MongoDB.
- `make run` (or `go run ./cmd/api`) starts the API, which connects to Mongo and logs a
  structured "server started" line.
- `GET /healthz` → `200` always (liveness, no dependencies).
- `GET /readyz` → `200` when Mongo ping succeeds, `503` otherwise (readiness).
- `Ctrl+C` / `SIGTERM` triggers **graceful shutdown** (stop accepting, drain, close Mongo).
- `make lint`, `make fmt`, `make test` all run and pass (test suite is minimal but real).
- `go vet ./...` and `golangci-lint run` are clean.
- README documents how to run everything.

---

## 2. Directory & File Layout Created in This Step

```
.
├── cmd/
│   └── api/
│       └── main.go                 # entrypoint: load config, build server, run, shutdown
├── internal/
│   ├── config/
│   │   ├── config.go               # env struct + Load()
│   │   └── config_test.go          # unit test: defaults + overrides + validation
│   ├── platform/
│   │   ├── logging/
│   │   │   └── logging.go          # slog setup (level, JSON/text by env)
│   │   └── mongo/
│   │       └── mongo.go            # Connect(), Ping(), Disconnect(), index hook (stub)
│   └── transport/
│       └── http/
│           ├── server.go           # http.Server wiring + graceful shutdown
│           ├── router.go           # chi router, middleware chain, route mounting
│           ├── health.go           # /healthz + /readyz handlers
│           ├── middleware.go       # requestID, logging, recover, timeout (thin wrappers)
│           └── health_test.go      # handler tests for health endpoints
├── deployments/
│   └── docker-compose.yml          # mongo service (+ api service, profile-gated)
├── .env.example                    # documented env vars (no secrets)
├── .gitignore
├── .golangci.yml                   # linter config
├── Dockerfile                      # multi-stage build for the api (used later/by compose)
├── Makefile                        # run, build, test, lint, fmt, tidy, compose targets
├── go.mod / go.sum
└── README.md                       # how to run, structure overview
```

> Folders for `domain/`, `service/`, `repository/`, `auth/` are intentionally **not** created
> yet — they arrive with the steps that need them, to avoid empty/placeholder packages.

---

## 3. Component Details

### 3.1 Configuration (`internal/config`)
- Single `Config` struct loaded from environment via `caarlos0/env`; `godotenv` loads `.env`
  **only in non‑production** for local dev.
- Fields (initial):
  - `AppEnv` (`development`|`production`, default `development`)
  - `HTTPPort` (default `8080`)
  - `HTTPReadTimeout`, `HTTPWriteTimeout`, `HTTPIdleTimeout`, `ShutdownTimeout` (sane defaults)
  - `MongoURI` (default `mongodb://localhost:27017`)
  - `MongoDatabase` (default `thisisyouryear`)
  - `LogLevel` (default `info`)
- `Load() (Config, error)` validates required values and returns a typed error on misconfig.
- **Test:** defaults applied; env overrides honored; invalid port/level rejected.

### 3.2 Logging (`internal/platform/logging`)
- `slog` logger: JSON handler in production, text handler in development; level from config.
- Returned and injected (no global logger reliance beyond a configured default).

### 3.3 MongoDB (`internal/platform/mongo`)
- `Connect(ctx, cfg) (*mongo.Client, *mongo.Database, error)` with connection + ping timeout.
- `Ping(ctx)` helper used by `/readyz`.
- `Disconnect(ctx)` for graceful shutdown.
- An `EnsureIndexes(ctx)` **stub/hook** that does nothing yet but establishes the pattern
  later steps extend (no collections defined this step).

### 3.4 HTTP transport (`internal/transport/http`)
- `router.go`: chi router with middleware chain in order:
  `RequestID → RealIP → structured access log → Recoverer → Timeout`.
  CORS and auth middleware are deliberately deferred to the steps that need them.
- `health.go`:
  - `/healthz`: always `200 {"status":"ok"}` (no dependency checks — liveness).
  - `/readyz`: pings Mongo; `200` ready / `503` not ready (readiness).
- `server.go`: builds `*http.Server` with timeouts from config; `Run(ctx)` starts listening;
  on context cancel performs `Shutdown` within `ShutdownTimeout`, then closes Mongo.

### 3.5 Entrypoint (`cmd/api/main.go`)
- Wire order: load config → build logger → connect Mongo → build router → build server →
  run until `SIGINT`/`SIGTERM` → graceful shutdown. Non‑zero exit on fatal startup error.

---

## 4. Tooling & Ops Files

### 4.1 `deployments/docker-compose.yml`
- `mongo` service: official `mongo:7` image, named volume for data, healthcheck, port `27017`.
- `api` service: built from `Dockerfile`, **behind a compose profile** (e.g. `--profile app`)
  so day‑to‑day dev can run Mongo in Docker but the API via `go run` for fast iteration.
- No credentials hard‑coded; dev defaults only, documented in `.env.example`.

### 4.2 `Dockerfile`
- Multi‑stage: `golang:1.26` build stage → minimal `distroless`/`alpine` runtime, non‑root
  user, only the compiled binary shipped.

### 4.3 `Makefile` targets
`run`, `build`, `test`, `test-int` (integration, opt‑in), `lint`, `fmt`, `vet`, `tidy`,
`compose-up`, `compose-down`. Self‑documenting `help` default.

### 4.4 `.golangci.yml`
- Enable a sensible, non‑noisy set: `govet, staticcheck, errcheck, ineffassign, revive,
  gofmt, goimports, misspell, gosec` (tunable). Fail CI on findings.

### 4.5 `.gitignore` / `.env.example`
- Ignore binaries, `.env`, coverage output, IDE folders.
- `.env.example` lists every variable with safe defaults and comments.

---

## 5. Testing in This Step

- **Unit:** `config_test.go` (load/defaults/validation), `health_test.go` (`/healthz` 200;
  `/readyz` 200 and 503 via an injected pinger fake — no real Mongo needed for the unit path).
- **No testcontainers yet** — the real‑Mongo integration harness lands in Step 2 where repos
  exist. `/readyz` against real Mongo is verified manually via `make run` + `curl`.
- All tests run with `make test`; `-race` enabled.

---

## 6. Dependencies Added (go.mod)

`go-chi/chi/v5`, `go-chi/chi/v5/middleware`, `go.mongodb.org/mongo-driver`,
`caarlos0/env/v11`, `joho/godotenv`, `stretchr/testify` (test only).
(`golang-jwt`, `bcrypt`, `validator`, `testcontainers` are added in the steps that first use
them, to keep `go.mod` honest.)

---

## 7. Verification Checklist (how I'll prove it works before "done")

1. `make tidy && make build` — compiles clean.
2. `make fmt && make vet && make lint` — all clean.
3. `make test` — green, with `-race`.
4. `make compose-up` then `make run` → logs show Mongo connected + server started.
5. `curl /healthz` → 200; `curl /readyz` → 200; stop Mongo → `/readyz` → 503.
6. `Ctrl+C` → logs show graceful shutdown, process exits 0.
7. I'll paste the relevant command output back to you as evidence.

---

## 8. Decisions Within This Step (my discretion — flag if you disagree)

- MongoDB image pinned to `mongo:7`.
- Default local port `8080` for API, `27017` for Mongo.
- `distroless` runtime base for the image.
- Linter set listed in §4.4.

No system‑wide decisions are introduced by this step (all align with the approved
`00-system-plan.md`).

---

## 9. What This Unblocks

Step 2 (Persistence foundation) plugs into: the `mongo` package (real `EnsureIndexes`),
the config struct, the test harness pattern, and the router (mounting future route groups).
