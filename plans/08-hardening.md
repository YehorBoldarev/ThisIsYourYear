# Step 8 — Hardening

> **Phase:** A (Backend) · **Status:** Draft for approval
> **Parent:** [`00-system-plan.md`](./00-system-plan.md) §8.8, §7 (cross-cutting), §10 (DoD)
> **Goal of this step:** Close the gap between "feature-complete" and "production-ready":
> add **rate limiting** on the abuse-prone auth endpoints, publish a **full, checked-in
> OpenAPI 3 spec** (served by the API) as the contract for the React client, and raise the
> **error-path test coverage** so every documented failure (401/404/409/422/429) is asserted
> end-to-end.
>
> **Out of scope:** Any React/UI (Phase B). No new product features, collections, or domain
> concepts — this step makes what Steps 1–7 built robust and documented.

---

## 1. Why this step exists

Steps 1–7 delivered the full backend feature set, verified happy-path + core error behavior.
Before the frontend consumes it, three cross-cutting concerns from the system plan (§7, §10)
remain:

1. **Rate limiting** — `register`/`login`/`refresh` are unauthenticated and are the natural
   targets for credential-stuffing and brute-force. The plan calls for "basic rate limiting on
   auth endpoints." Nothing throttles them today.
2. **OpenAPI spec** — §3 and §10 require a hand-authored OpenAPI 3 contract, checked in and
   served. The React client (Phase B) needs a precise contract; today the only API description
   is prose in `docs/dev_run.md`.
3. **Error-path coverage** — the error envelope and status mapping exist and are partially
   tested, but there is no single suite that asserts *every* documented failure shape. Hardening
   means locking the error contract with tests so it can't silently regress.

Everything here respects the architecture: rate limiting is a **transport middleware** (no
domain/service change), the OpenAPI file is a **static artifact** embedded and served, and the
error-path tests exercise the **existing** endpoints — zero new business logic.

---

## 2. Outcome / Definition of Done

- **Rate limiting:** the unauthenticated auth routes (`/auth/register`, `/auth/login`,
  `/auth/refresh`) are throttled per client IP. Exceeding the limit returns **`429`** with the
  standard error envelope (`code: "rate_limited"`) and a `Retry-After` header. Limits are
  **env-configurable** and can be effectively disabled for tests.
- **OpenAPI:** `api/openapi.yaml` (OpenAPI 3.1) describes every endpoint, request/response
  schema, the error envelope, auth (bearer + refresh cookie), and status codes. It is
  **embedded** (`go:embed`) and served at **`GET /openapi.yaml`**. A test asserts it parses and
  is served `200`.
- **Error-path coverage:** an integration suite asserts the envelope + status for `401`
  (missing/invalid token), `404` (cross-user / missing resource), `409` (duplicate email),
  `422` (validation / malformed JSON / unknown field), and `429` (rate limit), plus `405` for a
  wrong method on a known path.
- `go build`, `go vet` (both tags), `gofmt`, `golangci-lint` (both tags), `make test` pass;
  `make test-int` passes against real `mongo:7`.
- `docs/dev_run.md` updated (rate-limit config vars, `/openapi.yaml` row, 429 note, changelog);
  the OpenAPI file is referenced from the docs.

---

## 3. Design

### 3.1 Rate limiting (transport middleware)

- **Where:** applied to the **public `/auth` group only** (register/login/refresh/logout).
  Authenticated routes are already gated by a bearer token and are lower-risk; the plan scopes
  rate limiting to auth. `/healthz` and `/readyz` are never throttled (probes must always
  answer).
- **Key:** the **TCP peer IP** derived from `r.RemoteAddr` — *not* `X-Forwarded-For`. This is
  consistent with the deliberate choice in `router.go` to avoid `middleware.RealIP` (spoofable
  forwarding headers). A `// when behind a trusted proxy, revisit the key source` note is added,
  mirroring the existing comment. Behind a proxy today the limiter would see the proxy IP; that
  is an acceptable, documented limitation until a vetted proxy is introduced.
