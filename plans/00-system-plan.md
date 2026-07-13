# ThisIsYourYear — System Plan

> **Status:** Draft for approval
> **Scope of this document:** Tech stack + high‑level architecture + the ordered, step‑by‑step
> implementation roadmap for the **backend first**, then the frontend.
> Each numbered step in §8 will get its own detailed implementation plan (`plans/NN-*.md`)
> that must be approved before any code is written for it.

---

## 1. Product Summary

A multi‑user web application for tracking yearly goals.

**Core domain concepts**

| Concept | Meaning |
| --- | --- |
| **User** | An authenticated account. Owns everything below. |
| **Topic** (Big Goal) | A high‑level yearly goal/theme (e.g. "Get fit"). |
| **Goal** (Sub‑goal) | A concrete, achievable step under a Topic (e.g. "Run 5km"). |
| **Achievement** | The state of a Goal once completed (a Goal becomes an achievement when marked done). |
| **Motivation** | A message shown when viewing a Topic, chosen by the Topic's completion ratio. |
| **Dashboard** | Landing view: positive message, overall progress, countdown to New Year. |

**Key behaviours**
- A user creates Topics; under each Topic they create Goals.
- Completing a Goal marks it as an achievement and updates the Topic's progress.
- Viewing a Topic shows a motivational message derived from the % of completed Goals.
- The dashboard aggregates status across all Topics + shows a live countdown to Jan 1.

**Non‑functional requirements (from brief)**
- Multi‑user.
- Accessible (WCAG), universal, responsive/adaptive UI.
- Clean architecture, SOLID, conventional, well‑tested, reliable.
- **Expandable** — new tabs/features will be added; the design must not block this.

---

## 2. Confirmed Decisions (locked by you)

| Area | Decision |
| --- | --- |
| Backend language | **Go** (`go1.26`) |
| HTTP foundation | **`net/http` + `chi` router** (stdlib‑first, minimal magic) |
| API style | **REST / JSON**, versioned under `/api/v1` |
| Database | **MongoDB** (official `mongo-go-driver`) |
| Auth | **JWT access tokens + server‑persisted refresh sessions**. "Remember me" issues a refresh session with a **31‑day** expiry; logout/expiry revokes the session server‑side. |
| Frontend | **React + TailwindCSS** (details finalized when you provide designs) |
| Dev runtime | **Docker Compose** (MongoDB now; API + frontend added as built) |

These are treated as fixed for the rest of the plan. Anything that would change one of
them is a **system‑wide decision** and will be brought back to you.

---

## 3. Backend Tech Stack (proposed detail)

> Library choices below are *small technical decisions*. They are listed for transparency;
> flag any you want changed.

| Concern | Choice | Rationale |
| --- | --- | --- |
| Language / runtime | Go 1.26 | Confirmed. |
| Router | `github.com/go-chi/chi/v5` | Confirmed foundation; idiomatic, composable middleware. |
| MongoDB driver | `go.mongodb.org/mongo-driver` (official) | First‑party, supported, context‑aware. |
| Config | `github.com/caarlos0/env` + `.env` (via `godotenv` in dev only) | 12‑factor env config, no config files baked into images. |
| JWT | `github.com/golang-jwt/jwt/v5` | De‑facto standard, maintained. |
| Password hashing | `golang.org/x/crypto/bcrypt` | Standard, safe defaults. |
| Validation | `github.com/go-playground/validator/v10` | Declarative request validation. |
| Logging | `log/slog` (stdlib structured logging) | No third‑party dep; structured, leveled. |
| UUID/IDs | Mongo `ObjectID` for `_id`; expose as hex strings in API | Native to Mongo. |
| Testing | stdlib `testing` + `testify/require` + `testcontainers-go` (Mongo integration) | Unit + real‑DB integration tests. |
| Migrations/Indexes | Custom idempotent index‑ensure on startup | Mongo is schema‑less; indexes are the thing to manage. |
| API docs | OpenAPI 3 spec (hand‑authored, served + checked in) | Contract for the React client. |
| Lint/format | `gofmt`, `go vet`, `golangci-lint` | Enforced in CI. |

---

## 4. Architecture

**Layered / Clean‑ish architecture** (dependencies point inward):

```
cmd/api            -> process entrypoint, wiring
internal/
  domain           -> entities + domain rules (no framework imports)
  service          -> use-cases / business logic (depends on domain + repo interfaces)
  repository       -> interfaces (defined here) + mongo implementations
  transport/http   -> chi handlers, DTOs, middleware, routing (depends on services)
  auth             -> JWT issue/verify, password hashing, session logic
  config           -> env loading
  platform/mongo   -> client, connection, index management
pkg/                -> genuinely reusable, app-agnostic helpers (if any)
```

**Principles applied**
- **Dependency inversion:** services depend on repository *interfaces*, not Mongo.
  Lets us swap/ mock the DB and unit‑test business logic without a database.
- **Single responsibility:** handlers do HTTP only; services do logic; repos do persistence.
- **DTO boundary:** API request/response structs are separate from domain entities (never
  leak DB models or `_id`/internal fields directly).
- **Expandability:** new features = new domain type + repo + service + a new router group
  mounted under `/api/v1`. No existing code needs rewriting.
