# Step 4 — Topics

> **Phase:** A (Backend) · **Status:** Draft for approval
> **Parent:** [`00-system-plan.md`](./00-system-plan.md) §8.4, §5, §6
> **Goal of this step:** Deliver the first user-owned resource — **Topics** (big yearly goals)
> — as a full CRUD feature built on the Step 3 auth + HTTP template: create/list/get/update/
> delete, strict **per-user ownership**, request validation, and the **progress value object**
> that Steps 5–6 will populate.
>
> **Out of scope:** Goals themselves and their endpoints (Step 5); motivation messages
> (Step 6); the dashboard aggregate (Step 7). In this step a topic has no goals yet, so its
> computed progress is correctly `0%`.

---

## 1. Why this step exists

Topics are the parent of Goals and the anchor of the whole product. This step proves the
"new feature = new domain type + repo + service + router group" template from Step 3 works
unchanged, and establishes two things every later step reuses: **ownership-scoped queries**
(a user can only ever see or touch their own rows) and the **`Progress` value object**
(completed ÷ total, division-by-zero-safe) that powers both the topic view and the dashboard.

---

## 2. Outcome / Definition of Done

- `POST /api/v1/topics` creates a topic owned by the caller (201).
- `GET /api/v1/topics` lists **only the caller's** topics, each with computed progress.
- `GET /api/v1/topics/{id}` returns one of the caller's topics; another user's id → `404`
  (existence is never leaked).
- `PATCH /api/v1/topics/{id}` partially updates title/description/year/color/icon and
  archive state; validation errors → `422`.
- `DELETE /api/v1/topics/{id}` removes the caller's topic (`204`).
- All five routes require a valid access token (mounted behind the Step 3 auth middleware);
  every query is scoped by `userId`.
- A `domain.Progress` value object exists with a tested `ComputeProgress(total, completed)`
  that returns `0.0` ratio for zero goals (no NaN).
- `go build`, `go vet`, `gofmt`, `golangci-lint`, `make test` pass; `make test-int` passes
  against real `mongo:7`, covering CRUD, ownership isolation, and validation over HTTP.
- `docs/dev_run.md` updated (endpoints table, smoke test, changelog).

---

## 3. Data Model (this step)

### `topics`
| Field | BSON | Notes |
| --- | --- | --- |
| id | `_id` | ObjectID |
| userId | `userId` | owner (ObjectID); every query filters on it |
| title | `title` | required, ≤120 chars |
| description | `description` | optional, ≤2000 chars |
| year | `year` | int; defaults to current UTC year if omitted; range 2000–2100 |
| color | `color` | optional; hex `#RRGGBB` if present |
| icon | `icon` | optional; short string (icon name/emoji), ≤64 chars |
| archivedAt | `archivedAt` | nil = active; set = archived |
| createdAt / updatedAt | — | from `mongo.Document` |

**Indexes registered this step:**
- `topics`: `{ userId: 1, year: 1 }` — serves both "list my topics" and year-filtered lists.
- `topics`: `{ userId: 1, archivedAt: 1 }` — efficient active/archived filtering.

Progress is **never stored** (it would drift); it is computed per request.

---

## 4. Architecture & Files

Mirrors the Step 3 layering exactly. New files, plus a couple of touch-ups to wire the group.

```
internal/
  domain/
    topic.go                 # Topic entity + NewTopic (validation, year default)
    topic_test.go
    progress.go              # Progress value object + ComputeProgress (div-by-zero safe)
    progress_test.go

  repository/
    topics.go                # TopicRepository over Repository[topicDoc,*topicDoc]; ownership-scoped queries; index registration
    topics_integration_test.go

  service/
    topic_service.go         # TopicService: Create/List/Get/Update/Delete, all keyed by userID
    topic_service_test.go    # unit with in-memory fake repo

  transport/http/
    topic_dto.go             # Create/Update requests, TopicResponse (+ progress)
    topic_handler.go         # handlers: thin decode→validate→service→render
    router.go                # CHANGED: mount /topics inside the authenticated group
    topic_flow_integration_test.go   # CRUD + ownership over real HTTP + Mongo

  service/ or transport wiring in cmd/api/main.go   # CHANGED: construct TopicRepository/TopicService, pass into RouterDeps

docs/dev_run.md              # CHANGED: endpoints, smoke test, changelog
```

### 4.1 Domain
- **`Topic`**: pure struct (`ID, UserID, Title, Description, Year, Color, Icon, ArchivedAt,
  CreatedAt, UpdatedAt`). `NewTopic(userID, title, description, year, color, icon)` trims and
  validates (title required/length, year range, hex color if present), defaulting `year` to
  the current UTC year when zero. Invalid input → `ErrValidation`.
- **`Progress`**: `{ Total, Completed int; Ratio float64 }`. `ComputeProgress(total,
  completed int) Progress` clamps and guards `total == 0 → Ratio 0.0`. This is the shared
  primitive for the topic view (Step 6 motivation) and dashboard (Step 7).

### 4.2 Repository (ownership-first)
- `topicDoc` embeds `mongo.Document`; `userId` stored as ObjectID (mapped like sessions).
- `TopicRepository` methods **all take `userID`** and filter on it, so cross-user access is
  structurally impossible:
  - `Create(ctx, *domain.Topic)`
  - `ByIDForUser(ctx, id, userID)` — filter `{_id, userId}`; miss → `ErrNotFound`
  - `ListForUser(ctx, userID, filter)` — `filter` carries optional year + includeArchived
  - `UpdateForUser(ctx, id, userID, fields)` — filter `{_id, userId}`; miss → `ErrNotFound`
  - `DeleteForUser(ctx, id, userID)` — filter `{_id, userId}`; miss → `ErrNotFound`