- **Algorithm:** a fixed-window counter per IP (N requests per window), which is sufficient for
  "basic" abuse protection and cheap in memory with periodic eviction of stale keys.
- **Response:** on limit, write the **same `errorBody` envelope** used everywhere else with
  `code: "rate_limited"`, message "too many requests", HTTP `429`, and a `Retry-After` header
  (seconds until the window resets). No new domain error — rate limiting is purely a transport
  concern, so a small `writeRateLimited` helper in the transport layer emits it (keeping the
  envelope identical to `respondError`'s output).
- **Config (new env vars, with safe defaults):**

  | Variable | Default | Meaning |
  | --- | --- | --- |
  | `RATE_LIMIT_AUTH_REQUESTS` | `20` | max auth requests per window per IP (`0` = disabled) |
  | `RATE_LIMIT_AUTH_WINDOW` | `1m` | the window duration |

  `0` disables the middleware entirely (the integration harness sets it to `0` so the auth flow
  tests — which register/login many users quickly — are unaffected). Defaults are generous
  enough not to hinder a real user (a human logs in a handful of times per minute) while
  stopping automated brute force.

### 3.2 OpenAPI 3 spec (checked in + served)

- **File:** `api/openapi.yaml`, OpenAPI **3.1**, hand-authored, the single source of truth for
  the HTTP contract. Covers: all `/api/v1` routes + `/healthz`/`/readyz`; request/response
  schemas mirroring the existing DTOs (`RegisterRequest`, `TokenResponse`, `TopicResponse` with
  `progress`+`motivation`, `GoalResponse`, `DashboardResponse`, etc.); the shared **error
  envelope** (`{error:{code,message,fields?}}`) as a reusable component referenced by every
  `4xx`/`5xx`; security schemes for the **bearer access token** and the **refresh cookie**;
  and the documented status codes per operation (incl. `429`).
- **Serving:** embed the file with `go:embed` and serve the bytes at **`GET /openapi.yaml`**
  (`application/yaml`), outside the auth group. No Swagger-UI bundling (that needs external CDN
  assets and is a frontend concern); the raw spec is enough for client generation and for
  import into any viewer. Embedding means the binary is self-contained — no file-path
  dependency at runtime.
- **Correctness guard:** a unit test unmarshals the embedded YAML (asserting it is well-formed
  and has the expected top-level `openapi`/`paths` keys) and an integration/router test asserts
  `GET /openapi.yaml` returns `200` with the YAML content type. This keeps the spec from rotting
  into invalid YAML without pulling in a heavyweight schema validator.

### 3.3 Error-path coverage (integration)

A dedicated `error_paths_integration_test.go` asserts the **status + envelope** (`code`, and
`fields` where relevant) for each failure the API promises, in one place so the error contract
is locked:

| Status | Trigger |
| --- | --- |
| `422` | register with a bad email / short password (field errors); malformed JSON; unknown field |
| `401` | protected route with no token, a garbage token, and an expired/invalid token |
| `404` | GET/PATCH/DELETE another user's topic id (ownership) and a well-formed-but-absent id |
| `409` | register the same email twice |
| `429` | exceed the auth rate limit (suite enables a tiny limit for this case) |
| `405` | wrong method on a known path (e.g. `DELETE /api/v1/me`) |

These reuse the existing `testApp` harness. The 429 case builds its own app with a low
`RATE_LIMIT_AUTH_REQUESTS` so it doesn't couple to the default.

---

## 4. Files

```
api/
  openapi.yaml                              # NEW: OpenAPI 3.1 contract (checked in)

internal/
  config/
    config.go                               # CHANGED: RATE_LIMIT_AUTH_* vars + validation
    config_test.go                          # CHANGED: cover new vars + validation

  transport/http/
    ratelimit.go                            # NEW: per-IP fixed-window middleware + writeRateLimited
    ratelimit_test.go                       # NEW: allows under limit, 429 over, window reset, disabled=0
    openapi.go                              # NEW: go:embed api/openapi.yaml + GET /openapi.yaml handler
    openapi_test.go                         # NEW: spec parses; endpoint serves 200 + content type
    error_paths_integration_test.go         # NEW: 401/404/409/422/429/405 envelopes end-to-end
    router.go                               # CHANGED: apply limiter to /auth; mount /openapi.yaml; RouterDeps.RateLimit
    harness_integration_test.go             # CHANGED: pass RateLimit (disabled) into RouterDeps

cmd/api/main.go                             # CHANGED: read rate-limit config, wire into RouterDeps
docs/dev_run.md                             # CHANGED: config vars, /openapi.yaml row, 429 note, changelog
```

No new domain types, services, collections, or indexes.

### Dependency note
The fixed-window limiter is small and can be **hand-rolled in `ratelimit.go`** (a mutex-guarded
`map[string]*window` with periodic eviction) to stay stdlib-only and dependency-light, matching
the project's stdlib-first posture — **or** we use `github.com/go-chi/httprate` (same org as the
router, well-tested, handles eviction). This is the one choice worth your call — see §7.

