# Step 3 — Auth & Users

> **Phase:** A (Backend) · **Status:** Draft for approval
> **Parent:** [`00-system-plan.md`](./00-system-plan.md) §8.3, §5, §6, §7
> **Goal of this step:** Build the **first real domain entity (`User`)** and the full
> authentication system on top of the Step 2 persistence foundation: registration, login,
> JWT access tokens, server-persisted refresh **sessions** with "Remember me" (31-day),
> refresh with rotation, logout (server-side revocation), an auth middleware, and `GET /me`.
> This step also introduces the **transport-edge machinery** every later feature reuses:
> request DTOs + validation, the central **domain-error → HTTP-status** mapper, JSON request
> decoding, and CORS for the future SPA.
>
> **Out of scope:** Topics, Goals, dashboard (Steps 4–7). Rate limiting and the full OpenAPI
> spec are deferred to Step 8 (Hardening), though the code is structured to accept them.

---

## 1. Why this step exists

Everything a user owns hangs off an authenticated identity, so `User` + auth is the
prerequisite for Topics/Goals/Dashboard. It is also where the **HTTP feature template** is
established once: how a route group is wired, how requests are validated, how domain errors
become status codes, and how handlers stay thin. Steps 4–7 then follow this template exactly
— new entity = new repo + service + router group, no rewrites. This is the expandability
promise from the system plan made concrete.

---

## 2. Outcome / Definition of Done

- A user can **register** (`POST /api/v1/auth/register`) → account created, password stored
  bcrypt-hashed, duplicate email rejected as `409 Conflict`.
- A user can **log in** (`POST /api/v1/auth/login`, body includes `rememberMe`) → receives a
  short-lived **access token** (JSON) and a **refresh session** (transport per §7 decision);
  wrong credentials → `401` with a generic message (no user-enumeration).
- A user can **refresh** (`POST /api/v1/auth/refresh`) → a valid, unexpired, unrevoked refresh
  token yields a new access token and a **rotated** refresh token; a reused/rotated-away or
  revoked token is rejected `401` and (on detected reuse) the session lineage is revoked.
- A user can **log out** (`POST /api/v1/auth/logout`) → the server-side session is revoked;
  the refresh token no longer works.
- **`GET /api/v1/me`** returns the current user's public profile; unauthenticated/expired
  access tokens → `401`.
- Sessions expire automatically: **Remember me = 31 days**, otherwise a shorter TTL (§7);
  expiry is enforced both in query logic and by a Mongo **TTL index** that reaps old docs.
- Cross-cutting: request validation returns `422` with field detail; a single error-mapper
  translates `domain.*` errors to status codes at the transport edge only.
- `go build`, `go vet`, `gofmt`, `golangci-lint`, `make test` (unit) all pass; `make test-int`
  passes against real `mongo:7` (Docker) covering the full auth flow end-to-end over HTTP.

---

## 3. Data Model (this step)

### `users`
| Field | BSON | Notes |
| --- | --- | --- |
| id | `_id` | ObjectID (from `mongo.Document`) |
| email | `email` | normalized lower-case; **unique index** |
| passwordHash | `passwordHash` | bcrypt; never serialized to any API response |
| displayName | `displayName` | user-facing name |
| createdAt / updatedAt | — | from `mongo.Document` |

### `sessions`
| Field | BSON | Notes |
| --- | --- | --- |
| id | `_id` | ObjectID |
| userId | `userId` | owning user |
| refreshTokenHash | `refreshTokenHash` | **SHA-256** of the opaque refresh token (never store raw) |
| rememberMe | `rememberMe` | drives TTL |
| userAgent | `userAgent` | captured at issue for auditability |
| ip | `ip` | captured at issue |
| expiresAt | `expiresAt` | absolute expiry; **TTL index** reaps at this time |
| revokedAt | `revokedAt` | non-nil once logged out / rotated / reused-detected |
| createdAt / updatedAt | — | from `mongo.Document` |

**Indexes registered this step** (via the Step 2 registry, applied by `EnsureIndexes`):
- `users`: `{ email: 1 }` **unique**.
- `sessions`: `{ expiresAt: 1 }` **TTL (expireAfterSeconds: 0)** — Mongo deletes when
  `expiresAt` passes.
- `sessions`: `{ refreshTokenHash: 1 }` **unique** — fast lookup + guarantees no hash reuse.
- `sessions`: `{ userId: 1 }` — list/revoke a user's sessions.

> Note: the Step 2 `IndexSpec` already supports `Unique` and a `TTL *time.Duration`. A TTL of
> `0` (reap exactly at `expiresAt`) is expressed by setting `expireAfterSeconds: 0`; I will
> confirm `IndexSpec.model()` emits that correctly and, if the current shape can't express a
> zero-TTL-on-a-date index, make a **minimal** additive change to `IndexSpec` (an
> `ExpireAfter *time.Duration` already covers it). Any such change is backward-compatible.

