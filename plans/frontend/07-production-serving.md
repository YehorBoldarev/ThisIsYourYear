# Step C1 — Production serving (single-origin SPA + API)

> **Status:** Draft for approval
> **Parent:** [`00-system-plan.md`](00-system-plan.md) §8 (deferred: "Production serving (Phase C)")
> **Builds on:** Phase B (B1–B6, the finished SPA in `web/`) and the Go backend (Steps 1–9) — **done**.
> **Approval gate:** No code is written until you approve this document.
> **Note:** This is a **backend + build/deploy** step (Go, Docker, Make) — the first of Phase C.
> It touches Go source, so it also runs the **Go** quality gates (`make test`, `vet`, `fmt`, build),
> not just the web ones.

---

## 1. Objective

Ship the app as **one deployable artifact** that serves the built React SPA **and** the
JSON API from the **same origin**, so in production there is no CORS, no separate static
host, and the HttpOnly refresh cookie flows naturally. The dev experience (Vite dev
server + `/api` proxy) is unchanged.

### In scope
- **Static serving in Go:** serve `web/dist` (`index.html`, hashed JS/CSS, fonts,
  `favicon`) from the Go server, with a **SPA fallback** (unknown non-API GET →
  `index.html`, HTTP 200) so client-side routes like `/achievements` deep-link and
  reload correctly.
- **Correct 404 semantics:** unmatched **`/api/**`** paths still return the JSON error
  envelope (not `index.html`); operational endpoints (`/healthz`, `/readyz`,
  `/openapi.yaml`) keep working.
- **Caching headers:** fingerprinted assets (`/assets/*`) → `Cache-Control: public,
  max-age=31536000, immutable`; `index.html` → `no-cache` so new deploys are picked up.
- **Baseline security headers** on the HTML response (`X-Content-Type-Options: nosniff`,
  `Referrer-Policy: same-origin`, `X-Frame-Options: DENY`); **CSP** is a decision (**D4**).
- **Build wiring:** a multi-stage **Dockerfile** (Node build → Go build → distroless) and
  Make targets that build the SPA and produce the combined binary; the embed is toggled
  by a **build tag** (**D2**) so backend-only `go build`/tests don't require a built
  frontend.
- **Config for prod single-origin:** document that `CORS_ALLOWED_ORIGINS` stays **empty**
  in production (same origin), `COOKIE_SECURE=true` (already the default), and the
  existing cookie path `/api/v1/auth` is unaffected.
- **Verification:** a real production-like run — build the SPA, build the tagged binary,
  run it against Mongo, and curl the served `index.html`, a hashed asset, a deep-link
  fallback, an API route, a bogus `/api` route (JSON 404), and the ops endpoints.

### Explicitly NOT in this step
- No new product features or API endpoints; no `schema.d.ts`/OpenAPI change.
- No TLS termination / reverse proxy / real hosting provider setup (that's a later C
  step or ops concern); this step makes the artifact, not the cloud.
- No CDN, no server-side rendering, no HTTP/2 push.
- Reflections functionality remains out (locked).

---

## 2. Current state (audited)

- **Server:** `cmd/api/main.go` → `transporthttp.NewRouter(deps)` (chi) → `NewServer`.
  Routes today: `/healthz`, `/readyz`, `/openapi.yaml`, and `/api/v1/**`. **No** static
  file serving and **no** `NotFound` handler → the SPA has nowhere to be served from yet.
- **Frontend build:** `web/` builds to `web/dist/` via `npm run build` (hashed
  `assets/*`, self-hosted fonts, `index.html`). Vite dev proxy handles `/api` in dev only.
- **Config:** `internal/config` already has `AppEnv`/`IsProduction()`, `CookieSecure`
  (default true), `CORSAllowedOrigins` (empty ⇒ CORS middleware not installed — perfect
  for same-origin prod). No web-serving settings yet.
- **Docker:** single-stage Go build on distroless; **does not** build the frontend.
- **`go:embed` constraint:** embed paths can't contain `..`, so the binary can't embed
  `web/dist` in place. The build must place `dist` **inside** the embedding Go package
  before `go build` (handled by the Make/Docker steps below).

---

## 3. Design (how it fits, pending your decisions in §4)

- **New package `internal/transport/http/webui`** (name TBD):
  - `//go:embed all:dist` over a package-local `dist/` populated at build time, guarded by
    a build tag (**D2**): `//go:build embedui` provides the real `embed.FS`; the default
    file provides an empty FS + a flag "not embedded" so dev/backend runs and tests don't
    need a built SPA.
  - A `Handler(fsys) http.Handler` that: serves an existing file (with the caching +
    security headers), else returns `index.html` (200) for GET/HEAD, else 404.
- **Router wiring:** mount the SPA as chi's `NotFound` handler (it runs only for routes
  not already matched, so `/api/v1/**`, `/healthz`, `/readyz`, `/openapi.yaml` are
  untouched). The handler returns the **JSON 404** envelope for any path under `/api/`
  and serves the SPA otherwise. When the UI isn't embedded (dev/backend-only), `NotFound`
  falls back to today's JSON 404 for everything — no behaviour change.
