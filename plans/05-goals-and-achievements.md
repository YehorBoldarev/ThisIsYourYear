# Step 5 — Goals & Achievements

> **Phase:** A (Backend) · **Status:** Draft for approval
> **Parent:** [`00-system-plan.md`](./00-system-plan.md) §8.5, §5, §6
> **Goal of this step:** Add **Goals** (the concrete sub-goals under a Topic), including
> create/list/edit/delete, **mark done/undone** (a done goal *is* an achievement), and
> **ordering**. Completing this step also makes **topic progress live** — the Step-4 seam is
> filled with real goal counts — and makes **deleting a topic cascade** to its goals.
>
> **Out of scope:** Motivation messages (Step 6) and the dashboard aggregate (Step 7). A goal
> carries no motivational copy yet; this step only produces the counts those steps consume.

---

## 1. Why this step exists

Goals are where the product's value lives: a user breaks a Topic into achievable steps and
completes them. This step delivers that, and closes the loop opened in Step 4 — topic
`progress` stops being a constant `0%` and becomes `completed ÷ total` over real goals,
computed (never stored) so it can't drift. It reuses the Step 3–4 template wholesale
(ownership-scoped queries, DTO/validation, the error mapper).

---

## 2. Outcome / Definition of Done

- `POST /api/v1/topics/{topicId}/goals` creates a goal under one of the caller's topics
  (201); a topic the caller doesn't own → `404`.
- `GET /api/v1/topics/{topicId}/goals` lists that topic's goals in `order`.
- `PATCH /api/v1/goals/{id}` edits title/notes, **reorders**, and **marks done/undone**
  (done stamps `completedAt`; undone clears it).
- `DELETE /api/v1/goals/{id}` removes a goal (204).
- Every goal route requires auth and is scoped to the caller; another user's goal/topic id
  → `404` (existence never leaked).
- **Topic progress is live:** `GET /topics` and `GET /topics/{id}` return
  `progress = {total, completed, ratio}` computed from the topic's goals.
- **Cascade:** deleting a topic deletes its goals.
- `go build`, `go vet`, `gofmt`, `golangci-lint`, `make test` pass; `make test-int` passes
  against real `mongo:7`, covering goal CRUD, done/undone, ownership isolation, live progress,
  and topic-delete cascade.
- `docs/dev_run.md` updated (endpoints, smoke test, changelog).

---

## 3. Data Model (this step)

### `goals`
| Field | BSON | Notes |
| --- | --- | --- |
| id | `_id` | ObjectID |
| topicId | `topicId` | parent topic (ObjectID) |
| userId | `userId` | owner (ObjectID); denormalized so goal queries scope by user directly |
| title | `title` | required, ≤200 chars |
| notes | `notes` | optional, ≤5000 chars |
| status | `status` | `todo` \| `done` |
| completedAt | `completedAt` | set when status→done, cleared when →todo |
| order | `order` | int; new goals appended at the end; client PATCHes to reorder |
| createdAt / updatedAt | — | from `mongo.Document` |

**Index registered this step:**
- `goals`: `{ userId: 1, topicId: 1, order: 1 }` — one compound index that serves the
  ordered list within a topic, the progress counts (prefix `userId, topicId`), ownership
  checks, and the cascade delete. (Covers the master plan's `goals.topicId` / `goals.userId`
  intent via prefixes; no separate single-field indexes needed.)

An **achievement** is not a separate collection — it is simply a goal with `status: done` and a
`completedAt`. This keeps progress and (later) the dashboard trivially derivable.

---

## 4. Architecture & Files

Same layering as Steps 3–4. New files, plus focused changes to `TopicService` (progress +
cascade) and the router/wiring.

```
internal/
  domain/
    goal.go                  # Goal entity, GoalStatus, NewGoal, GoalUpdate (+ Validate)
    goal_test.go

  repository/
    goals.go                 # GoalRepository: ownership-scoped CRUD, ordered list, counts (aggregation), DeleteByTopic
    goals_integration_test.go

  service/
    goal_service.go          # GoalService: create/list/edit/delete, done/undone, order assignment; verifies topic ownership
    goal_service_test.go
    topic_service.go         # CHANGED: inject a goals port -> live progress + cascade-on-delete
    topic_service_test.go    # CHANGED: fake goals port

  transport/http/
    goal_dto.go              # Create/Update requests, GoalResponse
    goal_handler.go          # handlers (thin)
    router.go                # CHANGED: mount goal routes in the authenticated group
    goal_flow_integration_test.go
    topic_flow_integration_test.go  # CHANGED: assert progress reflects goals

cmd/api/main.go              # CHANGED: build GoalRepository/GoalService; inject goals port into TopicService
docs/dev_run.md              # CHANGED
```

### 4.1 Domain
- **`Goal`**: pure struct (`ID, TopicID, UserID, Title, Notes, Status, CompletedAt, Order,
  CreatedAt, UpdatedAt`). `GoalStatus` is `todo`/`done`. Helper `IsAchieved()`.
- `NewGoal(userID, topicID, title, notes)` trims/validates (title required/≤200, notes ≤5000),
  starts `todo` with nil `completedAt`. Order is assigned by the service on create.
- **`GoalUpdate`** partial: `Title *string`, `Notes *string`, `Done *bool`, `Order *int`,
  with `Validate()` and `IsEmpty()`. `Done` is the transition lever (true→done+now,
  false→todo+clear); the timestamp is applied in the repository so it stays a storage detail.

