# Running ThisIsYourYear Locally

How to run the backend on your machine, exercise it, and shut it down cleanly.

> **Keep this current:** this document is updated at the end of every implementation
> step. It reflects the app **as of Step B6 (frontend accessibility & polish — Phase B complete)**.
> Backend is complete through Step 9; the frontend (Phase B) lives in `web/` — see
> [§5 Frontend](#5-frontend-phase-b) and [`web/README.md`](../web/README.md).

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
starred milestones across the year's chapters. One **four-band motivation system**
(`lib/motivation`, the design's copy) feeds both the dashboard rows and the topic-detail
callout — there is no achievements endpoint; the Wins page fans out `GET /topics/{id}/goals`.
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
- **Contrast (open item):** measured token contrast passes AA in **dark**; the **light**
  palette's secondary/label/accent-on-paper values fall below AA and need a palette
  decision (see the B6 changelog note) — token *values* aren't changed without sign-off.

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
- **Docker:** the `Dockerfile` is now multi-stage (Node build → Go build with
  `-tags embedui` → distroless). One image serves everything on `:8080`.
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

## Changelog

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
