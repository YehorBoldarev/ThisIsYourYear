# Step B3 — Dashboard (Editorial Journal)

> **Status:** Draft for approval
> **Parent:** [`00-system-plan.md`](00-system-plan.md) §6 (step B3)
> **Builds on:** B1 (design system) + B2 (typed client, auth) — **done & approved**.
> **Approval gate:** No code is written until you approve this document.

---

## 1. Objective

Build the **home screen** — the Editorial-Journal dashboard from the handoff —
wired to real data from `GET /api/v1/dashboard` and `GET /api/v1/topics`. When the
user has topics it shows the day's greeting, the year-progress bar + New-Year
countdown, the italic sub-message, and "The chapters" list; when they have none it
shows the empty state. Pixel-accurate in **both themes** at **desktop / 768 / 390**.

### In scope
- **Data hooks** (first TanStack Query hooks): `useDashboard()` → `GET /dashboard`,
  `useTopics()` → `GET /topics` (default: current year, non-archived).
- **Dashboard view** replacing the B1 placeholder: date eyebrow, greeting
  (name + time-of-day), italic sub-message (dashboard `message`), **year progress
  bar** (elapsed % + "YOU ARE HERE" + days-left from the countdown), section row
  "The chapters", and the **chapter list** (accent bar, title, ★ ACHIEVED at 100%,
  status · done/total, percent, mini bar).
- **Empty state** (screen 8): faint logo, "Your year is a blank page.", CTA, TRY chips.
- **Loading** (skeleton) and **error** states.
- **Responsive**: 680px editorial column desktop; tightened tablet; **mobile** stacks
  each chapter into a card with a full-width bar (bottom tab bar already present).
- Small shared libs: `motivation` (ratio → short band label), `greeting`
  (time-of-day), reused by later steps.
- **Tests** (MSW): data render, 100% ★ ACHIEVED, empty state, loading, error; unit
  tests for the libs. Plus a **live** check against the real API + Mongo.

### Explicitly NOT in this step
- **No modals / mutations.** Opening a chapter (topic detail) and creating/editing
  (create-goal, add-milestone) are **B4**. In B3 the "＋ Begin a new chapter" link,
  the empty-state CTA, and chapter rows are **rendered but their open/create actions
  are stubbed** (no-ops wired in B4). The full motivation copy + celebration is B5.

---

## 2. Backend contract (verified against the code)

**`GET /api/v1/dashboard`** → `DashboardResponse`:
```
{ year:int,
  message:string,                        // positive/aggregate sub-message (empty-state copy when no topics)
  progress:{ total:int, completed:int, ratio:float },   // aggregate milestones across current, non-archived topics
  topics:{ total:int, completed:int },   // topic tally
  countdown:{ target:string(ISO-UTC), secondsRemaining:int64 } }
```

**`GET /api/v1/topics`** (query: `year?` default current, `includeArchived?` default
false) → `Topic[]`:
```
{ id, title, description, year, color(hex), icon, category,
  targetDate:string(DD-MM-YYYY, "" if none), archived,
  progress:{ total, completed, ratio }, motivation:string, createdAt, updatedAt }
```

Notes that shape the UI:
- **Accent** for a chapter's bar = the topic's own `color` (a user hex), applied via
  inline style — no 4-color mapping needed.
- **Percent / done-total** for a row come from `progress` (`ratio`, `completed`,
  `total`). **★ ACHIEVED** shows when `ratio === 1` (and `total > 0`).
- **Year-progress bar** is **time-based**, not milestone-based: fill = fraction of
  the calendar year elapsed; "days left" = `ceil(countdown.secondsRemaining / 86400)`
  (backend-authoritative, timezone-correct). *Decision: bar fill uses the local
  `yearElapsedFraction()` (B1, already unit-tested) for a smooth %, and the day count
  uses the API countdown.* The dashboard `message` is the italic sub-message.
- **Empty state** triggers on `topics.total === 0` (equivalently an empty topics list).

---

## 3. Small technical decisions (mine; for transparency)

| Decision | Choice | Why |
| --- | --- | --- |
| Data fetching | **TanStack Query** hooks in `src/features/*` | The provider is already installed (B2); typed via `client`. |
| Query keys | `['dashboard']`, `['topics', { year, includeArchived }]` | Stable keys so B4 mutations can invalidate precisely. |
| Chapter "status" label | Derived on the client from `progress.ratio` | The API sends a long `motivation` message, not a short band label; the row needs the short word. `lib/motivation.ts` maps ratio → band + short label (0 = "Not started", <0.8 = "In progress", <1 = "Almost there", =1 = "Achieved"). **B5 finalizes the full band copy/thresholds**; B3 only needs the short label. |
| Greeting | `lib/greeting.ts` from local hour (morning/afternoon/evening) | Matches the mock ("Good evening,"). |
| Loading UI | Lightweight skeleton blocks (token-colored) | On-brand; no spinner jank for a content page. |