---

## 4. Architecture & Files

Following the layered design (`domain → service → repository → transport/http`) with an
`auth` package for cryptographic concerns.

```
internal/
  domain/
    errors.go              # CHANGED: add ErrUnauthenticated, ErrInvalidCredentials
    user.go                # User entity (embeds mongo.Document) + construction/invariants
    session.go             # Session entity (embeds mongo.Document) + helpers (IsActive, etc.)
    user_test.go, session_test.go

  auth/
    password.go            # bcrypt Hash / Compare (cost from config)
    password_test.go
    token.go               # JWT access-token issue/verify (jwt/v5, HS256)
    token_test.go          # issue→verify round-trip, expiry, tamper, wrong-secret
    refresh.go             # opaque refresh token: generate (crypto/rand) + SHA-256 hash
    refresh_test.go

  repository/
    users.go               # UserRepository wraps mongo.Repository[User,*User]; ByEmail(); registers index
    sessions.go            # SessionRepository wraps Repository[Session,*Session]; ByTokenHash(), Revoke(); registers indexes

  service/
    auth_service.go        # Register, Login, Refresh, Logout — orchestrates repos + auth pkg
    auth_service_test.go   # unit with in-memory/mock repos (no DB)
    user_service.go        # GetByID (for /me)

  transport/http/
    dto.go                 # request/response DTOs (RegisterRequest, LoginRequest, TokenResponse, MeResponse, ...)
    render.go              # decodeJSON + respondJSON + respondError (domain-error → status mapper)
    render_test.go         # unit: error mapping table
    validate.go            # validator/v10 wiring; ValidationError -> 422 field detail
    middleware_auth.go     # Authenticator middleware: verify access token, inject userID into ctx
    middleware_auth_test.go
    cors.go                # CORS for the SPA (allow-list from config)
    auth_handler.go        # register/login/refresh/logout handlers
    user_handler.go        # /me handler
    router.go              # CHANGED: mount /auth (public) and protected group with /me
    context.go             # typed context key + UserID(ctx) accessor

  config/
    config.go              # CHANGED: JWT secret+TTLs, refresh TTLs, bcrypt cost, cookie + CORS settings
    config_test.go         # CHANGED: new fields defaults/validation

  platform/mongo/
    index.go               # CHANGED only if needed to express zero-TTL-on-date (see §3 note)

cmd/api/main.go            # CHANGED: construct repos+services, pass AuthService/Authenticator into RouterDeps
```

Integration tests (`//go:build integration`, `*_integration_test.go`):
```
internal/transport/http/auth_flow_integration_test.go   # full HTTP flow against real Mongo
internal/repository/users_integration_test.go           # ByEmail + unique index
internal/repository/sessions_integration_test.go        # token-hash lookup, revoke, TTL index present
```

---

## 5. Component Details

### 5.1 Domain (`internal/domain`)
- **`User`**: embeds `mongo.Document`; fields email, passwordHash, displayName. A constructor
  `NewUser(email, displayName, passwordHash)` **normalizes email** (trim + lower) and enforces
  non-empty invariants, returning `ErrValidation` on violation. `passwordHash` has `json:"-"`
  and never appears in a DTO regardless.
- **`Session`**: embeds `mongo.Document`; helper `IsActive(now)` = `revokedAt == nil && now.Before(expiresAt)`.
- **`errors.go`**: add `ErrUnauthenticated` (missing/invalid access token → 401) and
  `ErrInvalidCredentials` (wrong email/password → 401, generic message) with `Is*` helpers,
  matching the existing pattern.

### 5.2 Auth package (`internal/auth`) — pure, no DB, no HTTP
- **`password.go`**: `Hash(plain string) (string, error)` and `Compare(hash, plain) error`
  over `x/crypto/bcrypt`; cost injected (default 12). Compare returns `ErrInvalidCredentials`
  on mismatch so callers don't branch on bcrypt internals.
- **`token.go`**: `Issuer` with `Issue(userID string) (token string, expiresAt time.Time)`
  and `Verify(token) (Claims, error)`. HS256, secret + access-TTL injected. Claims: `sub`,
  `iat`, `exp`, `jti`. `Verify` maps any invalid/expired token to `ErrUnauthenticated`.
- **`refresh.go`**: `GenerateRefreshToken()` → 32 bytes from `crypto/rand`, base64url-encoded
  (the raw secret handed to the client); `HashRefreshToken(raw) string` → hex SHA-256 (stored).
  High-entropy random ⇒ SHA-256 is the correct, fast choice (bcrypt is for low-entropy
  passwords).