- **Make/Docker:**
  - `make web-build` → `npm ci && npm run build` in `web/`, then copy `web/dist` →
    `internal/transport/http/webui/dist` (git-ignored).
  - `make build` stays backend-only (fast, no UI); a new `make build-all` builds the SPA,
    copies it, and `go build -tags embedui` → a self-contained binary.
  - **Dockerfile**: stage 1 `node:22` builds `web/` → `dist`; stage 2 `golang:1.26`
    copies `dist` into the embed dir and `go build -tags embedui`; stage 3 distroless
    runs the single binary. One image serves everything on `:8080`.

---

## 4. Decisions for you (defaults recommended; flag any before I build)

**D1 — Serving strategy (the architectural call).**
- **(Recommended) Embed the SPA in the Go binary** (`go:embed`) → one self-contained
  artifact, matching the current single-binary distroless deploy; no runtime filesystem or
  second service.
- **(Alt A) Serve from a configured directory** (`WEB_DIST_DIR`) the deployment mounts —
  decouples the two builds, but now two things ship and must stay in sync.
- **(Alt B) Separate static host / CDN** for `web/dist`, API-only Go service — best cache
  locality and scaling, but reintroduces **CORS + cross-origin cookie** complexity and a
  second deploy target.

**D2 — Embed behind a build tag?**
- **(Recommended) Yes** (`-tags embedui`): default `go build`/`make build`/`go test`
  stay fast and need **no** built frontend; the combined artifact is an explicit,
  tagged build. Keeps backend CI independent of Node.
- **(Alt) Always embed:** simpler mental model, but every `go build` and the Go test
  suite then require `web/dist` to exist first.

**D3 — Where the combined build's tests/CI run.**
- **(Recommended) Add a Go handler test** using a small in-memory `fs.FS` fixture (no real
  `dist` needed) covering: asset served with immutable header, deep-link → `index.html`,
  `/api/**` miss → JSON 404, `index.html` → `no-cache`. Fast and hermetic.
- **(Alt) Only verify via the live run** — lighter, but no regression guard in CI.

**D4 — Content-Security-Policy.**
- **(Recommended) Add a conservative CSP** now (`default-src 'self'`; `img-src 'self'
  data:`; `style-src 'self' 'unsafe-inline'` for the theme's inline styles; `font-src
  'self'`) — meaningful hardening since we now serve the HTML, and the SPA is same-origin
  with self-hosted fonts so it should fit. I'll **verify the app runs clean under it** in
  the live check and loosen only if something legitimately breaks.
- **(Alt) Defer CSP** to a later hardening step and ship only the non-breaking baseline
  headers (nosniff / referrer / frame options) now.

---

## 5. Files (created ·/ changed) — indicative

```
internal/transport/http/
  webui/
    webui.go            + Handler(fs.FS): file-or-index fallback + caching/security headers
    embed_on.go         + //go:build embedui  → real go:embed FS
    embed_off.go        + //go:build !embedui → empty FS + Enabled()=false (dev/backend default)
    webui_test.go       + handler tests over an in-memory fs.FS (D3)
    dist/               + (git-ignored) frontend build output, populated by `make web-build`
  router.go             ~ mount the SPA handler as NotFound (JSON 404 kept for /api/**)
cmd/api/main.go         ~ pass the webui handler/FS into RouterDeps
Dockerfile              ~ add Node build stage; go build -tags embedui in the Go stage
Makefile                ~ web-build, build-all (tagged), keep build backend-only
.gitignore              ~ ignore internal/transport/http/webui/dist
docs/dev_run.md         ~ new "Production build & serving" section + changelog
web/README.md           ~ note the single-origin production build
```

Exact shape depends on D1/D2; the above assumes the recommended embed + build-tag path.

---

## 6. Verification (definition of done)

- **Go gates:** `make fmt` / `make vet` / `make test` (race) green, incl. the new webui
  handler test; `go build -tags embedui ./...` compiles.
- **Web gates:** unchanged and green (`npm run build` produces `dist`).
- **Live production-like run:** `make web-build && make build-all`, run the binary against
  Mongo with `APP_ENV=production COOKIE_SECURE=false` (local, no TLS), then curl:
  - `GET /` → 200 `text/html` (the SPA shell);
  - `GET /assets/<hashed>.js` → 200 with `Cache-Control: …immutable`;
  - `GET /achievements` (deep link) → 200 `index.html` (SPA fallback);
  - `GET /api/v1/health`-style bogus `/api/v1/nope` → **JSON 404** envelope;
  - `GET /healthz` / `/readyz` / `/openapi.yaml` → still correct;
  - a full **register → login → dashboard** happy path over the single origin (refresh
    cookie set with `Path=/api/v1/auth`), confirming no CORS is needed;
  - if **D4** = CSP on, confirm the browser/app loads with **no CSP violations**.
- Docs updated (`docs/dev_run.md`, `web/README.md`). **Nothing committed.**

The honesty caveat continues: the API/serving contract is verified by the Go tests + a
curl-level live run; a full browser click-through under the production binary is the
documented manual check, not a screenshot I capture here.

---

On approval I implement C1 on the chosen strategy (D1–D4), verify per §6, update docs, and
report back for your **"Task completed."** Phase C can then continue with real hosting/TLS
as a subsequent step when you want it.
