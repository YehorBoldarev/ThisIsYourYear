# Step 7 — Dashboard Endpoint

> **Phase:** A (Backend) · **Status:** Draft for approval
> **Parent:** [`00-system-plan.md`](./00-system-plan.md) §8.7, §6 (`GET /dashboard`), §9
> **Goal of this step:** A single landing endpoint — `GET /dashboard` — that returns the
> user's **overall status for the current year**: aggregate progress across their topics, a
> positive message derived from that progress, and the data to render a **New-Year countdown**.
>
> **Out of scope:** Any React/UI (Phase B). This step delivers the JSON the dashboard screen
> will consume. No new persistence — it composes what Steps 4–6 already built.

---

## 1. Why this step exists

The dashboard is the app's front door. Instead of opening one topic, the user sees "how is my
year going, overall?" in one glance: a headline message, one progress bar across everything,
and a ticking countdown to Jan 1. This step turns the per-topic primitives from Steps 4–6 into
one aggregate view.

It stays true to the architecture: the aggregate math is a **pure domain function**, the
service **reuses the existing topic-list path** (so progress stays live and correct), and the
transport layer just projects the result. No new collections, indexes, or queries beyond what
`TopicService.List` already issues.

---

## 2. Outcome / Definition of Done

- `GET /dashboard` (authenticated) returns, for the caller's **current-year, non-archived**
  topics:
  - `progress` — aggregate `{total, completed, ratio}` summed across all their goals.
  - `topics` — `{total, completed}` (how many topics exist, how many are 100% done).
  - `message` — a positive message derived from the overall progress (and a distinct
    empty-state line when the user has no topics yet).
  - `countdown` — `{target, secondsRemaining}` to the next New Year.
  - `year` — the year this dashboard summarizes.
- Aggregation is a **pure, unit-tested** domain function; the countdown is a pure function of
  "now".
- `go build`, `go vet`, `gofmt`, `golangci-lint`, `make test` pass; `make test-int` passes
  against real `mongo:7`, asserting the dashboard reflects created/completed goals end-to-end.
- `docs/dev_run.md` updated (endpoint row, response note, smoke test, changelog).

---

## 3. Design