### 5.3 Repositories (`internal/repository`)
- Thin wrappers over the generic base, demonstrating the intended pattern:
  - `UserRepository`: `Create`, `ByEmail(ctx, email)`, `ByID`. Constructor calls
    `client.RegisterIndexes("users", {email unique})`.
  - `SessionRepository`: `Create`, `ByTokenHash`, `Revoke(ctx, id, at)`,
    `RevokeAllForUser(ctx, userID, at)`. Constructor registers the three session indexes.
- Registering indexes at repo construction (before `EnsureIndexes` runs in `main`) is exactly
  the Step 2 contract, now exercised for real.

### 5.4 Services (`internal/service`)
- **`AuthService`** orchestrates the use-cases; depends on repo **interfaces** (declared here)
  + the `auth` package, so it is unit-testable with mocks and **no database**:
  - `Register(ctx, email, displayName, password)`: hash → build `User` → `Create`; duplicate
    email surfaces as `ErrConflict`.
  - `Login(ctx, email, password, rememberMe, ua, ip)`: fetch by email → bcrypt compare
    (constant-ish; same generic `ErrInvalidCredentials` for unknown email **and** bad password)
    → mint access token → generate refresh token, persist hashed session with TTL per
    `rememberMe` → return access token + raw refresh token + expiries.
  - `Refresh(ctx, rawRefresh, ua, ip)`: hash → lookup session → validate active → **rotate**
    (revoke old, create new session) → new access + new refresh. If the presented token maps to
    an already-revoked session (reuse), treat as compromise: `RevokeAllForUser` and return
    `ErrUnauthenticated`.
  - `Logout(ctx, rawRefresh)`: hash → revoke that session (idempotent; unknown token → success
    to avoid leaking existence).
- **`UserService.GetByID`** backs `/me`.

### 5.5 Transport (`internal/transport/http`)
- **`render.go`** — the reusable edge:
  - `decodeJSON(r, dst)`: size-limited body, `DisallowUnknownFields`, clear errors.
  - `respondJSON(w, status, v)`.
  - `respondError(w, r, err)`: the **single** place mapping domain errors → HTTP:
    | domain error | status |
    | --- | --- |
    | `ErrValidation` | 422 |
    | `ErrInvalidCredentials`, `ErrUnauthenticated` | 401 |
    | `ErrNotFound` | 404 |
    | `ErrConflict` | 409 |
    | (unclassified) | 500 (+ logged, generic body) |
    Response body shape: `{ "error": { "code": "...", "message": "..." } }` — stable contract
    for the SPA. Internal error details are never leaked to clients.
- **`validate.go`**: `go-playground/validator/v10` on DTO structs; failures → `ErrValidation`
  carrying per-field messages rendered in the 422 body.
- **`middleware_auth.go`**: reads `Authorization: Bearer <access>`, verifies via the `auth`
  issuer, injects `userID` into context (typed key in `context.go`); missing/invalid → 401 via
  `respondError`. Protected routes live in a chi sub-group using this middleware.
- **`cors.go`**: `github.com/go-chi/cors` with an **allow-list** of origins from config
  (`CORS_ALLOWED_ORIGINS`), credentials enabled only if the refresh transport needs cookies
  (§7). Mounted before route groups.
- **Handlers** are thin: decode → validate → call service → render. No business logic.

### 5.6 Wiring (`cmd/api/main.go`)
Construct `UserRepository` + `SessionRepository` (which register their indexes) → build the
`auth` issuer/hasher from config → `AuthService`/`UserService` → pass them (and the
`Authenticator`) into `RouterDeps`. `EnsureIndexes` already runs after wiring, so the new
indexes are created on boot. Order is unchanged except for the added constructions.

### 5.7 Config additions
`JWT_SECRET` (required in production; dev default only for local), `ACCESS_TOKEN_TTL`
(default 15m), `REFRESH_TTL_REMEMBER` (default 744h = 31d), `REFRESH_TTL_DEFAULT`
(default 24h), `BCRYPT_COST` (default 12), `CORS_ALLOWED_ORIGINS` (comma list),
and cookie settings if §7 = cookie (`COOKIE_DOMAIN`, `COOKIE_SECURE`). `validate()` gains:
non-empty `JWT_SECRET` in production, sane TTL/cost ranges.

---

## 6. Testing Strategy

| Layer | What | Tooling | Runs in |
| --- | --- | --- | --- |
| `auth` | bcrypt hash/verify; JWT issue/verify, expiry, tamper, wrong secret; refresh gen/hash determinism | stdlib + testify | `make test` |
| `service` | Register/Login/Refresh/Logout logic, rotation, reuse-detection, generic-credentials | mock repos (no DB) | `make test` |
| transport | error mapper table; validation → 422; auth middleware accept/reject | httptest, no DB | `make test` |
| repository | `ByEmail`, unique-email conflict, session lookup/revoke, TTL index exists | testcontainers `mongo:7` | `make test-int` |
| **end-to-end** | register→login→/me→refresh(rotate)→logout→refresh-fails, over real HTTP + real Mongo | httptest server + testcontainers | `make test-int` |

