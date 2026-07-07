# Running ThisIsYourYear Locally

How to run the backend on your machine, exercise it, and shut it down cleanly.

> **Keep this current:** this document is updated at the end of every implementation
> step. It reflects the app **as of Step 5 (Goals & Achievements)**.

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

Available endpoints as of Step 5 (all JSON, base path `/api/v1`):

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
| `GET`  | `/healthz`, `/readyz` | — | liveness / readiness |

Topics and goals are **owned by the caller**: another user's id returns `404`. The topic list
is **year-scoped**, defaulting to the current year (`?year=YYYY` for others). A topic's
`progress` (`{total, completed, ratio}`) is computed from its goals; completing a goal
(`{"done":true}`) marks it an achievement and raises the ratio.

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
  -H 'Content-Type: application/json' -d '{"title":"Get fit","color":"#aabbcc","icon":"🏃"}' \
  | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

curl -s $BASE/topics -H "Authorization: Bearer $ACCESS"                 # list (current year)
curl -s -X PATCH $BASE/topics/$TID -H "Authorization: Bearer $ACCESS" \
  -H 'Content-Type: application/json' -d '{"title":"Get very fit"}'      # partial update

# --- goals under the topic ---
GID=$(curl -s -X POST $BASE/topics/$TID/goals -H "Authorization: Bearer $ACCESS" \
  -H 'Content-Type: application/json' -d '{"title":"Run 5k"}' \
  | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
curl -s $BASE/topics/$TID/goals -H "Authorization: Bearer $ACCESS"      # list goals
curl -s -X PATCH $BASE/goals/$GID -H "Authorization: Bearer $ACCESS" \
  -H 'Content-Type: application/json' -d '{"done":true}'                 # complete it
curl -s $BASE/topics/$TID -H "Authorization: Bearer $ACCESS"           # progress ratio rose

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

- **Step 5 — Goals & Achievements:** added goals under topics (create/list/edit/delete,
  mark done/undone, ordering), live topic progress, topic-delete cascade, the goal endpoint
  rows, and the goals smoke test.
- **Step 4 — Topics:** added user-owned Topics CRUD (year-scoped list, archive), the topics
  endpoint rows, and the topics smoke test.
- **Step 3 — Auth & Users:** added auth/user endpoints, JWT + refresh-cookie config vars,
  and the manual auth smoke test.
- **Step 1–2:** initial local-run, health checks, MongoDB via compose, unit/integration tests.
