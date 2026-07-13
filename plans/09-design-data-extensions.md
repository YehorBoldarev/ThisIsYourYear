# Step 9 — Design Data Extensions

> **Phase:** A (Backend) · **Status:** Draft for approval
> **Parent:** [`00-system-plan.md`](./00-system-plan.md) §8.9
> **Goal of this step:** Add the small, **additive** API fields the design handoff requires — so the
> React SPA (Phase B) has everything it needs — without changing any existing behavior. Two fields on
> Topics and two on Goals.
>
> **Out of scope:** Any frontend. The Reflections/Notes feature (a whole new domain) is **not** built.
> The Achievements page will be **derived client-side** from these fields, so no new endpoint here.

---

## 1. Why this step exists

The `design_handoff_this_is_your_year/` designs assume data the current API doesn't expose. Analysis
against the backend found exactly four missing fields (everything else the design needs is already
served or derivable):

| Design element | Field to add |
| --- | --- |
| "Category" on the Create-big-goal form; topic grouping | **Topic.category** |
| "Target date" on the Create-big-goal form | **Topic.targetDate** |
| "Target date — optional" on the Add-milestone form; "Due Oct 4" in the milestone list | **Goal.dueDate** |
| "Celebrate as an achievement ★" toggle; the **✓ done vs ★ ACHIEVED** distinction in the list | **Goal.isAchievement** |

All four are **optional and backward-compatible**: existing documents without them read back as
empty/false, and every current request stays valid. This is the last backend step before Phase B.

---

## 2. Outcome / Definition of Done

- **Topic** gains `category` (optional text) and `targetDate` (optional date). Settable on create,
  changeable on `PATCH`, returned in `TopicResponse`.
- **Goal** gains `dueDate` (optional date) and `isAchievement` (bool, default `false`). Settable on
  create, changeable on `PATCH`, returned in `GoalResponse`.
- The **✓-vs-★** rule stays a frontend concern: the backend only stores `isAchievement`; a milestone
  renders ★ when `isAchievement && done`, ✓ when `done && !isAchievement`. No backend logic change to
  completion or progress.
- `api/openapi.yaml` documents all four fields; the OpenAPI parse-guard stays green.
- `go build`, `go vet` (both tags), `gofmt`, `golangci-lint` (both tags), `make test` pass;
  `make test-int` passes against real `mongo:7`, asserting the fields round-trip create → read →
  patch → read.
- `docs/dev_run.md` updated (field notes + changelog).

---

## 3. Design

### 3.1 Date representation — `DD-MM-YYYY` strings (DECIDED, §7)
`targetDate` and `dueDate` are **calendar dates**, not instants ("Dec 31, 2026", "Due Oct 4"). We
model them at the API boundary as **date-only strings in `DD-MM-YYYY` form** (`"31-12-2026"`),
validated with the validator `datetime=02-01-2006` rule (Go's reference date: `02`=day, `01`=month,
`2006`=year), and store them as a normalized **UTC-midnight `*time.Time`** in Mongo (nil = unset).
Date-only avoids the timezone day-shift bug that an RFC3339 instant would introduce (a local-midnight
value sent to the server could land on the previous day in UTC). The domain keeps `*time.Time`; the
DTO owns the string↔time mapping and formats output with `t.Format("02-01-2006")`. (Note: the SPA's
`<input type="date">` yields ISO `YYYY-MM-DD`, so the frontend converts to/from `DD-MM-YYYY` at the
API edge — handled in Phase B.)

### 3.2 Topic changes
- **Domain (`topic.go`):** add `Category string` and `TargetDate *time.Time`. `NewTopic` accepts and
  validates them (`category` trimmed, `TopicCategoryMaxLen = 60`; `targetDate` any valid date).
  `TopicUpdate` gains `Category *string` and `TargetDate **time.Time`-style optionality — using a
  small `Optional[time.Time]` isn't warranted; instead `TargetDate *time.Time` with a companion
  `ClearTargetDate bool` **or** the simpler convention below. **Chosen convention:** `TopicUpdate`
  carries `Category *string` and `TargetDate *time.Time`; sending `targetDate: null` in the PATCH
  clears it, `omitted` leaves it unchanged (the DTO distinguishes absent from null via a pointer +
  presence flag). This matches how `Archived`/partial fields already work.
