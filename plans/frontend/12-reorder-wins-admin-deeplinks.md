# Step D3 — Milestone reordering, Wins endpoint, delete-account command, popups that survive a reload

> **Status:** Approved (2026-11-15; D1 **drag-and-drop only**, D2–D5 as recommended) and
> implemented (2026-11-15). See §7.
> **Parent:** [`backlog.md`](../backlog.md): four postponed items, picked together on 2026-11-15.
> **Approval gate:** No code is written until you approve this document. The four parts are
> independent; you can approve them separately.
> **Note:** This step touches Go, the SPA and the OpenAPI contract, so it runs both quality gates.
> **Not in this step:** the mobile footer fix (a small issue, fixed directly on 2026-11-15).

---

## 1. Objective

| Part | Today | After |
|---|---|---|
| **A · Reorder milestones** | Milestones show in creation order; the API can change one milestone's position at a time, and nothing in the app uses it | Reorder milestones inside a goal's popup, on phone and desktop |
| **B · Wins endpoint** | The Wins page sends one request per big goal (N goals → N + 1 requests) | One request returns all wins for the year |
| **C · Delete an account** | A hand-typed `mongosh` script (handbook §7.5) | `api admin delete-user --email X`, with a confirmation |
| **D · Popup survives a reload** | Reloading the page (or pressing Back) loses the open goal popup | The open goal is in the address (`?topic=<id>`): reload reopens it, Back closes it |

---

## 2. Current state (audited 2026-11-15)

- **Ordering.** Every milestone has an `order` field.
  - `GET /topics/{id}/goals` sorts by it, then by creation time.
  - `PATCH /goals/{id}` accepts `order`, but only for one milestone at a time. Moving one
    milestone means renumbering several, which would be several separate requests.
  - **Latent bug:** a new milestone gets "number of milestones" as its position. After a
    deletion that can **duplicate** an existing position (positions 1,2 → add → 2 again). It's
    harmless today, but reordering would expose it.
- **Wins.** `useAchievements` runs one `GET /topics/{id}/goals` per big goal, then filters
  `status === 'done' && isAchievement` in the browser (`lib/achievements.ts`). The Wins page and
  the celebration statistics use that derivation.
- **Account removal.** Repositories can delete one goal, one topic, or a topic's goals. Nothing
  deletes *all* of a user's data. The handbook's `mongosh` script does that by hand.
- **Popups.** `ModalHost` keeps the open popup in React state only (`{kind: 'detail', topicId}`
  and others), so the URL never changes. Reload and Back both lose it.

---

## 3. Design

### Part A — Reorder milestones

**API.** New endpoint, which sets a topic's complete milestone order in one request:

```
PUT /api/v1/topics/{id}/goals/order      { "goalIds": ["g3", "g1", "g2"] }   → 200 [ordered goals]
```

- **Validation:** the list must contain **exactly** that topic's milestones, each once.
  Otherwise the response is a 422 with `fields.goalIds`. That guards against a stale screen
  reordering a list that has changed meanwhile.
- **Storage:** the repository writes positions `0…n-1` in one bulk write, scoped by user *and*
  topic, like every other goal query.
- **Fix:** a new milestone goes **after the last one** (highest position + 1) instead of
  "count", which removes the duplicate-position bug.
- **Compatibility:** `PATCH /goals/{id}` with `order` keeps working as before.

**UI (D1).** An **"Arrange"** toggle in the goal popup's milestone header. In arrange mode each
milestone shows **↑ / ↓** buttons ("Move *Register for the race* up"), and Done ends it.

- **Every move is optimistic:** the list reorders instantly, the request runs in the background,
  and a failed request puts the list back (the same pattern as the other edits).
- **The ends are disabled:** ↑ on the first milestone and ↓ on the last are greyed out.
- **Why buttons rather than drag-and-drop:** buttons work the same with a thumb, a mouse, a
  keyboard and a screen reader. Dragging inside a scrolling popup on a phone is fiddly, and it
  needs a library (`@dnd-kit`, ~15 KB) to be accessible. The API supports either, so
  drag-and-drop can be added later without server changes.