- **Context propagation:** `context.Context` threaded end‑to‑end (cancellation, timeouts).
- **Errors:** typed domain errors mapped to HTTP status at the transport edge only.

---

## 5. Data Model (initial)

Collections (fields abbreviated; full schema defined in the per‑step plan):

- **users** — `_id, email (unique), passwordHash, displayName, createdAt, updatedAt`
- **sessions** — `_id, userId, refreshTokenHash, userAgent, ip, rememberMe, expiresAt, createdAt, revokedAt` (TTL index on `expiresAt`)
- **topics** — `_id, userId, title, description, year, color/icon, createdAt, updatedAt, archivedAt`
- **goals** — `_id, topicId, userId, title, notes, status (todo|done), completedAt, order, createdAt, updatedAt`

**Progress / motivation** are *computed* (completed goals ÷ total goals per topic), not stored,
so they never drift. Motivation thresholds (e.g. 0%, 1–25%, 26–50%, 51–99%, 100%) map to
message sets — exact copy decided during that step.

**Indexes (initial):** `users.email` unique; `sessions.expiresAt` TTL; `topics.userId`;
`goals.topicId`, `goals.userId`.

---

## 6. API Surface (initial, `/api/v1`)

```
POST   /auth/register
POST   /auth/login            (body: rememberMe bool)
POST   /auth/refresh
POST   /auth/logout
GET    /me

GET    /topics                 list my topics (+ computed progress)
POST   /topics
GET    /topics/{id}            includes goals + progress + motivation
PATCH  /topics/{id}
DELETE /topics/{id}

GET    /topics/{id}/goals
POST   /topics/{id}/goals
PATCH  /goals/{id}             (edit, reorder, mark done/undone)
DELETE /goals/{id}

GET    /dashboard              aggregate status + message + countdown payload
```

Every non‑auth route is owned by the authenticated user; cross‑user access is forbidden.

---

## 7. Cross‑Cutting Concerns

- **Middleware:** request ID, structured access logging, panic recovery, CORS (for the SPA),
  auth (JWT verification), rate limiting (basic, on auth endpoints), request timeout.
- **Security:** bcrypt passwords, refresh tokens stored **hashed**, short‑lived access tokens,
  HttpOnly refresh cookie *or* token response (decided in auth step), input validation,
  no secrets in code, CORS allow‑list.
- **Health/ops:** `/healthz` (liveness) and `/readyz` (Mongo ping).
- **Testing strategy:** unit tests for services (mock repos), integration tests for repos +
  handlers (testcontainers Mongo). CI runs lint + vet + tests.

---

## 8. Implementation Roadmap (ordered steps)

> We execute **one step at a time**. For each, I will first write a detailed
> `plans/NN-*.md`, get your approval, implement, and wait for your **"Task completed"**
> before moving on.

**Phase A — Backend**

1. **Project bootstrap & tooling** — Go module, folder skeleton, config loading, `slog`,
   Makefile, `golangci-lint`, Docker Compose with MongoDB, `/healthz` + `/readyz`,
   graceful shutdown. *Outcome: server boots, connects to Mongo, CI green.*
2. **Persistence foundation** — Mongo client wiring, repository interfaces, index management,
   base entity/DTO conventions, error types, testcontainers harness.
3. **Auth & users** — register/login, bcrypt, JWT access tokens, refresh sessions with
   "Remember me" (31‑day), refresh/logout, auth middleware, `/me`.
4. **Topics** — CRUD, ownership enforcement, validation, progress computation scaffolding.
5. **Goals & achievements** — CRUD under topics, mark done/undone, ordering, completedAt.
6. **Motivation engine** — threshold→message mapping, surfaced on topic view.
7. **Dashboard endpoint** — aggregate progress, positive message, New‑Year countdown data.
8. **Hardening** — rate limiting, full OpenAPI spec, end‑to‑end tests, error‑path coverage.
9. **Design data extensions** — additive API fields the design handoff requires: Topic
   `category` + `targetDate`, Goal `dueDate` + `isAchievement` (the ✓-vs-★ flag). Domain +
   repo + service + DTOs + OpenAPI + tests. *Closes the design ↔ API gap before the frontend.*

**Phase B — Frontend** (detailed roadmap in [`plans/frontend/00-system-plan.md`](./frontend/00-system-plan.md))

10. Frontend foundation — React + Vite + TS + Tailwind scaffold, design‑system/tokens (both
    themes), app shell, routing, API client, auth flow.
11. Feature UI — Dashboard, Topic/Goal flows, Achievements, celebration + motivation system,
    accessibility pass, responsive/adaptive polish.

**Phase C — Delivery**

12. Production Docker images, compose/orchestration, env/secrets, deployment runbook.

---

## 9. Open Questions Deferred to You (not blocking this plan)

These will be raised at the relevant step, not decided unilaterally:
- Refresh token transport: **HttpOnly cookie** vs **JSON body** (security/SPA trade‑off).
- Exact motivation message copy and threshold bands.
- Whether Topics are scoped per‑year (multi‑year history) from day one or added later.
- Hosting/deployment target for Phase C.

---

## 10. Definition of Done (per step)

A step is complete when: code compiles, `gofmt`/`vet`/`golangci-lint` pass, unit +
integration tests for the step pass, the feature is reachable end‑to‑end, and docs
(OpenAPI/README) are updated.