- **Repository (`topics.go`):** `topicDoc` gains `Category string \`bson:"category,omitempty"\`` and
  `TargetDate *time.Time \`bson:"targetDate,omitempty"\``; `toDomain`, `Create`, and
  `topicUpdateFields` map them. No new index.
- **Service (`topic_service.go`):** `Create` accepts the two new inputs. To avoid positional-arg
  bloat, `Create` switches to a `service.CreateTopicInput` struct (title, description, year, color,
  icon, **category, targetDate**); `Update` already takes `domain.TopicUpdate`, which now carries the
  new fields. Progress/motivation unchanged.

### 3.3 Goal changes
- **Domain (`goal.go`):** add `DueDate *time.Time` and `IsAchievement bool`. `NewGoal` accepts them
  (`isAchievement` default false; `dueDate` optional). `GoalUpdate` gains `DueDate *time.Time` (null
  clears, absent leaves) and `IsAchievement *bool`. Completion logic (`Done`, `completedAt`) is
  untouched — `isAchievement` is independent metadata.
- **Repository (`goals.go`):** `goalDoc` gains `DueDate *time.Time \`bson:"dueDate,omitempty"\`` and
  `IsAchievement bool \`bson:"isAchievement"\``; `toDomain`, `Create`, `goalUpdateFields` map them.
- **Service (`goal_service.go`):** `Create` accepts `isAchievement` + `dueDate` (via a
  `service.CreateGoalInput` struct for the same anti-bloat reason); `Update` takes the extended
  `domain.GoalUpdate`.

### 3.4 Transport / DTO
- `CreateTopicRequest`: `+ category string \`validate:"omitempty,max=60"\``,
  `+ targetDate string \`validate:"omitempty,datetime=02-01-2006"\``.
- `UpdateTopicRequest`: `+ category *string`, `+ targetDate *string` (null-clears; presence tracked).
- `TopicResponse`: `+ category string`, `+ targetDate string` (`DD-MM-YYYY`, empty when unset).
- `CreateGoalRequest`: `+ dueDate string (datetime=02-01-2006)`, `+ isAchievement bool`.
- `UpdateGoalRequest`: `+ dueDate *string`, `+ isAchievement *bool`.
- `GoalResponse`: `+ dueDate string`, `+ isAchievement bool`.
- The DTO layer parses/formats dates and maps to the domain/service inputs; handlers stay thin.

> **Null vs absent in PATCH:** to support "clear this date", the update DTOs decode into a type that
> distinguishes an omitted key from an explicit `null`. We already decode with
> `DisallowUnknownFields`; the cleanest local approach is a tiny generic `nullable[T]` wrapper (or a
> custom `UnmarshalJSON`) on the two optional date fields. Kept internal to the transport package.

### 3.5 OpenAPI
Add the fields to `CreateTopicRequest`, `UpdateTopicRequest`, `Topic`, `CreateGoalRequest`,
`UpdateGoalRequest`, and `Goal` schemas. The date strings are `type: string` with
`pattern: '^\d{2}-\d{2}-\d{4}$'` + an `example: "31-12-2026"` (not `format: date`, which implies ISO
`YYYY-MM-DD`), and clear descriptions incl. the null-clears-on-PATCH semantics. The parse-guard test
already covers that the spec stays valid YAML.

---

## 4. Files

