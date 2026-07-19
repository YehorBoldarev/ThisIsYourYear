# Step B4 — Topic & milestone flows (CRUD modals)

> **Status:** Draft for approval
> **Parent:** [`00-system-plan.md`](00-system-plan.md) §6 (step B4)
> **Builds on:** B1 (design system + Modal/Field/Toggle/Swatch/Ring primitives), B2
> (typed client, auth, TanStack Query), B3 (dashboard + chapters list) — **done & approved**.
> **Approval gate:** No code is written until you approve this document.

---

## 1. Objective

Make the dashboard **interactive**: turn the three stubbed actions from B3 (open a
chapter, "Begin a new chapter", empty-state CTA) into the **four designed popups**
and wire the full **create / read / update / delete** cycle for topics ("big goals")
and goals ("milestones") to the API — with **optimistic updates** so progress bars,
percentages, and the dashboard aggregate move the instant you act. Pixel-accurate in
**both themes** at **desktop / 768 / 390** (popups become near-full-width sheets on mobile).

> **Terminology (locked by the domain + contract).** In the UI a **big goal /
> chapter** is a `Topic` (`/api/v1/topics`); a **milestone** is a `Goal` under a topic
> (`/api/v1/topics/{id}/goals`, `/api/v1/goals/{id}`). This plan uses the UI words
> ("chapter", "milestone") for people and the schema words (`Topic`, `Goal`) for code.

### In scope

- **Topic-detail popup** (handoff screen 3) — opened from a chapter row/card:
  category eyebrow, title, description, motivational callout, progress **ring**,
  "N of M milestones complete" + band line, and the **milestone checklist** (toggle a
  circle to complete/reopen; ✓ normal-done, ★ achievement-done struck-through, open
  rows show the due date). Per-row edit + delete. Footer: **Edit chapter**, **＋ Add
  milestone**.
- **Create-a-big-goal popup** (screen 4) — title, "why does it matter?" (description),
  target date, category, **accent swatch** (teal/blue/gold/rose). Reached from the
  "Begin a new chapter" link **and** the empty-state CTA.
- **Add-a-milestone popup** (screen 5) — milestone title, target date (optional), notes
  (optional), and the **"Celebrate as an achievement ★"** toggle (`isAchievement`).
- **Edit popups** — reuse the create/add forms in an "edit" mode for a chapter
  (`PATCH /topics/{id}`) and a milestone (`PATCH /goals/{id}`), pre-filled.
- **Delete** — a small confirm step for deleting a chapter (cascades to milestones)
  and for deleting a milestone.
- **Mutations** (typed hooks, all with **optimistic update + rollback + settle-invalidate**):
  create/patch/delete topic; create/patch/delete goal; toggle milestone done↔todo;
  toggle `isAchievement`. Optimistic changes recompute the topic `progress` and the
  dashboard tally locally so the UI never waits on the round-trip.
- **A modal host** (feature-local state, no new routing) tracking which popup is open
  and its target; opened via callbacks passed down from the dashboard (replacing the
  B3 stubs).
- Small libs: `lib/date.ts` (DD-MM-YYYY ⇄ native `<input type="date">` `YYYY-MM-DD`,
  and a "Due Oct 4" display formatter), reused by both forms and the checklist.
- **Tests** (MSW + RTL): open topic detail, toggle a milestone (optimistic tick +
  progress bump), create a chapter (appears in the list), add a milestone, edit,
  delete-with-confirm, and a validation-error (422 → inline field messages). Unit
  tests for `lib/date`. Plus a **live** create→milestone→complete→delete run against
  the real Go API + Mongo through the Vite proxy.

### Explicitly NOT in this step

- **No celebration / confetti.** Reaching 100% updates the ring and the ★ ACHIEVED tag,
  but the celebration modal (screen 6) and the final four-band motivation **copy** are
  **B5**. B4 keeps using B3's `motivationBand` short labels for the detail callout.