### Part B — Wins endpoint

```
GET /api/v1/achievements?year=YYYY   → 200 [{ goal: Goal, topic: TopicSummary }]
```

- **What it returns:**
  - **Scope:** the caller's finished, starred milestones (`status: done`, `isAchievement: true`)
    across their big goals of that year. `year` defaults to the **season year**, the same rule
    as everywhere else (plan D1).
  - **Order:** newest first (`completedAt` descending).
  - **Shape:** `TopicSummary` is the subset the Wins cards use (id, title, colour, icon,
    category, year).
- **Server:** a new `AchievementService`. It reuses the topic list for the year, plus **one**
  goal query (`topicId in […]`, done + starred), and is user-scoped like everything else.
- **SPA:** `useAchievements` becomes one query (`['achievements', year]`).
  - Goal mutations (complete, reopen, star, edit, delete) also invalidate it, so Wins stays
    current.
  - The celebration statistics keep their current data source, so their behaviour is unchanged.
- **Indexes:** none added; the query rides the existing `userId + topicId` index. At family
  scale (tens of goals) that's ample.

### Part C — `api admin delete-user`

```
sudo ./backup.sh
sudo docker compose exec api /app/api admin delete-user --email someone@example.com
```

1. **Look up and summarise:** it finds the account and prints what will be deleted, e.g.
   *"someone@example.com (Someone): 3 big goals, 17 milestones, 2 sessions"*.
2. **Confirm (D4):** it asks you to **type the email** to confirm. `--yes` skips the question,
   for scripts.
3. **Delete, in this order:** milestones, big goals, sessions, then the account. If it's
   interrupted, a rerun finishes the job, since nothing ever points at a missing account.
4. **Report:** it prints the counts.

**What it needs underneath:**
- **Repositories:** `DeleteAllForUser` on goals, topics and sessions, and `Delete` on users. All
  are scoped by user id.
- **Service:** `AdminService.DeleteUser(email)` and a `Summary(email)` for step 1.
- **Docs:** handbook §7.5 is replaced by this command (backup first), and the backlog item is
  closed.

### Part D — Popup survives a reload