### 4.2 Repository (ownership-first)
- `goalDoc` embeds `mongo.Document`; `topicId`/`userId` as ObjectIDs.
- `GoalRepository` methods, all scoped by `userID`:
  - `Create`, `ByIDForUser(id, userID)`, `ListForTopic(userID, topicID)` (sorted by `order`,
    `createdAt` tiebreak), `UpdateForUser(id, userID, GoalUpdate)`, `DeleteForUser(id, userID)`.
  - `DeleteByTopic(userID, topicID)` — cascade support (best-effort bulk delete).
  - `CountForTopic(userID, topicID)` — used to append new goals at the end.
  - `CountsByTopics(userID, topicIDs)` — a **single aggregation** returning `{topicId →
    {total, completed}}` (group + `$cond` on status), so listing N topics is one round-trip,
    not N. Topics with no goals are absent from the result → default `0/0`.
- Reuses the generic `UpdateBy`/`DeleteBy` added in Step 4 for ownership-scoped writes.

### 4.3 Services
- **`GoalService`** depends on `GoalRepository` (interface) **and** a topic-ownership checker
  (interface satisfied by `TopicRepository.ByIDForUser`), so it can reject a goal aimed at a
  topic the caller doesn't own before touching the goals collection. Create assigns `order =
  CountForTopic(...)` (append). Update maps `Done` to the status transition.
- **`TopicService` (changed):** gains a `goals` port:
  ```go
  type TopicGoals interface {
      CountsByTopics(ctx, userID string, topicIDs []string) (map[string]domain.Progress, error)
      DeleteByTopic(ctx, userID, topicID string) error
  }
  ```
  `Get`/`List` now compute progress from real counts (List batches via one `CountsByTopics`).
  `Delete` removes the topic (ownership-checked) then cascades `DeleteByTopic`. `Create`
  returns `0%` (a brand-new topic has no goals) without a query.

### 4.4 Transport
- `goal_dto.go`: `CreateGoalRequest{ Title, Notes }`; `UpdateGoalRequest{ Title*, Notes*,
  Done*, Order* }` (pointers = partial); `GoalResponse{ id, topicId, title, notes, status,
  completedAt, order, createdAt, updatedAt }`.
- `goal_handler.go`: five thin handlers reusing the shared decode/validate/render helpers and
  `chi.URLParam` for `{topicId}` / `{id}`.
- `router.go`: mount inside the existing authenticated group:
  - `POST /topics/{topicId}/goals`, `GET /topics/{topicId}/goals`
  - `PATCH /goals/{id}`, `DELETE /goals/{id}`

### 4.5 Wiring
`main.go` builds `GoalRepository` (registers its index before `EnsureIndexes`) and
`GoalService`, and passes the goal repo into `TopicService` as the `TopicGoals` port.

---

## 5. Endpoints (this step)

| Method | Path | Body | Success |
| --- | --- | --- | --- |
| `POST` | `/api/v1/topics/{topicId}/goals` | `{title, notes?}` | `201` GoalResponse |
| `GET` | `/api/v1/topics/{topicId}/goals` | — | `200` `[]GoalResponse` (ordered) |
| `PATCH` | `/api/v1/goals/{id}` | subset of `{title, notes, done, order}` | `200` GoalResponse |
| `DELETE` | `/api/v1/goals/{id}` | — | `204` |

All require a Bearer access token; non-owned `{topicId}`/`{id}` → `404`.

---

## 6. Testing Strategy

| Layer | What | Runs in |
| --- | --- | --- |
| domain | `NewGoal` validation; `GoalUpdate.Validate`; done/undone semantics | `make test` |
| service | create appends order; done stamps/clears completedAt; ownership passthrough; topic-not-owned → 404; **TopicService live progress + cascade** (fake goals port) | `make test` |
| repository | CRUD, ordered list, `CountsByTopics` aggregation, `DeleteByTopic`, cross-user isolation | `make test-int` |
| end-to-end | create topic → add goals → list → mark done (progress rises) → reorder → delete goal → delete topic cascades goals; a second user gets 404 on another's goals | `make test-int` |

Security core: a user can never create/read/update/delete goals under a topic they don't own,
and progress/counts are always user+topic scoped.

## 7. Decisions Within This Step (my discretion — flag to change)

- **Ordering is an explicit settable `order` int** (new goals appended; client PATCHes `order`
  to move a goal). Full drag-drop with automatic sibling re-normalization is deferred as a
  later enhancement — this keeps the step focused while supporting reordering.
- **Topic detail returns progress, not an embedded goals array.** Goals are fetched via
  `GET /topics/{id}/goals`. (The master plan's "detail includes goals" is satisfiable by the
  SPA with one extra call; embedding is easy to add later if you'd rather.) Motivation copy is
  added to topic detail in Step 6.
- **Cascade = hard delete** of a topic's goals when the topic is deleted; the topic delete is
  ownership-checked first.
- Limits: title ≤200, notes ≤5000.

## 8. System-Wide Decisions

None new. Motivation thresholds/copy are a Step 6 decision; nothing here changes a locked
choice. (If you'd prefer topic detail to embed its goals array, that's a small shape change —
say so and I'll fold it in.)

## 9. New Dependencies

None.

## 10. Verification Checklist

1. `make tidy && go build ./...` — clean.
2. `gofmt` / `go vet` (default **and** `-tags=integration`) / `golangci-lint` — clean.
3. `make test` — unit suite green, `-race`.
4. `make test-int` — goal CRUD + done/undone + ownership isolation + live progress + cascade,
   green against real `mongo:7`.
5. Live smoke: create a topic, add goals, mark some done and watch the topic's `progress.ratio`
   rise, reorder, delete a goal, delete the topic and confirm its goals are gone; confirm a
   second account gets `404` on the first account's goals. Update `docs/dev_run.md` and
   re-verify its commands.

## 11. What This Unblocks

Step 6 (Motivation): maps a topic's now-live progress ratio to a motivational message on the
topic view. Step 7 (Dashboard): aggregates progress and achievements across topics using the
same counts primitive.