- **No Achievements ("Wins") page** — B5.
- **No drag-to-reorder** milestones. The schema supports `order`, but reordering UX is
  out of scope here (create appends; we render in the API's `order`). Flag if you want it.
- **No URL-deep-linking** of popups (see §6 decision D3).

---

## 2. API surface used (all already in `api/openapi.yaml`)

| Action | Call | Notes |
| --- | --- | --- |
| Load a chapter's milestones | `GET /topics/{id}/goals` | On popup open; cached `['topics', id, 'goals']`. |
| Create chapter | `POST /topics` (`CreateTopicRequest`) | `title`* , `description`, `color`, `category`, `targetDate` (DD-MM-YYYY). |
| Edit chapter | `PATCH /topics/{id}` (`UpdateTopicRequest`) | Partial; `targetDate:""` clears. |
| Delete chapter | `DELETE /topics/{id}` | 204; cascades to milestones. |
| Add milestone | `POST /topics/{id}/goals` (`CreateGoalRequest`) | `title`*, `notes`, `dueDate`, `isAchievement`. |
| Edit milestone | `PATCH /goals/{id}` (`UpdateGoalRequest`) | `title`, `notes`, `dueDate` (`""` clears), `isAchievement`. |
| Complete / reopen | `PATCH /goals/{id}` `{done: true|false}` | `status` flips `todo`⇄`done`; `completedAt` set/cleared by server. |
| Delete milestone | `DELETE /goals/{id}` | 204. |

Server recomputes `Topic.progress` and `Topic.motivation`; we mirror `progress`
optimistically and reconcile from the server response on settle. Validation failures
return `422` with the `{error:{code,message,fields}}` envelope B2 already maps to
inline field errors.

---

## 3. Files

**New — data / logic**
- `src/features/topics/useTopicGoals.ts` — `useQuery(['topics', id, 'goals'])`.
- `src/features/topics/mutations.ts` — `useCreateTopic`, `useUpdateTopic`,
  `useDeleteTopic`, `useCreateGoal`, `useUpdateGoal`, `useDeleteGoal`,
  `useToggleGoalDone` (thin wrapper over `useUpdateGoal`). Each owns its
  `onMutate/onError/onSettled` optimistic logic against the `['topics', …]` and
  `['dashboard']` caches. A shared `applyProgress()` helper recomputes a topic's
  `{total, completed, ratio}` after a local milestone change.
- `src/lib/date.ts` — `toApiDate(iso)`, `toInputDate(apiDate)`, `formatDueShort(apiDate)`.

**New — UI**
- `src/features/topics/TopicDetailModal.tsx` — screen 3 (uses `Ring`, milestone list).
- `src/features/topics/MilestoneRow.tsx` — one checklist row (status token + label +
  tag/date + edit/delete affordances).
- `src/features/topics/TopicFormModal.tsx` — create/edit chapter (screen 4).
- `src/features/topics/MilestoneFormModal.tsx` — add/edit milestone (screen 5).
- `src/features/topics/ConfirmDelete.tsx` — small reusable confirm body (in a `Modal`).
- `src/features/topics/ModalHost.tsx` + `useTopicModals.ts` — feature-local state for
  which popup is open and its target; renders the active modal.

**Changed**
- `src/features/dashboard/Dashboard.tsx` — wrap in `ModalHost`; pass real
  `onOpenTopic` / `onCreate` callbacks (replacing B3 stubs) to `ChaptersSection` /
  `EmptyState`.
- `src/features/topics/ChaptersSection.tsx`, `ChapterRow.tsx`, `EmptyState.tsx` —
  accept and fire the real open/create callbacks.
- `src/components/index.ts` — export any primitive newly needed by the modals (all
  already exist: `Modal`, `Field`, `Input`, `Textarea`, `Toggle`, `Swatch`/
  `SwatchGroup`, `Ring`, `Button`, `Chip`, `Icon`).
- `src/test/msw.ts` — add `goalResponse()` builder + default handlers for the goal
  endpoints (`GET/POST /topics/:id/goals`, `PATCH/DELETE /goals/:id`,
  `POST /topics`, `PATCH/DELETE /topics/:id`).

**New — tests**
- `src/features/topics/TopicDetailModal.test.tsx`,
  `src/features/topics/TopicFormModal.test.tsx`,
  `src/features/topics/MilestoneFormModal.test.tsx`,
  `src/features/topics/mutations.test.tsx` (optimistic tick + rollback on error),
  `src/lib/date.test.ts`.

---

## 4. Behaviour details

- **Opening a chapter.** Clicking a row/card opens `TopicDetailModal` for that topic.
  The topic object comes from the already-cached `['topics']` list (instant paint);
  its milestones load via `useTopicGoals` (skeleton rows while pending). Progress ring
  + band line derive from the (optimistically maintained) `progress`.
- **Toggling a milestone.** Clicking the status circle fires `useToggleGoalDone`:
  optimistically flip `status`, recompute the topic's `completed`/`ratio`, and bump the
  dashboard tally; on error roll back and surface a toast/inline message; on settle
  invalidate `['topics', id, 'goals']`, `['topics']`, `['dashboard']`. A ★-flagged
  milestone shows the gold star treatment when done; a normal one shows the teal ✓.
- **Dates.** Forms use a token-styled native `<input type="date">` (accessible, no
  masking); we convert to **DD-MM-YYYY** for the API and render "Due Oct 4" in the
  checklist via `lib/date`. Empty = omitted on create, `""` to clear on edit.
- **Validation.** Client guards the one hard rule (title required, non-empty); anything
  else defers to the server `422` and maps `fields` onto the inputs (reusing B2's
  `Field` inline-error path). Buttons show a pending state and disable during flight.
- **Accessibility.** All four use the B1 `Modal` (backdrop + Esc + focus-trap + return
  focus + scroll-lock). Milestone toggles are real `<button>`s with
  `aria-pressed`/label; the checklist is a `<ul>`. Full a11y sweep is B6, but nothing
  here regresses it.
- **Responsive.** Popups center on desktop/tablet and become near-full-width bottom
  sheets on mobile (≤430), matching the handoff; the milestone list scrolls within the
  card if long.

---

## 5. Verification (Definition of Done)

- `npm run typecheck`, `lint`, `format:check`, `test` (new suites green + the existing
  39 still pass), `build`, and `gen:api` **no-diff** — all green.
- **Live run** against the real Go API + Mongo through the Vite proxy: register/login →
  create a chapter (accent + category + target date) → add two milestones (one flagged
  ★) → complete one (ring + dashboard move) → edit a milestone → reopen it → delete a
  milestone → delete the chapter (list returns to empty state). Verified in **both
  themes**; processes shut down cleanly afterward.
- `docs/dev_run.md` (status → B4, §5 frontend prose, changelog entry) and
  `web/README.md` ("What's here (B4)") updated.
- Nothing committed unless you ask.

---

## 6. Decisions I need from you (before I implement)

Small product ambiguities in the handoff that touch behaviour — I have a recommended
default for each; tell me if you'd rather go another way.

- **D1 — the footer "★ Mark goal achieved" button (topic-detail, screen 3).** The data
  model has **no** topic-level "achieved" flag; a chapter is 100% *derived* from all its
  milestones being done. So a literal one-click "mark achieved" doesn't map cleanly.
  **My recommendation:** in B4, replace that footer slot with **Edit chapter** +
  **＋ Add milestone**, and let 100% happen naturally when the last milestone is checked
  (the celebration for that moment is B5). *Alternative:* make "Mark achieved" a
  shortcut that completes every remaining milestone at once — say so if you want that.
- **D2 — deleting.** I'll gate both deletes behind a small **inline confirm** (no native
  `confirm()`), since chapter deletion cascades to milestones. OK?
- **D3 — popups & the URL.** Default: **feature-local state** (no route change; simplest,
  fully testable). *Alternative:* deep-link topic detail at `/topics/:id` so browser Back
  closes it. I lean local-state for B4 and can add URL-sync later; flag if you want it now.
- **D4 — reorder.** Out of scope here (create appends, render by `order`). Confirm that's
  fine, or I'll fold drag-reorder into B4.

Anything in §1/§2 that looks like a **system-wide** change (contract, data model, the
locked decisions) I'll stop and bring back to you rather than decide.