- **What the address stores:** opening a goal's popup adds `?topic=<id>` to the current page's
  address (`/?topic=…`, `/achievements?topic=…`).
  - **Opening (D5):** it's a new history entry, so **Back closes the popup**. That's the natural
    phone behaviour (Android's back gesture).
  - **Closing:** it removes the parameter.
- **What a reload does:**
  - `ModalHost` reads `?topic=` and reopens that goal's popup once the goal has loaded.
  - That works even for a goal outside the current year's list: it fetches `GET /topics/{id}`.
  - An unknown id, or someone else's (shared link), simply opens nothing, and the parameter is
    cleaned up.
- **Only the goal popup goes in the address.** Forms and confirmations opened from it (edit
  goal, add milestone, delete) are short-lived steps. After a reload you're back on the goal popup
  they came from, with nothing half-typed to lose.
- **Works alongside the other parameters:** `?invite=` (sign-up) and the planned `?year=`
  (Part B of the preparation-stage plan) are separate keys.

---

## 4. Decisions

| # | Decision | Status |
|---|---|---|
| D1 | ~~Arrange mode with ↑/↓ buttons~~ → **Drag-and-drop only** (`@dnd-kit`, with its keyboard and screen-reader support) | **Decided by you** |
| D2 | **One `PUT …/goals/order` with the full order** *(alt: one `PATCH` per moved milestone — no new endpoint, but several requests per move and no stale-list protection)* | Recommended |
| D3 | **`GET /achievements?year=` returning goals with a topic summary, newest first** | Recommended |
| D4 | **`delete-user` asks you to type the email**; `--yes` skips it | Recommended |
| D5 | **`?topic=<id>` with a history entry**: reload reopens, Back closes *(alt: replace the entry — Back then leaves the page instead of closing the popup)* | Recommended |

---

## 5. Files (indicative)

```
A  internal/repository/goals.go (+int test)        ~ SetOrder (bulk), max-order for create
   internal/service/goal_service.go (+test)       ~ Reorder (exact-set check), create order fix
   internal/transport/http/goal_handler.go, router.go, goal_dto.go ~ PUT …/goals/order
   web: useReorderGoals mutation · TopicDetailModal/MilestoneRow arrange mode (+tests, axe)
B  internal/repository/goals.go                    ~ ListAchievements(userID, topicIDs)
   internal/service/achievement_service.go (+test) + year-scoped wins
   internal/transport/http/achievement_handler.go, router.go ~ GET /achievements
   web: useAchievements → one query; invalidation in mutations.ts (+tests)
C  internal/repository/{goals,topics,sessions,users}.go (+int tests) ~ delete-all-for-user
   internal/service/admin_service.go (+test)       ~ Summary, DeleteUser
   cmd/api/commands.go (+test)                     ~ admin delete-user [--yes]
   docs/operations.md §7.5                         ~ the command replaces the mongosh script
D  web: ModalHost ↔ ?topic= (push on open, remove on close, reopen on load); useTopic(id)
   (+tests: reload reopens, Back closes, unknown id is ignored)
*  api/openapi.yaml · schema.d.ts · docs/dev_run.md · CLAUDE.md · plans/backlog.md
```

## 6. Verification

- **Go:**
  - Unit tests for each service: reorder with an exact / missing / extra / duplicate id list;
    wins year scope and order; delete counts and the order of deletion.
  - Integration tests over HTTP for reorder and wins, and repository integration tests for the
    bulk order write and delete-all.
  - CLI tests: usage errors, the `--yes` path, the typed-confirmation mismatch.
  - Full `make fmt vet lint test test-int`.
- **Web:**
  - arrange-mode moves (optimistic, plus rollback on a failed request), the ends disabled, axe;
  - Wins from one request, refreshed after starring a milestone;
  - reload reopens the popup, Back closes it, an unknown id is ignored;
  - the full gate.
- **Docs updated. Nothing committed.**

---

## 7. Implementation notes (2026-11-15)

**Decisions as approved:** D1 is **drag-and-drop only**; D2–D5 as recommended.

**Part A, drag-and-drop:**
- **Library:** `@dnd-kit/core` 6.3.1, `/sortable` 10.0.0, `/utilities` 3.2.2, adding about
  16 KB gzipped.
- **Handles:** a grip handle per milestone (`touch-none`, so dragging doesn't scroll the popup).
  They're hidden when there's only one milestone, and disabled while a new milestone is still
  saving (its id is temporary).
- **Keyboard and screen readers:** the keyboard sensor keeps it accessible without the ↑/↓
  buttons.
  - **Announcements** use title and position. "Picked up …" is no longer overwritten straight away
    by dnd-kit's "over itself" event.
  - **Rendering:** they're rendered *inside* the popup, because it's `aria-modal`.
- **Escape:** the popup closed on Escape before dnd-kit could cancel a keyboard drag. Content
  mid-interaction now marks itself `data-captures-escape`, and `Modal` leaves Escape to it.

**Part B, Wins endpoint:**
- `AchievementService` uses one topics query and one goals query. `deriveAchievements` was
  removed, since the server does that work now.

**Part C, delete-account command:**
- **Wiring:** `AdminService` now takes an `AdminDeps` struct, and `runCommand` gains stdin for the
  typed confirmation.
- **Timeouts:** each database step gets its own timeout, so time spent typing the confirmation
  doesn't eat into the delete.

**Part D, popup in the address:**
- **How it follows the address:** `ModalHost` adjusts the popup *during render* when `?topic=`
  changes (React's pattern for state derived from props); the only effect is the
  quiet-cleanup navigation for an unknown link.
- **History:** the history entry is flagged `{ topicPopup: true }`, so closing pops it; without
  the flag, the parameter is replaced.
- **Goals outside the list** go through `useTopic` (key `['topic', id]`, kept apart from the
  optimistic list keys).
- **Login** now returns to the full address, including `?topic=`.

**Found and fixed along the way:** the goal popup's progress ring had no accessible name. The
first accessibility test of the goal popup caught it (`Ring` takes a `label` now).