None of these touch a locked/system-wide decision. The band thresholds are a
**placeholder to be confirmed in B5** (the system plan already flags motivation copy
as a B5/B6 open question).

---

## 4. Files (created ·/ changed)

```
web/src/
  features/
    dashboard/
      useDashboard.ts        + GET /dashboard query hook (+ typed DashboardResponse)
      Dashboard.tsx          + the composed view (data → sections; loading/error/empty)
      DateGreeting.tsx        + date eyebrow + greeting + italic sub-message
      YearProgress.tsx        + year bar with JAN / "YOU ARE HERE · N% · D days left" / DEC
      DashboardSkeleton.tsx   + loading placeholder
    topics/
      useTopics.ts           + GET /topics query hook
      ChaptersSection.tsx     + "The chapters" header (+ "Begin a new chapter" link, stub)
      ChapterRow.tsx          + one topic row/card (accent bar, title, ★ ACHIEVED, status·d/t, %, mini bar)
      EmptyState.tsx          + "Your year is a blank page." + CTA (stub) + TRY chips
  lib/
    motivation.ts            + ratio → { band, shortLabel }  (extended in B5)
    greeting.ts              + time-of-day greeting
  routes/
    Dashboard.tsx            ~ replace B1 placeholder with <Dashboard/> from features
  test/
    msw.ts                   ~ add dashboard/topics default handlers + builders
```

Wiring only — no route/table/provider changes. The dashboard renders inside the
existing `AppShell` (header/footer/tab bar from B1).

## 5. Layout & responsive

- **Desktop**: centered **max-width 680px** editorial column — date eyebrow (mono,
  centered) → hairline rule → greeting H1 → italic sub-message → year bar with
  captions → "The chapters" row (title + "＋ Begin a new chapter") → chapter rows
  (flex: 3px accent bar | title + ★ tag + mono status·done/total | right: percent +
  96px mini bar).
- **Tablet 768**: same, tightened per handoff (H1 30px, padding 30×36).
- **Mobile 390**: greeting 25px; each chapter becomes a **stacked card** (left 3px
  accent border, title + ★ + percent on one line, mono status line, full-width 6px
  bar). Bottom tab bar already handles nav.
- **Empty state**: centered column, faint `Logo` (near-empty ring), H1, paragraph,
  primary CTA, TRY chips row — all present; CTA is a stub until B4.

## 6. States

- **Loading**: `DashboardSkeleton` (eyebrow/greeting/bar/rows as token blocks).
- **Error**: an on-brand inline message with a Retry button (refetch), reusing the
  error envelope's `message` where available.
- **Empty**: `topics.total === 0` → `EmptyState`.
- **Populated**: greeting + year bar + chapters.

## 7. Testing

- **Libs**: `motivation` band boundaries (0 / mid / 0.8 / 1.0); `greeting` by hour.
- **Dashboard** (MSW + Query): renders greeting with the user's name, the year-bar
  caption, and a chapter row with its percent; a `ratio:1` topic shows **★ ACHIEVED**;
  `topics.total:0` renders the empty state; a failing query renders the error state;
  the skeleton shows while pending. (Query wrapped with a fresh `QueryClient` +
  `AuthProvider` authenticated via the B2 MSW refresh handler.)
- **Live** (real API + Mongo): log in; with no topics see the empty state; create a
  topic + a couple of milestones **via the API** (curl), reload, and confirm the
  greeting, year %/countdown, and the chapter row (title, done/total, percent, and
  ★ ACHIEVED once all milestones are done) render correctly in **both themes** at the
  three breakpoints.

## 8. Verification (definition of done)

`typecheck` · `lint` · `format:check` · `test` (new suites green) · `build` ·
`gen:api` still a no-op diff. The live flow in §7 passes in both themes. `web/README.md`
and `docs/dev_run.md` (frontend section) updated. Nothing committed.

## 9. Open questions (non-blocking; defaults stated, proceed unless you object)

- **Band thresholds/labels** for the row status are a **B3 placeholder** (0 / <0.8 /
  <1 / =1); the canonical copy is decided in **B5**. Flag if you want specific words now.
- **Chapter accent** uses each topic's stored `color` hex. If a topic has no color,
  I'll fall back to the teal token. (Create-goal in B4 offers the 4-swatch picker.)
- **Row/CTA interactions** (open topic, begin chapter, empty CTA) are **stubbed in B3
  and wired in B4** — per the roadmap split. Say if you'd rather fold the create modal
  into B3 instead.

On approval I implement B3, verify per §8 (including the live run in both themes),
update docs, and report back for your **"Task completed"** before B4.