```
internal/
  domain/
    topic.go              # + Category, TargetDate; NewTopic + TopicUpdate + validation
    topic_test.go         # + category maxlen, date validity, update-clears
    goal.go               # + DueDate, IsAchievement; NewGoal + GoalUpdate
    goal_test.go          # + new-field construction & update
  repository/
    topics.go             # topicDoc fields + mapping
    goals.go              # goalDoc fields + mapping
  service/
    topic_service.go      # CreateTopicInput struct; pass-through
    topic_service_test.go # updated calls + a new-field assertion
    goal_service.go       # CreateGoalInput struct; pass-through
    goal_service_test.go  # updated calls
  transport/http/
    topic_dto.go          # request/response fields + date mapping + nullable date
    goal_dto.go           # request/response fields + date mapping + nullable date
    topic_handler.go      # thread new fields into service (via input struct)
    goal_handler.go       # thread new fields into service
    topic_flow_integration_test.go  # round-trip category + targetDate (incl. clear)
    goal_flow_integration_test.go   # round-trip dueDate + isAchievement (incl. clear)
api/openapi.yaml          # schema fields
docs/dev_run.md           # field notes + changelog
```

No new endpoints, collections, indexes, or dependencies.

---

## 5. API shape (additions)

`POST /api/v1/topics` (additions):
```json
{ "title": "Run a Half-Marathon", "category": "Health & fitness", "targetDate": "31-12-2026" }
```
`TopicResponse` gains `"category": "Health & fitness", "targetDate": "31-12-2026"` (empty string when
unset).

`POST /api/v1/topics/{id}/goals` (additions):
```json
{ "title": "Complete a 15K training run", "dueDate": "09-08-2026", "isAchievement": true }
```
`GoalResponse` gains `"dueDate": "09-08-2026", "isAchievement": true`.

`PATCH` on either accepts the same fields; sending `"targetDate": null` / `"dueDate": null` clears it.

---

## 6. Testing Strategy

| Layer | What | Runs in |
| --- | --- | --- |
| domain | category maxlen; date fields optional; invalid date rejected; `NewGoal` defaults `isAchievement=false`; updates set/clear each field | `make test` |
| service | create-with-new-fields persists; update changes and clears them; progress/motivation unaffected | `make test` |
| transport | request validation (`datetime`, `max=60`); date format on output; PATCH null-clears vs absent-leaves | `make test` |
| end-to-end | create topic+goal with the fields → read them back → PATCH (change + clear) → read → assert; existing flows unchanged | `make test-int` |

## 7. Decisions — DECIDED

- **(a) Date representation — LOCKED: date-only `DD-MM-YYYY` strings** (`"31-12-2026"`), validated
  `datetime=02-01-2006`, stored as UTC-midnight `*time.Time`. Chosen over RFC3339 instants to avoid a
  one-day timezone shift.
- **(b) `isAchievement` editable on PATCH — LOCKED: yes.** `UpdateGoalRequest` carries
  `isAchievement *bool`, so it can be toggled after creation.

Everything else (field names matching the design vocabulary, additive/optional, no new indexes, the
`nullable` decode helper, input structs to avoid positional bloat) is small technical discretion.

## 8. Verification Checklist

1. `make tidy && go build ./...` — clean.
2. `gofmt` / `go vet` (both tags) / `golangci-lint` (both tags) — clean.
3. `make test` — unit suite green, `-race`.
4. `make test-int` — the four fields round-trip (create → read → patch/clear → read) against real
   `mongo:7`; existing topic/goal/dashboard flows still green.
5. Live smoke: create a topic with `category` + `targetDate` and a goal with `dueDate` +
   `isAchievement`, read both back, PATCH to change and to clear a date, confirm the JSON. Fetch
   `/openapi.yaml` and confirm the new fields are documented. Update `docs/dev_run.md` and re-verify.

## 9. What This Unblocks

Closes the design ↔ API gap. With these fields live and documented in the OpenAPI contract, Phase B
can generate its typed client and build the Create-big-goal / Add-milestone / Topic-detail screens at
full design fidelity. After your **"Task completed"** on this step, we begin **Phase B, Step B1**
(app scaffold & design system) per [`plans/frontend/00-system-plan.md`](./frontend/00-system-plan.md).