### 3.1 Scope of the aggregate
The dashboard summarizes the **same set** the topics list shows by default: the caller's
topics for the **current year**, **excluding archived** ones. This keeps the dashboard and the
topics list consistent (an archived topic shouldn't drag down "your year"). Year is the current
year; a `?year=YYYY` selector is a trivial future addition but is **not** in this step (the
landing view is "this year").

### 3.2 Aggregate progress (pure domain)
`domain.AggregateProgress(items []Progress) Progress` sums `Total` and `Completed` across
topics and recomputes the ratio via `ComputeProgress` (so the div-by-zero and clamping guards
are reused). Aggregating *goal counts* — not averaging per-topic ratios — is the correct
"overall completion": 1/10 and 9/10 aggregate to 10/20 = 50%, not the misleading mean of 55%.

### 3.3 Topic tallies
`topics.total` = number of topics in scope. `topics.completed` = topics whose progress ratio
is exactly `1.0` **and** that have at least one goal (`total > 0`) — an empty topic is not a
completed one.

### 3.4 Message
The positive message reuses the Step-6 voice:
- **No topics yet** (`topics.total == 0`): a distinct empty-state line (see §7) — "Set your
  first goal" is wrong here because the user has no *topic* to add goals to.
- **Otherwise**: `domain.Motivate(overallProgress)` — the same six warm bands, now reading as
  an overall status (0% → "the hardest part is starting", 100% → the 🎉 line). Reusing it keeps
  one source of truth for the copy and means the dashboard and topics speak the same language.

Encapsulated as `domain.DashboardMessage(overall Progress, topicCount int) string`.

### 3.5 Countdown (pure domain)
`domain.NextNewYear(now time.Time) time.Time` → Jan 1 `00:00:00` of `now.Year()+1` in `now`'s
location. `domain.Countdown(now time.Time)` → `(target time.Time, secondsRemaining int64)`,
with `secondsRemaining` floored at 0. The server passes `time.Now().UTC()`, so `target` is the
next Jan 1 in **UTC** and `secondsRemaining` counts down to it.

> **Timezone note (flagged, §7):** New Year is inherently local. The backend returns a
> canonical **UTC** target + seconds; the React client (which knows the browser timezone) can
> refine to the user's local Jan 1 later. No per-user timezone is stored in this step.

### 3.6 Service — composition, not new persistence
`DashboardService.Get(ctx, userID)` calls the existing `TopicService.List(ctx, userID, nil,
false)` (current year, non-archived) — which already returns each topic's **live** progress via
the goals aggregation — then folds the views into the aggregate. This reuses the exact
progress path (no drift, no N+1: `List` already batches counts in one aggregation) and adds
**zero** repository code.

The service depends on a small consumer-defined interface, not the concrete type:

```go
type DashboardTopics interface {
    List(ctx context.Context, userID string, year *int, includeArchived bool) ([]TopicView, error)
}
```

`*TopicService` satisfies it. A `clock func() time.Time` (defaulting to `time.Now`) is injected
so the countdown is deterministic in tests.

### 3.7 Transport
`GET /dashboard`, mounted in the existing authenticated group. `DashboardResponse` projects the
service view; the `progress` block reuses the same shape as `ProgressResponse`.

---

## 4. Files

```
internal/
  domain/
    dashboard.go                       # AggregateProgress, DashboardMessage, NextNewYear, Countdown
    dashboard_test.go                  # aggregate math, message states, countdown boundaries

  service/
    dashboard_service.go               # DashboardService + DashboardTopics port + DashboardView
    dashboard_service_test.go          # empty state, aggregate across topics, injected clock

  transport/http/
    dashboard_dto.go                   # DashboardResponse (+ countdown/topics sub-structs)
    dashboard_handler.go               # GET /dashboard
    dashboard_flow_integration_test.go # end-to-end against mongo:7
    router.go                          # CHANGED: mount GET /dashboard; RouterDeps.DashboardService

cmd/api/main.go                        # CHANGED: construct & wire DashboardService
docs/dev_run.md                        # CHANGED: endpoint row, note, smoke test, changelog
```

No new endpoints beyond `/dashboard`, no new collections, indexes, or dependencies.

---

## 5. API shape

`GET /api/v1/dashboard` → `200`:

```json
{
  "year": 2026,
  "message": "Real progress — you're building something. Keep at it.",
  "progress": { "total": 12, "completed": 5, "ratio": 0.4166666666666667 },
  "topics":   { "total": 3, "completed": 1 },
  "countdown": {
    "target": "2027-01-01T00:00:00Z",
    "secondsRemaining": 15466923
  }
}
```

Unauthenticated → `401`. A user with no topics gets `progress {0,0,0}`, `topics {0,0}`, and the
empty-state `message`.

---

## 6. Testing Strategy

| Layer | What | Runs in |
| --- | --- | --- |
| domain | `AggregateProgress` sums counts (incl. empty, single, many); `DashboardMessage` per state incl. no-topics; `Countdown` target is next Jan 1 and seconds > 0 (incl. a late-Dec "now") | `make test` |
| service | no topics → empty-state message + zero aggregate; several topics with mixed progress → correct totals, `topics.completed`, and message; countdown uses the injected clock | `make test` |
| end-to-end | register → `GET /dashboard` empty state → create topics + goals, complete some → dashboard reflects aggregate, tallies, message, and a `target` in the next year | `make test-int` |

## 7. Decisions — DECIDED

**(a) Empty-state copy — LOCKED.** In the "encouraging & warm" voice; the progressed states
reuse the Step-6 messages verbatim, so only the *no-topics* line is new:

| State | Message |
| --- | --- |
| No topics yet | A blank page is a fresh start. Create your first topic and shape your year. |
| Has topics | *(reuses the Step-6 bands via `Motivate`)* |

**(b) Countdown timezone — LOCKED: UTC.** Return the next Jan 1 in **UTC** plus
`secondsRemaining`; the React client localizes to the browser's timezone. No per-user timezone
is stored in the backend in this step.

Everything else here (current-year + non-archived scope, aggregate-by-counts, reuse of the
topic-list path, injected clock) is small technical discretion.

## 8. Verification Checklist

1. `make tidy && go build ./...` — clean.
2. `gofmt` / `go vet` (both tags) / `golangci-lint` (both tags) — clean.
3. `make test` — unit suite green, `-race`.
4. `make test-int` — dashboard reflected end-to-end against real `mongo:7`.
5. Live smoke: register, `GET /dashboard` (empty state); create topics + goals, complete some,
   `GET /dashboard` again and confirm the aggregate, tallies, message, and next-year countdown.
   Update `docs/dev_run.md` and re-verify.

## 9. What This Unblocks

Completes the core backend feature set. Step 8 (Hardening) then adds rate limiting, the full
OpenAPI spec, and end-to-end/error-path coverage across all endpoints — after which Phase B
(the React client) consumes `/dashboard` as the landing view.