- Registers the two indexes above at construction.

### 4.3 Service
- `TopicService` orchestrates topic use-cases; depends on a `TopicRepository` interface it
  declares. Every method receives the `userID` (the handler pulls it from `UserID(ctx)`).
- **Progress in this step:** a topic has no goals yet, so the service builds each response's
  progress as `ComputeProgress(0, 0)` → `0%`. This is a *correct* value, not a placeholder.
  **Seam for Step 5:** Step 5 injects a goal-counting collaborator (`GoalStats`) into
  `TopicService`; the response builder then calls `ComputeProgress(total, completed)` with
  real counts. No response shape changes between steps — only the numbers become non-zero.

### 4.4 Transport
- `topic_dto.go`:
  - `CreateTopicRequest{ Title, Description, Year, Color, Icon }` with validate tags.
  - `UpdateTopicRequest` uses **pointer fields** so PATCH updates only what's supplied; a
    supplied title must still be non-empty.
  - `TopicResponse{ ID, Title, Description, Year, Color, Icon, Archived, Progress, CreatedAt,
    UpdatedAt }`; `Progress` is `{ total, completed, ratio }`.
- `topic_handler.go`: five thin handlers reusing `decodeJSON`, `validateStruct`,
  `respondError`, `writeJSON`, and `UserID(ctx)`.
- `router.go`: mount `/topics` **inside the existing authenticated group** (`protected.Use(
  authenticator.Middleware)`), next to `/me`. No existing routes change.

### 4.5 Wiring
`main.go` constructs `TopicRepository` (registers indexes before `EnsureIndexes`) and
`TopicService`, and passes them into `RouterDeps`.

---

## 5. Endpoints (this step)

| Method | Path | Body / Query | Success |
| --- | --- | --- | --- |
| `POST` | `/api/v1/topics` | `{title, description?, year?, color?, icon?}` | `201` TopicResponse |
| `GET` | `/api/v1/topics` | `?year=`, `?includeArchived=true` | `200` `[]TopicResponse` |
| `GET` | `/api/v1/topics/{id}` | — | `200` TopicResponse |
| `PATCH` | `/api/v1/topics/{id}` | any subset of updatable fields (+ `archived` bool) | `200` TopicResponse |
| `DELETE` | `/api/v1/topics/{id}` | — | `204` |

All require `Authorization: Bearer <access>`. `{id}` not owned by the caller → `404`.

---

## 6. Testing Strategy

| Layer | What | Runs in |
| --- | --- | --- |
| domain | `NewTopic` validation + year default; `ComputeProgress` incl. 0-total, full, partial | `make test` |
| service | create/list/get/update/delete logic; ownership passed through; not-found mapping | `make test` (fake repo) |
| repository | CRUD, **cross-user isolation** (user B cannot read/update/delete user A's topic), year/archived filters, indexes present | `make test-int` |
| end-to-end | register→login→create→list→get→patch→delete over real HTTP; a second user gets 404 on the first user's topic | `make test-int` |

The cross-user isolation assertions are the security core of this step.

## 7. Decisions Within This Step (my discretion — flag to change)

- **Progress = `ComputeProgress(0,0)` for now**, with a documented Step-5 injection seam
  (vs. omitting `progress` from responses until goals exist). Recommendation: include it — the
  API shape stays stable for the SPA.
- **DELETE is a hard delete.** With no goals yet there is nothing to cascade; Step 5 adds
  goal cascade-on-delete. Archiving (soft) is available via `PATCH {archived:true}`.
- **List excludes archived by default**, opt in with `?includeArchived=true`.
- Validation limits: title ≤120, description ≤2000, icon ≤64, color `#RRGGBB`, year 2000–2100.

## 8. Per-Year Scoping — DECIDED: year-aware now (A)

**Locked by you:** the Topics list is year-scoped from day one.

- `GET /api/v1/topics` returns the caller's topics for the **current UTC year** by default.
- `GET /api/v1/topics?year=YYYY` returns that year's topics (validated to the 2000–2100 range).
- On create, `year` defaults to the current UTC year when omitted.
- The `{ userId: 1, year: 1 }` index serves these queries directly.

Combined with the archived filter, the effective default list query is
`{ userId, year: <current-or-requested>, archivedAt: nil }`, with `?includeArchived=true`
dropping the `archivedAt` clause.

## 9. New Dependencies

None. Everything reuses the existing stack (chi, mongo driver, validator).

## 10. Verification Checklist

1. `make tidy && go build ./...` — clean.
2. `gofmt` / `go vet` (default **and** `-tags=integration`) / `golangci-lint` — clean.
3. `make test` — unit suite green, `-race`.
4. `make test-int` — CRUD + ownership-isolation + validation green against real `mongo:7`.
5. Live smoke: create/list/get/patch/delete a topic with a real bearer token; confirm a second
   account gets `404` on the first account's topic. Update `docs/dev_run.md` and re-verify its
   commands. Paste evidence.

## 11. What This Unblocks

Step 5 (Goals & achievements): goals live under a topic, reusing this ownership pattern, and
inject the `GoalStats` collaborator that turns topic `progress` from `0%` into live
completion — with no change to the topic response contract.
