# Running ThisIsYourYear Locally

How to run the backend on your machine, exercise it, and shut it down cleanly.

> **Keep this current:** this document is updated at the end of every implementation
> step. It reflects the app **as of Step 9 (Design data extensions)**.

---

## 1. Prerequisites

| Tool | Version | Needed for |
| --- | --- | --- |
| Go | 1.26.x (see `go.mod`) | building/running the API, unit tests |
| Docker | any recent | MongoDB, integration tests, containerized run |
| `golangci-lint` | v1.64.x | `make lint` (optional but recommended) |
| `mongosh` | optional | inspecting the database directly |

All commands below are run from the repository root. `make help` lists every target.

---

## 2. Run the app locally

The default workflow runs **MongoDB in Docker** and the **API on the host** via `go run`,
so the API is fast to iterate on.

### 2.1 Start MongoDB

```bash
make compose-up          # starts the `mongo` service in the background
```

This runs `mongo:7` on `localhost:27017` with a persistent `mongo-data` volume. It is
healthy once `docker ps` shows the `tiyy-mongo` container as `healthy`.

### 2.2 Start the API

```bash
make run                 # == go run ./cmd/api
```

The service listens on `http://localhost:8080`. With zero configuration it connects to the
Mongo instance above, ensures indexes, and logs `connected to mongo`.

### 2.3 Verify it's up

```bash
curl -s localhost:8080/healthz   # {"status":"ok"}    liveness
curl -s localhost:8080/readyz    # {"status":"ready"} readiness (pings Mongo)
```

If Mongo is down, `/readyz` returns `503` within ~2s while `/healthz` stays `200`.

### 2.4 Configuration (environment variables)

Everything is 12-factor env config with sensible development defaults; nothing is required
for a local run. Common overrides:

| Variable | Default | Notes |
| --- | --- | --- |
| `APP_ENV` | `development` | `production` enables JSON logging and stricter validation |
| `HTTP_PORT` | `8080` | API port |
| `MONGO_URI` | `mongodb://localhost:27017` | |
| `MONGO_DATABASE` | `thisisyouryear` | |
| `LOG_LEVEL` | `info` | `debug`/`info`/`warn`/`error` |
| `JWT_SECRET` | dev placeholder | **must** be set to a strong value in production |
| `ACCESS_TOKEN_TTL` | `15m` | access-token lifetime |
| `REFRESH_TTL_REMEMBER` | `744h` (31d) | refresh session TTL with "remember me" |
| `REFRESH_TTL_DEFAULT` | `24h` | refresh session TTL without "remember me" |
| `BCRYPT_COST` | `12` | password hashing cost (10–31) |
| `COOKIE_SECURE` | `true` | **set `false` for local HTTP** so the refresh cookie is sent over `http://` |
| `COOKIE_DOMAIN` | empty | cookie domain; empty = host-only |
| `CORS_ALLOWED_ORIGINS` | empty | comma-separated SPA origins (no `*`, credentials are used) |
| `RATE_LIMIT_AUTH_REQUESTS` | `20` | max auth requests per window per client IP; **`0` disables** rate limiting |
| `RATE_LIMIT_AUTH_WINDOW` | `1m` | the rate-limit window duration |

Example with overrides:

```bash
APP_ENV=development COOKIE_SECURE=false LOG_LEVEL=debug make run
```

> **Tip:** set `COOKIE_SECURE=false` when you want to test the refresh flow locally with a
> browser or `curl` cookie jar — a `Secure` cookie is not sent over plain HTTP.

### 2.5 Alternative: run everything in Docker

To run the API itself in a container instead of on the host (uses the multi-stage
`Dockerfile`):

```bash
docker compose -f deployments/docker-compose.yml --profile app up --build
```

This starts Mongo and the API together; the API waits for Mongo to be healthy.

---

## 3. Testing

### 3.1 Automated tests

```bash
make test        # unit tests, race detector, no Docker needed
make test-int    # integration tests against a real mongo:7 (testcontainers; needs Docker)
```

`make test-int` spins up throwaway Mongo containers, so Docker must be running. It covers the
repositories and the **full HTTP auth flow** (register → login → /me → refresh rotation →
reuse detection → logout).

### 3.2 Static checks

```bash
make fmt         # gofmt -w .
make vet         # go vet
make lint        # golangci-lint
```

### 3.3 Manual API smoke test

