# Running ThisIsYourYear Locally

How to run the backend on your machine, exercise it, and shut it down cleanly.

> **Keep this current:** this document is updated at the end of every implementation
> step. It reflects the app **as of Step C2 (CI/CD pipeline)**.
> Backend is complete through Step 9; the frontend (Phase B) lives in `web/` — see
> [§5 Frontend](#5-frontend-phase-b) and [`web/README.md`](../web/README.md). Production
> serving is §6 and continuous integration is [§7](#7-continuous-integration-github-actions).

---

## 1. Prerequisites

| Tool | Version | Needed for |
| --- | --- | --- |
| Go | 1.26.x (see `go.mod`) | building/running the API, unit tests |
| Node | 24 (see `web/.nvmrc`), npm 11 | the frontend (`web/`) |
| Docker | any recent | MongoDB, integration tests, containerized run |
| `golangci-lint` | v2.15.0 — **no install needed** | `make lint` builds the pinned version into `./bin` on first run |
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
make fmt         # go fmt ./...  (skips web/node_modules, see below)
make vet         # go vet, twice: -tags=integration and -tags=embedui (covers every file)
make lint        # golangci-lint v2.15.0 (pinned), same two tag passes
```

- **Pinned linter:** `make lint` builds golangci-lint at `GOLANGCI_LINT_VERSION` (Makefile) into
  `./bin/golangci-lint-<version>` the first time (about a minute), then reuses it. CI pins the same
  version in `.github/workflows/ci.yml`; bump both together. The config (`.golangci.yml`) is in
  the **v2 format** with no exclusion presets, so revive's doc-comment rule on exported
  identifiers is enforced; deliberate exceptions are inline `//nolint:<linter> // reason`.
- **`web/node_modules` is ignored by Go:** an npm package ships a stray `.go` file, which made
  `./...` pick it up. `go.mod` has `ignore ./web/node_modules` (Go 1.25+), so `go test/vet/fmt
  ./...` and the linter skip it.

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

Each topic response also carries a `motivation` object — `{band, label, title, message}` —
derived from its live progress using the design's four bands: `not-started` (no goals, or none
done), `in-progress` (below 80%), `almost-there` (80% up to 100%), `achieved` (100%). It is
computed, never stored, so it always matches the current ratio. The copy lives in one table in
`internal/domain/motivation.go`; the SPA renders it as-is and only maps `band` to a colour.

Topics and goals also carry optional **design fields** (added for the frontend): a topic has a
`category` and a `targetDate`; a goal (milestone) has a `dueDate` and an `isAchievement` flag (the
"celebrate as an achievement ★" toggle). **Dates use `DD-MM-YYYY`** (e.g. `"31-12-2026"`) — a
calendar date, no time-of-day. On `PATCH`, a date is left unchanged when omitted, **cleared with an
empty string** `""`, or set with a `DD-MM-YYYY` value; an invalid format returns `422`.

`GET /dashboard` is the landing summary for the caller's **current-year, non-archived** topics:
aggregate `progress` (`{total, completed, ratio}` summed across all goals), a `topics`
tally (`{total, completed}`), a positive `message` (a motivational empty-state line when there
are no topics yet, otherwise the overall progress band's `message`), and a `countdown` to the next New Year
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

## 5. Frontend (Phase B)

The React SPA lives in `web/` and talks to this API. Full instructions are in
[`web/README.md`](../web/README.md); the short version:

```bash
cd web
npm install
npm run dev        # http://localhost:5173 — proxies /api and /openapi.yaml to :8080
```

As of **Step B6** the frontend registers, logs in, stays logged in across reloads
(silent refresh via the HttpOnly cookie), renders the **dashboard** from live data
(`GET /dashboard` + `GET /topics`), and is **interactive**: clicking a chapter opens
the topic-detail popup (progress ring + milestone checklist), and the create/edit/
delete popups drive full CRUD for chapters (`Topic`) and milestones (`Goal`) — create
a big goal, add a milestone (with the "Celebrate as an achievement ★" flag), toggle a
milestone done/undone, edit, and delete (with an inline confirm). Mutations apply
**optimistic updates** (rings, percentages and the dashboard aggregate move
immediately) and reconcile from the server on settle. Completing the **last** milestone
of a chapter fires the **100% celebration** (medallion, confetti — skipped under
`prefers-reduced-motion` — a congratulatory quote, and a milestones/duration/rank stat
strip). The **Wins** tab (`/achievements`) derives its list client-side from completed,
starred milestones across the year's chapters. The **four-band motivation copy** comes from
the API (`topic.motivation`, plus the dashboard `message`); `lib/motivation` only maps each band
to its tone colour, so the greeting, chapter rows and topic-detail callout speak in one voice — there is no achievements endpoint; the Wins page fans out `GET /topics/{id}/goals`.
B6 hardened accessibility: a **skip-to-content** link, `aria-current` on the active nav,
`role="alert"` on form/confirm errors, a global **`prefers-reduced-motion`** guard (on top
of the JS Confetti gate), and an automated **axe** gate over the key screens.
Run the backend first (`make compose-up && make run`), then `cd web && npm run dev` —
the dev proxy makes the browser see one origin, so the refresh cookie flows without
CORS. The dev-only component gallery is at **/gallery**. Quality gates mirror the
backend: `npm run typecheck`, `npm run gen:api` (regenerates the typed client; should
be a no-op diff), `npm run lint`, `npm run format:check`, `npm test`, `npm run build`.

### 5.1 Accessibility & cross-theme/breakpoint QA checklist

Run through this after frontend changes (both **dark** and **light**, at **desktop /
768 / 390**). Automated coverage: `npm test` includes axe assertions (login, populated
dashboard, Wins, an open modal) plus cross-theme AppShell render tests.

- **Keyboard:** Tab from page top reveals the **Skip to content** link → focuses `#main`;
  Tab order is logical; every control has a visible focus ring in both themes.
- **Modals:** focus moves in on open, is trapped, returns to the opener on close; **Esc**
  and backdrop close; this holds for topic-detail, the forms, confirm-delete, and the
  celebration.
- **Nav:** the active item carries `aria-current="page"` (header + bottom tab bar).
- **Reduced motion:** with the OS "reduce motion" setting on, skeleton pulses/hover
  fades are neutralized and the celebration shows **no** confetti.
- **States:** each of dashboard / Wins / topic-detail shows loading, error (with retry),
  and empty; form and confirm errors are announced (`role="alert"`).
- **Contrast:** enforced by `src/styles/contrast.test.ts` (WCAG AA for every text token on
  every surface, both themes). In the browser, spot-check that accent-coloured **text** uses
  the `-ink` utilities (`text-teal-ink`, `text-gold-ink`, `text-blue-ink`) — the plain accent
  utilities are for fills/strokes/borders only — and that `text-faint` isn't used for text.

---

## 6. Production build & serving (Phase C)

In production the Go binary serves **both** the API and the built React SPA from the
**same origin**, so there is no CORS and the HttpOnly refresh cookie flows normally. Dev
is unchanged (Vite dev server + `/api` proxy).

```bash
make build-all        # builds web/ , stages it under the Go embed dir, and
                      # compiles a self-contained binary (go build -tags embedui)
./bin/api             # serves the SPA + API on :8080
```

- **How it works:** the frontend build (`web/dist`) is copied into
  `internal/transport/http/webui/dist` and compiled into the binary via
  `//go:embed` behind the **`embedui`** build tag. The default `make build` /
  `go build` / `go test` stay backend-only and need no built frontend. The SPA is
  mounted as the router's `NotFound` handler, so it never shadows the API/ops routes.
- **Behaviour:** `GET /` and unknown non-API GETs serve `index.html` (SPA client-side
  routing / deep links); `/assets/*` (content-hashed) are served `immutable`, `index.html`
  is `no-cache`; unmatched `/api/**` still returns the JSON 404 envelope; `/healthz`,
  `/readyz`, `/openapi.yaml` are unaffected. Responses carry baseline security headers
  and a strict **CSP** (`default-src 'self'`, `script-src 'self'`) — the theme bootstrap
  moved to an external `/theme-init.js` so no inline script is needed.
- **Docker:** the `Dockerfile` is multi-stage (Node build → Go build with
  `-tags embedui` → distroless). One image serves everything on `:8080`. It builds for
  **`linux/amd64` and `linux/arm64`** (Raspberry Pi): the build stages run on the builder's own
  platform and Go cross-compiles, so
  `docker buildx build --platform linux/amd64,linux/arm64 .` needs no emulation. A plain
  `docker build .` builds for the local machine. `.dockerignore` keeps `.git`, `.env` files and
  the host's `node_modules`/build output out of the build context.
- **Prod config:** leave `CORS_ALLOWED_ORIGINS` **empty** (same origin), keep
  `COOKIE_SECURE=true` (needs TLS), and set a real `JWT_SECRET`. TLS termination / a
  reverse proxy / hosting are a later Phase-C concern.

### 6.1 One-click launch for a non-technical customer

For sharing a **local** demo with someone who doesn't use the terminal, the repo root
carries a self-contained Windows launcher (double-click, no CLI):

- `START-HERE.txt` — plain-language instructions (download ZIP → unzip → run).
- `start-website.bat` — ensures Docker Desktop is present (auto-installs via `winget`
  if missing), starts the engine, then `docker compose up -d --build`, waits on
  `/healthz`, and opens `http://localhost:8080`.
- `stop-website.bat` — `compose down` (keeps the `mongo-data` volume).
- `uninstall-website.bat` — `compose down -v --rmi local` (deletes data + built image;
  confirmation-gated).
- `deployments/docker-compose.customer.yml` — mongo + api (built from the `Dockerfile`),
  project name `thisisyouryear`, `APP_ENV=development` so it runs with zero config, and
  **`COOKIE_SECURE=false`** because the demo is served over plain `http://localhost`
  (a `Secure` cookie would be dropped and break sign-in). Mongo's port is **not**
  published — only the api container reaches it over the compose network.

This path is a **local single-machine demo**, distinct from the production guidance above
(which keeps `COOKIE_SECURE=true` behind TLS). Nothing here changes the app or dev flow.

---

## 7. Continuous integration (GitHub Actions)

`.github/workflows/ci.yml` runs on every push and pull request to **`master`**, on `v*` tags,
and on demand (Actions tab → CI → *Run workflow*). Design: `plans/frontend/08-ci-cd-pipeline.md`.

| Job | What it checks | Reproduce locally |
| --- | --- | --- |
| `backend` | gofmt · `go mod tidy` leaves no diff · `go vet` · golangci-lint · unit tests | `make fmt vet lint test tidy`, then `git diff` shows nothing |
| `backend-integration` | integration tests (testcontainers MongoDB) | `make test-int` |
| `frontend` | `npm ci` · typecheck · ESLint · Prettier · Vitest · build · `schema.d.ts` matches `api/openapi.yaml` | in `web/`: `npm ci && npm run typecheck && npm run lint && npm run format:check && npm test && npm run build && npm run gen:api`, then `git diff src/api/schema.d.ts` shows nothing |
| `docker` | runs after the three above; builds the image for amd64 + arm64 | `docker buildx build --platform linux/amd64,linux/arm64 .` |

- **Publishing:** on `master` and `v*` tags the `docker` job pushes to
  **`ghcr.io/yehorboldarev/thisisyouryear`**. Pull requests build but never push.
  - **`:latest`**: the newest `master` build.
  - **`:sha-<short commit>`**: every published build, so it doubles as a rollback handle.
  - **`:X.Y.Z` and `:X.Y`**: added when you push a tag like `v1.2.3`. A version tag never moves
    `:latest`.
- **No secrets to configure:** the workflow authenticates to GHCR with the built-in
  `GITHUB_TOKEN`.
- **Concurrency:** a new push to a PR cancels that PR's previous run; runs on `master` always
  finish, so every merged commit gets an image.
- **Dependabot** (`.github/dependabot.yml`) opens one grouped PR a week when the pinned GitHub
  Actions have new versions. Go and npm updates stay manual.

### 7.1 One-time GitHub settings (after the first green run)

1. **Image visibility.** The package is already **public**: an anonymous pull works (checked
   2026-11-11). To check or change it, open
   *github.com/YehorBoldarev* → *Packages* tab → **thisisyouryear**. The badge next to the name
   says *Public*/*Private*. *Package settings* (right sidebar) → *Danger Zone* → *Change
   visibility*.
   - **Verify from any machine without logging in:**
     `docker buildx imagetools inspect ghcr.io/yehorboldarev/thisisyouryear:latest`.
     It lists `linux/amd64` and `linux/arm64`.
2. **Protect `master` with a ruleset.** Go to *Settings* → *Rules* → *Rulesets* → *New ruleset* →
   *New branch ruleset*:
   - Name `master`; *Enforcement status* **Active**.
   - *Targets* → *Add target* → **Include default branch**.
   - Keep **Restrict deletions** and **Block force pushes** (on by default).
   - **Require a pull request before merging**, with *Required approvals* **0**. You're the
     only developer, and GitHub won't let you approve your own PR.
   - **Require status checks to pass** → *Add checks* → **`backend`**,
     **`backend-integration`**, **`frontend`**, **`docker`**. `docker` makes sure a PR can't
     break the image build; checks are listed once they've run in the last week.
   - *Create*.
   - **Effect:** `git push origin master` is now **rejected**. Work on a branch, push it, open a
     PR, and merge when the four checks are green; the merge then publishes `:latest`. To keep
     an emergency direct-push escape hatch, add *Repository admin* to the ruleset's *Bypass list*.

---

## Changelog

- **Step C2 — CI/CD pipeline (GitHub Actions):**
  - **Workflow:** `.github/workflows/ci.yml` with four jobs.
    - `backend`, `backend-integration` and `frontend` run in parallel.
    - `docker` runs after them and builds a multi-arch (amd64 + arm64) image.
    - On `master` and `v*` tags, `docker` pushes to `ghcr.io/yehorboldarev/thisisyouryear`
      (`:latest`, `:sha-…`, semver).
  - **Dependabot:** grouped weekly updates for GitHub Actions.
  - **golangci-lint v2:**
    - `.golangci.yml` migrated to the v2 format; no exclusion presets.
    - v2.15.0 pinned in the Makefile and CI; `make lint` builds it into `./bin`.
    - `make vet`/`lint` cover both build-tag sets.
    - The v2 findings were fixed: doc comments on `EnvDevelopment`/`EnvProduction` and the
      `GoalStatus` constants, and a documented `//nolint:gosec` (G124) on the refresh cookie,
      whose `Secure` flag is config-driven. The `RefreshCookieConfig` comment no longer claims
      production can't turn `Secure` off.
  - **Node 24:** the Dockerfile and CI use Node 24, and a new `web/.nvmrc` pins it.
  - **Dockerfile:** the build stages run on `$BUILDPLATFORM` and Go cross-compiles to the target.
  - **Fixes found while verifying:**
    - **Docker build failed at `npm ci`.** The Dockerfile didn't copy `web/.npmrc`
      (`legacy-peer-deps`), so the TypeScript 6 / openapi-typescript peer conflict failed the
      install. It's copied now.
    - **Go tooling scanned `web/node_modules`.** `./...` matched a stray Go package inside it, so
      `go.mod` gained `ignore ./web/node_modules` and `make fmt` uses `go fmt`.
    - **No `.dockerignore`.** Without one, `.git`, `.env` and the host's `node_modules` went into
      the build context; the new `.dockerignore` keeps them out.
  - **Verified locally:**
    - gofmt (empty), `go mod tidy` (no diff), `go vet` on both tag sets, golangci-lint v2.15.0
      on both tag sets (0 issues), `make test`, `make test-int`;
    - web typecheck, ESLint, Prettier, Vitest (178), build, `gen:api` (no diff);
    - actionlint 1.7.12 with shellcheck (clean);
    - plain `docker build` plus a runtime smoke test (`/healthz`, `/readyz`, `/`, deep link,
      JSON 404);
    - `docker buildx` for amd64 + arm64 (the arm64 binary is a static aarch64 ELF);
    - both compose builds.
  - **Verified on GitHub (2026-11-11):** all four jobs green on the first run, and the published
    image is public and multi-arch.

- **Fix — Mobile "New ＋" tab did nothing:** the tab bar lives in the app shell, but the topic
  popups were owned by a `ModalHost` inside the Dashboard page, so the shell passed it a no-op.
  `ModalHost` now sits once above the routes (`App.tsx`), making the topic popups app-level; the
  tab opens "Create a big goal" over the dashboard from any page (navigating there first, so the
  saved chapter is visible). `AppShell` takes an `onNew` prop. Added an app-level test (Wins →
  tap New → create popup over the dashboard), confirmed to fail on the old no-op. Also started
  `plans/backlog.md` for postponed features (year rollover, Reflections, reordering, …) and known
  small issues. Verified: web typecheck/ESLint/Prettier/Vitest (178)/build green.

- **Fix — Big-goal completion no longer claims to be an achievement:** in this app a ★
  achievement (a Wins entry) is a milestone the user starred, but finishing a whole big goal
  showed "Achievement unlocked", a "★ Achieved" chapter tag and an "Achieved" status label —
  none of which add anything to Wins. The celebration eyebrow now reads **"Big goal complete"**,
  the 100% chapter tag is **"✓ Complete"** (check, not star), and the server's 100% band label is
  **"Complete"** (band id `achieved` unchanged, so no API change). The celebration's **"Reflect on
  it"** button was removed because it led to the "coming soon" Reflections placeholder; restore it
  with the Reflections feature. Milestone-level "Achieved" tags (real ★ wins) are unchanged.
  Verified: web typecheck/ESLint/Prettier/Vitest (176) green; `go test ./...` green.

- **Fix — Page background, light-theme contrast (WCAG AA) & unified motivation copy:**
  *Page background.* Pages were painted with the handoff's cream radial gradient, which in the
  design is only the backdrop behind popups; the design's page frames sit on paper (`#faf6ee`,
  dark `#131210`) under white (dark `#16140f`) header/footer bars. The body now uses `--surface`
  (the unused `--bg`/`--bg-radial-*` tokens were removed), and cards that sit on the page (auth
  card, Wins cards, mobile chapter cards, gallery sections) moved to `bg-elevated` so they still
  lift off it.
  *Contrast.* A measured audit showed AA failures in **both** themes (the B6 note had dark as
  passing): in light, `text-muted`/`text-faint` ≈2.5–3.1:1, gold-as-text ≈2.3:1, teal ≈3.1:1,
  every error message (rose) ≈3.2:1, and white button text on teal/gold/rose; in dark,
  `text-faint` ≈2.2:1. Fix keeps the designed fills (rings, bars, swatches,
  medallion) and fixes text: new `--accent-{teal,blue,gold}-ink` tokens for accent-coloured text
  and glyphs (deeper in light, equal to the accent in dark); light `text-2`/`text-muted` and
  `--accent-rose` darkened; light buttons use dark ink; the danger button and avatar got their
  own ink tokens; `text-faint` is now decorative-only (its 8 text uses moved to `text-muted`);
  unused `text-label` removed. The done-milestone ✓ (white on the stored topic accent, ≈2:1 in
  both themes) now uses the button ink. New `src/styles/contrast.test.ts` asserts every pair
  from `tokens.css`, so palette edits can't regress silently.
  *Motivation.* The dashboard greeting showed the backend's six-band copy while chapter rows and
  the topic callout showed the frontend's four-band copy (different thresholds and voice), and
  `topic.motivation` was computed but never displayed. The backend is now the single source:
  `domain/motivation.go` implements the design's four bands (80% "almost there" threshold) and
  copy; `motivation` on topics changed from a string to `{band, label, title, message}`
  (`api/openapi.yaml` + regenerated `schema.d.ts`); the dashboard `message` uses the same table.
  The SPA renders the server copy and `lib/motivation.ts` only maps band → tone. During an
  optimistic milestone toggle the ring/percent move instantly; the copy follows on the settle
  refetch. Verified: `gofmt`, `go vet` (both tags), `make test` (race) and `make test-int`
  green; web typecheck, ESLint, Prettier, Vitest (208 tests, incl. the new contrast gate, which
  was confirmed to fail on the old values), `gen:api`, and build green.

- **Customer one-click launcher (local demo):** added root-level `START-HERE.txt` +
  `start/stop/uninstall-website.bat` and `deployments/docker-compose.customer.yml` so a
  non-technical Windows user can run the whole site (web + API + Mongo) by double-clicking,
  with the launcher auto-installing Docker Desktop when absent. See §6.1. No app changes.

- **Step C1 — Production serving (single-origin SPA + API):** the Go server now serves the
  built React SPA alongside the JSON API from one origin. Added an `internal/transport/http/
  webui` package that embeds `web/dist` via `//go:embed` behind the **`embedui`** build tag
  (default builds/tests stay backend-only), and a SPA handler (mounted as chi `NotFound`)
  with a client-side-routing fallback to `index.html`, `immutable` caching for hashed
  assets, `no-cache` for the shell, baseline security headers, and a strict **CSP**
  (`script-src 'self'` — the theme bootstrap was externalized to `/theme-init.js`). Unmatched
  `/api/**` still returns the JSON 404 envelope. New `make web-build` / `make build-all`
  targets and a multi-stage `Dockerfile` (Node → Go `-tags embedui` → distroless) produce
  one self-contained image; added `web/.npmrc` (`legacy-peer-deps`) for reproducible installs.
  Verified: `gofmt`/`go vet` (both tags)/`go test -race` green incl. hermetic SPA-handler
  tests; web typecheck/lint/format/77 tests/build green; and a **production-like live run**
  of the embedded binary against Mongo — SPA shell, hashed asset (immutable), `/achievements`
  deep-link fallback, JSON 404 for `/api/v1/nope`, ops endpoints, and a full register → login
  → dashboard flow over the single origin (refresh cookie `Path=/api/v1/auth`, no CORS header).

- **Step B6 — Frontend accessibility, responsive polish & states (Phase B complete):**
  audit-and-harden pass. Added a **skip-to-content** link targeting the `#main` landmark;
  confirmed/verified `aria-current="page"` on the active nav (header + bottom tab bar);
  added `role="alert"` to form-level and confirm-delete error messages; added a global
  `@media (prefers-reduced-motion: reduce)` guard in `global.css` (neutralizes skeleton
  pulses + hover transitions) on top of B5's JS Confetti gate. Introduced an automated
  **axe** accessibility gate (`vitest-axe`) over four key screens (login, populated
  dashboard, Wins, an open modal) plus skip-link / `aria-current` tests — **77 tests**
  total; typecheck, ESLint, Prettier, build, and `gen:api` all clean. Documented the
  cross-theme × breakpoint QA checklist (§5.1). **Open item — light-theme contrast:**
  a measured audit shows the **dark** palette passes WCAG AA, but several **light**-theme
  token usages fall below AA on the paper backgrounds (e.g. `text-muted` ≈2.8:1,
  `text-faint` ≈2.9:1, accent `gold` ≈2.3:1 / `teal` ≈3.1:1 as text) — and the next step
  up (`muted`) also fails, so usage swaps can't reach AA. Fixing this needs **palette
  value** changes, which are a system-wide design decision: token values were **not**
  altered pending sign-off (proposed adjustments to follow on request).

- **Step B5 — Frontend achievements, celebration & motivation system:** finalized the
  single four-band motivation util (`lib/motivation`) carrying the design's copy (label,
  tone, title, message) for both dashboard rows and the topic-detail callout. Added the
  **100% celebration** (`features/celebration/CelebrationModal`) — a gold-medallion modal
  with a `Confetti` primitive (skipped under `prefers-reduced-motion`), a congratulatory
  quote, and a computed stat strip (milestones · months-to-complete · rank) — triggered
  from `ModalHost` when completing a milestone brings its chapter to 100%. Built the
  **Wins** page (`features/achievements/*`), derived client-side via a `useQueries`
  fan-out over the year's chapters (no backend achievements endpoint) filtered to
  completed, starred milestones, plus `lib/achievements` (`deriveAchievements`,
  `celebrationStats`) and a `formatIsoDate` helper. Verified: typecheck, ESLint
  (+jsx-a11y), Prettier, Vitest (**71 tests**), production build, `gen:api` clean, and a
  **live run** against the real API + Mongo — create chapter → two milestones (one
  starred) → first completion stays at 50% (no celebration) → last completion hits 100%
  (celebration + derived win) → reopen retracts both. Reflections/Notes page and the
  full a11y/responsive sweep remain B6.

- **Step B4 — Frontend topic & milestone flows (CRUD modals):** made the dashboard
  interactive. Added the four designed popups — topic-detail (motivational callout,
  progress ring, milestone checklist with ✓/★/open states + due dates), create/edit a
  big goal (title, why, target date, category, accent swatch), add/edit a milestone
  (title, target date, notes, "Celebrate as an achievement ★" toggle), and an inline
  delete confirm — plus typed mutation hooks (`useCreateTopic`/`useUpdateTopic`/
  `useDeleteTopic`/`useCreateGoal`/`useUpdateGoal`/`useToggleGoalDone`/`useDeleteGoal`)
  with **optimistic update + rollback + settle-invalidate** that recompute topic
  progress and the dashboard aggregate locally. Added `lib/date` (DD-MM-YYYY ⇄ native
  date input + short "Due Oct 4" label), `useTopicGoals`, a feature-local `ModalHost`,
  and a `trash` icon + `danger` button variant + a `color` prop on `Ring`. Verified:
  typecheck, ESLint (+jsx-a11y), Prettier, Vitest (**56 tests**), production build,
  `gen:api` clean, and a **live run** against the real API + Mongo exercising the full
  create → add milestones → complete → edit → reopen → delete-with-cascade cycle. The
  celebration modal and final motivation copy remain B5.

- **Step B3 — Frontend dashboard (Editorial Journal):** the home screen now renders
  from live data — `useDashboard()`/`useTopics()` TanStack Query hooks feed the date
  greeting, italic sub-message, year-progress bar + New-Year countdown, and "The
  chapters" list (per-topic accent, ★ ACHIEVED at 100%, status · done/total, percent,
  mini bar), with loading / error / empty states. Added `lib/motivation` (ratio → band
  label) and `lib/greeting`. Opening a chapter / creating goals are stubbed until B4.
  Verified: typecheck, lint, Prettier, Vitest (39 tests), build, and a **live run**
  against the real API + Mongo confirming the dashboard/topics payloads (empty state,
  a topic with progress, and the updated aggregate).

- **Step B2 — Frontend API client & auth flow:** generated a typed client from
  `api/openapi.yaml` (`openapi-typescript` → `web/src/api/schema.d.ts`) wrapped with
  `openapi-fetch` (bearer injection + single-flight refresh-on-401 retry); added an
  `AuthProvider` that keeps the access token in memory and silently refreshes from the
  HttpOnly cookie on boot (reloads stay logged in), `RequireAuth` route gating, and
  on-brand login/register screens with envelope-driven field errors. Installed the
  TanStack Query provider for upcoming data hooks. Verified: typecheck, lint, Prettier,
  Vitest (32 tests, MSW-backed), production build, and a **live end-to-end run** against
  the real API + Mongo (register → login → /me → refresh → logout through the dev proxy).

- **Step B1 — Frontend scaffold & design system:** stood up `web/` (Vite + React +
  TypeScript + Tailwind v4). Encoded both theme token sets as CSS variables with a
  persisted, no-flash `data-theme` toggle; self-hosted the three fonts; built the shared
  primitives (Logo, Icon, Button, Input, Textarea, Label, ProgressBar, Ring, Modal, Chip,
  Toggle, SwatchGroup, Avatar), the app shell (header/footer/mobile tab bar), the routing
  skeleton, and a dev-only `/gallery`. Verified: typecheck, ESLint (+jsx-a11y), Prettier,
  Vitest (17 tests), production build. No API calls yet — that's Step B2.

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