Security-focused assertions: `passwordHash` never appears in any response body; raw refresh
token never stored (only its hash); reused refresh token revokes the lineage; expired access
token → 401.

---

## 7. Refresh-Token Transport — DECIDED: HttpOnly cookie (A)

**Locked by you:** the refresh token travels in an **`HttpOnly; Secure; SameSite=Lax` cookie**
scoped to `Path=/api/v1/auth`; the access token is returned in the JSON body (SPA keeps it in
memory). Concretely:

- **Login** → `200` with `{ accessToken, expiresAt, user }` in the body **and**
  `Set-Cookie: refresh_token=<raw>; HttpOnly; Secure; SameSite=Lax; Path=/api/v1/auth;
  Max-Age=<ttl>`. `Max-Age` = 31d when `rememberMe`, else the default TTL (§8).
- **Refresh** → browser auto-sends the cookie; server rotates and re-issues a new
  `Set-Cookie` refresh token + a fresh access token in the body.
- **Logout** → session revoked server-side **and** the cookie cleared
  (`Set-Cookie: refresh_token=; Max-Age=0`).

**Implications baked into the plan:**
- `cors.go` enables `AllowCredentials: true` against the `CORS_ALLOWED_ORIGINS` allow-list
  (a wildcard origin is incompatible with credentialed requests — validated in config).
- CSRF stance: `SameSite=Lax` + the cookie's narrow `Path` scope is the baseline; a
  double-submit/CSRF token can be layered in Step 8 if we expose state-changing GETs (we don't).
- `COOKIE_SECURE` is `true` by default and only relaxable in development (so local `http://`
  testing works); `COOKIE_DOMAIN` is configurable for the deployed origin.
- Handlers read the refresh token from the cookie (not the body) in `refresh`/`logout`; the
  raw token is set/cleared exclusively through helpers in `render.go` so the attributes stay
  consistent in one place.

## 8. Other Decisions (my discretion — flag to change)

- **Refresh token rotation ON** with reuse-detection (revoke lineage) — a security best
  practice; I'll implement it unless you object.
- **Non-remember-me TTL = 24h** (remember-me = 31d as locked). Confirmable via config.
- Access-token TTL **15m**; bcrypt cost **12**.
- Generic `401` for both unknown email and wrong password (no user-enumeration).
- Logout of an unknown/already-revoked token returns success (no existence leak).

## 9. New Dependencies (go.mod)

- `github.com/golang-jwt/jwt/v5` — access tokens.
- `golang.org/x/crypto/bcrypt` — password hashing.
- `github.com/go-playground/validator/v10` — request validation.
- `github.com/go-chi/cors` — CORS for the SPA.
- (`crypto/rand`, `crypto/sha256`, `encoding/base64` are stdlib.)

## 10. Verification Checklist (how I'll prove it works)

1. `make tidy && go build ./...` — clean.
2. `gofmt` / `go vet` (default **and** `-tags=integration`) / `golangci-lint` — clean.
3. `make test` — unit suite green, `-race`, no Docker.
4. `make test-int` — repository + full end-to-end auth flow green against real `mongo:7`.
5. Live smoke (Docker available here): boot the server, `curl` register → login → `/me` with
   the bearer token → refresh → logout → confirm the old refresh token is now rejected; paste
   the transcript as evidence.

## 10a. Implementation Notes / Deviations (as built)

One deviation from §4 as written, made for correctness:

- **Domain entities are pure; persistence models live in the repository layer.** §4 sketched
  `User`/`Session` in `internal/domain` "embedding `mongo.Document`". That is impossible: the
  Step 2 `platform/mongo` package imports `domain` (for the sentinel errors), so having
  `domain` embed `mongo.Document` creates an import cycle (`domain → mongo → domain`). The
  built design instead keeps `domain.User`/`domain.Session` as **framework-free structs** and
  defines private `userDoc`/`sessionDoc` models (embedding `mongo.Document`) in
  `internal/repository`, mapping between them. This is strictly *more* aligned with the
  "domain has no framework imports" principle, and the generic `Repository[T]` is still used
  (over the doc models). No change to the API, security model, or roadmap.

Everything else was built as planned, including the HttpOnly-cookie decision (§7) and all §8
defaults (refresh rotation + reuse-detection, 24h/31d TTLs, 15m access, bcrypt cost 12,
generic 401).

## 11. What This Unblocks

Step 4 (Topics): a `Topic` entity + repo + service + `/topics` router group, all guarded by
the auth middleware and owned by `UserID(ctx)` — reusing the DTO/validation/error-mapping
template established here without modification.