---

## 5. API shape (additions)

`GET /openapi.yaml` → `200`, `Content-Type: application/yaml`, body = the spec.

Rate-limited auth request → `429`:

```json
{ "error": { "code": "rate_limited", "message": "too many requests" } }
```
with header `Retry-After: 42`.

No change to any existing success response.

---

## 6. Testing Strategy

| Layer | What | Runs in |
| --- | --- | --- |
| unit | limiter: under limit → allowed; over → `429` + `Retry-After`; window reset re-allows; `0` disables; distinct IPs isolated (via `RemoteAddr`) | `make test` |
| unit | OpenAPI: embedded file parses as YAML and has `openapi`/`paths`; `GET /openapi.yaml` handler returns `200` + `application/yaml` | `make test` |
| unit | config: new vars parse; window/requests defaults; invalid values rejected | `make test` |
| end-to-end | error-path suite (§3.3): every documented status + envelope; 429 over a tiny limit; existing auth/topic/goal/dashboard flows still green with the limiter present (disabled) | `make test-int` |

## 7. Decisions — DECIDED

- **(a) Rate-limiter implementation — LOCKED: hand-rolled, stdlib-only.** A small
  mutex-guarded per-IP fixed-window map (`ratelimit.go`) with periodic eviction of stale keys.
  No new dependency; matches the project's stdlib-first posture; fully unit-tested. Keys on
  `RemoteAddr`, emits the standard envelope.
- **(b) Rate-limit scope — LOCKED: auth endpoints only** (per system plan §7). Not a global
  limiter.
- **(c) OpenAPI serving — LOCKED: raw `/openapi.yaml` only, no bundled Swagger UI.** A UI needs
  external assets and is a Phase-B concern; the raw spec suffices for client generation and
  import into any viewer.

**Also locked:** error envelope shape is unchanged; `429` uses `code: "rate_limited"`; rate
limiting never touches health probes; the spec is checked in and embedded.

## 8. Verification Checklist

1. `make tidy && go build ./...` — clean.
2. `gofmt` / `go vet` (both tags) / `golangci-lint` (both tags) — clean.
3. `make test` — unit suite green, `-race` (limiter, openapi, config).
4. `make test-int` — full flow + error-path suite green against real `mongo:7`.
5. Live smoke: start the API; `GET /openapi.yaml` returns the spec; hammer `/auth/login` past
   the limit and confirm `429` + `Retry-After`; confirm a normal login still works within the
   limit. Update `docs/dev_run.md` and re-verify its commands.

## 9. What This Unblocks

Completes **Phase A (Backend)**: the service is now abuse-resistant on its public surface,
self-documenting via a served OpenAPI contract, and its error behavior is locked by tests.
Phase B (React + Tailwind) can generate a typed client from `api/openapi.yaml` and build against
a stable, documented API.