Available endpoints as of Step 9 (all JSON, base path `/api/v1`):

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| `POST` | `/api/v1/auth/register` | — | create an account |
| `POST` | `/api/v1/auth/login` | — | log in; sets refresh cookie, returns access token |
| `POST` | `/api/v1/auth/refresh` | refresh cookie | rotate & renew tokens |
| `POST` | `/api/v1/auth/logout` | refresh cookie | revoke session, clear cookie |
| `GET`  | `/api/v1/me` | Bearer access token | current user profile |
| `POST` | `/api/v1/topics` | Bearer | create a topic |
| `GET`  | `/api/v1/topics` | Bearer | list my topics (`?year=YYYY`, `?includeArchived=true`) |
| `GET`  | `/api/v1/topics/{id}` | Bearer | get one of my topics (+ computed progress) |
| `PATCH`| `/api/v1/topics/{id}` | Bearer | update (partial; `{"archived":true}` to archive) |
| `DELETE`| `/api/v1/topics/{id}` | Bearer | delete a topic (cascades its goals) |
| `POST` | `/api/v1/topics/{id}/goals` | Bearer | add a goal to a topic |
| `GET`  | `/api/v1/topics/{id}/goals` | Bearer | list a topic's goals (ordered) |
| `PATCH`| `/api/v1/goals/{id}` | Bearer | update a goal (`{"done":true}` to complete; `order` to reorder) |
| `DELETE`| `/api/v1/goals/{id}` | Bearer | delete a goal |
| `GET`  | `/api/v1/dashboard` | Bearer | year summary: aggregate progress, message, New-Year countdown |
| `GET`  | `/openapi.yaml` | — | the OpenAPI 3.1 contract (served as `application/yaml`) |
| `GET`  | `/healthz`, `/readyz` | — | liveness / readiness |

Topics and goals are **owned by the caller**: another user's id returns `404`. The topic list
is **year-scoped**, defaulting to the current year (`?year=YYYY` for others). A topic's
`progress` (`{total, completed, ratio}`) is computed from its goals; completing a goal
(`{"done":true}`) marks it an achievement and raises the ratio.

Each topic response also carries a `motivation` string — an encouraging message derived from
its live progress (no goals → "set your first goal…", partway → momentum copy, 100% → a
celebratory 🎉). It is computed, never stored, so it always matches the current ratio.

Topics and goals also carry optional **design fields** (added for the frontend): a topic has a
`category` and a `targetDate`; a goal (milestone) has a `dueDate` and an `isAchievement` flag (the
"celebrate as an achievement ★" toggle). **Dates use `DD-MM-YYYY`** (e.g. `"31-12-2026"`) — a
calendar date, no time-of-day. On `PATCH`, a date is left unchanged when omitted, **cleared with an
empty string** `""`, or set with a `DD-MM-YYYY` value; an invalid format returns `422`.

`GET /dashboard` is the landing summary for the caller's **current-year, non-archived** topics:
aggregate `progress` (`{total, completed, ratio}` summed across all goals), a `topics`
tally (`{total, completed}`), a positive `message` (a motivational empty-state line when there
are no topics yet, otherwise the progress-derived band), and a `countdown` to the next New Year
(`target` in UTC + `secondsRemaining`; the SPA localizes the target to the browser timezone).

`GET /openapi.yaml` serves the checked-in **OpenAPI 3.1** contract (`api/openapi.yaml`, embedded
in the binary) as `application/yaml` — the source of truth for the API shape and the basis for
generating the React client. It needs no authentication.

The public **auth endpoints** (`/auth/register`, `/auth/login`, `/auth/refresh`, `/auth/logout`)
are **rate limited per client IP** (default 20 requests / minute; see `RATE_LIMIT_AUTH_*` above).
Exceeding the limit returns `429` with `{"error":{"code":"rate_limited",...}}` and a `Retry-After`
header (whole seconds until the window resets). Set `RATE_LIMIT_AUTH_REQUESTS=0` to disable it.
Health probes are never throttled. The limiter keys on the direct connection IP (not
`X-Forwarded-For`); behind a trusted proxy this should be revisited.

Run the API with `COOKIE_SECURE=false` first, then:

```bash
BASE=localhost:8080/api/v1
JAR=$(mktemp)

# register
curl -s -X POST $BASE/auth/register -H 'Content-Type: application/json' \
  -d '{"email":"you@example.com","displayName":"You","password":"password123"}'

# login (rememberMe) — stores the HttpOnly refresh cookie in $JAR
ACCESS=$(curl -s -c "$JAR" -X POST $BASE/auth/login -H 'Content-Type: application/json' \
  -d '{"email":"you@example.com","password":"password123","rememberMe":true}' \
  | grep -o '"accessToken":"[^"]*"' | cut -d'"' -f4)

# call a protected endpoint with the access token
curl -s $BASE/me -H "Authorization: Bearer $ACCESS"

# --- topics (need a fresh access token; refresh above rotated it) ---
ACCESS=$(curl -s -c "$JAR" -X POST $BASE/auth/login -H 'Content-Type: application/json' \
  -d '{"email":"you@example.com","password":"password123"}' \
  | grep -o '"accessToken":"[^"]*"' | cut -d'"' -f4)

# create a topic (year defaults to the current year)
TID=$(curl -s -X POST $BASE/topics -H "Authorization: Bearer $ACCESS" \
  -H 'Content-Type: application/json' \
  -d '{"title":"Get fit","color":"#aabbcc","icon":"🏃","category":"Health","targetDate":"31-12-2026"}' \
  | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

curl -s $BASE/topics -H "Authorization: Bearer $ACCESS"                 # list (current year)
curl -s -X PATCH $BASE/topics/$TID -H "Authorization: Bearer $ACCESS" \
  -H 'Content-Type: application/json' -d '{"title":"Get very fit"}'      # partial update

# --- goals under the topic ---
GID=$(curl -s -X POST $BASE/topics/$TID/goals -H "Authorization: Bearer $ACCESS" \
  -H 'Content-Type: application/json' -d '{"title":"Run 5k","dueDate":"09-08-2026","isAchievement":true}' \
  | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
curl -s $BASE/topics/$TID/goals -H "Authorization: Bearer $ACCESS"      # list goals
curl -s -X PATCH $BASE/goals/$GID -H "Authorization: Bearer $ACCESS" \
  -H 'Content-Type: application/json' -d '{"done":true}'                 # complete it
curl -s $BASE/topics/$TID -H "Authorization: Bearer $ACCESS"           # progress ratio rose;
                                                                       # "motivation" reflects it

# --- dashboard: aggregate summary + New-Year countdown ---
curl -s $BASE/dashboard -H "Authorization: Bearer $ACCESS"             # message, progress, topics, countdown

# --- the API contract (no auth) ---
curl -s localhost:8080/openapi.yaml | head -3                          # openapi: 3.1.0 ...

curl -s -X DELETE $BASE/topics/$TID -H "Authorization: Bearer $ACCESS" \
  -o /dev/null -w '%{http_code}\n'                                       # 204 (cascades goals)

# log out
curl -s -b "$JAR" -c "$JAR" -X POST $BASE/auth/logout -o /dev/null -w '%{http_code}\n'  # 204
```

Errors use a stable envelope: `{"error":{"code":"...","message":"...","fields":{...}}}`
(`422` validation, `401` auth, `404` not found, `409` conflict).

---

## 4. Shutdown

### 4.1 Stop the API

Press **Ctrl+C** in the terminal running `make run`. The server shuts down gracefully:
it stops accepting new connections, drains in-flight requests (up to `SHUTDOWN_TIMEOUT`,
default 15s), then disconnects from Mongo.

If running via the Docker `app` profile, stop it with **Ctrl+C** or
`docker compose -f deployments/docker-compose.yml --profile app down`.

### 4.2 Stop MongoDB

```bash
make compose-down        # stops and removes the dev containers
```

Data persists in the `mongo-data` volume across restarts. To wipe the database entirely:

```bash
docker compose -f deployments/docker-compose.yml down -v   # also removes the volume
```

### 4.3 Clean build artifacts (optional)

```bash
rm -rf bin/             # created by `make build`
```

---

## Changelog

- **Step 9 — Design data extensions:** added the optional fields the design handoff needs —
  Topic `category` + `targetDate`, Goal `dueDate` + `isAchievement` — settable on create and
  `PATCH` (omit = unchanged, `""` = clear, `DD-MM-YYYY` = set), returned in responses, and
  documented in `api/openapi.yaml`. Additive and backward-compatible; no new endpoints.

- **Step 8 — Hardening:** added per-IP rate limiting on the public auth endpoints
  (`RATE_LIMIT_AUTH_*`, `429` + `Retry-After`, `0` disables); published the checked-in
  **OpenAPI 3.1** contract (`api/openapi.yaml`, embedded and served at `GET /openapi.yaml`);
  and locked the error contract with an end-to-end suite covering `401/404/409/422/429/405`.
- **Step 7 — Dashboard:** added `GET /dashboard` — the year's aggregate progress across
  current, non-archived topics, a topic tally, a positive message (motivational empty state
  when there are no topics), and a New-Year countdown (UTC target + seconds). Composes the
  existing topic-list path; no new persistence.
- **Step 6 — Motivation Engine:** each topic response now carries a computed
  `motivation` message derived from its live progress (six bands from "no goals yet"
  to a celebratory 100%); no new endpoints or dependencies.
- **Step 5 — Goals & Achievements:** added goals under topics (create/list/edit/delete,
  mark done/undone, ordering), live topic progress, topic-delete cascade, the goal endpoint
  rows, and the goals smoke test.
- **Step 4 — Topics:** added user-owned Topics CRUD (year-scoped list, archive), the topics
  endpoint rows, and the topics smoke test.
- **Step 3 — Auth & Users:** added auth/user endpoints, JWT + refresh-cookie config vars,
  and the manual auth smoke test.
- **Step 1–2:** initial local-run, health checks, MongoDB via compose, unit/integration tests.
