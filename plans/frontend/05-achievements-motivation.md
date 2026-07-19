# Step B5 — Achievements page, celebration & motivation system

> **Status:** Draft for approval
> **Parent:** [`00-system-plan.md`](00-system-plan.md) §6 (step B5)
> **Builds on:** B1 (design system) · B2 (typed client/auth) · B3 (dashboard) · B4 (topic & milestone modals + optimistic mutations) — **done & approved**.
> **Approval gate:** No code is written until you approve this document.

---

## 1. Objective

Close the loop on the emotional payoff of the app: **completing goals feels like
something, and the wins accumulate somewhere.** Three connected pieces:

1. **A single four-band motivation system** (`lib/motivation`) carrying the design's
   canonical copy — band label, tone, Newsreader title, and italic message — used by
   **both** the dashboard chapter rows and the topic-detail callout, so there is one
   source of truth for "how is this goal going."
2. **The 100% celebration** (design screen 7): a gold-medallion modal with confetti,
   a congratulatory quote, a 3-up stat strip, and two actions — shown when a big goal
   reaches 100%.
3. **The Achievements ("Wins") page** — the real content behind the placeholder route,
   derived from the user's achieved milestones.

Pixel-consistent in **both themes** at **desktop / 768 / 390**, and honouring
`prefers-reduced-motion`.

### In scope
- **`lib/motivation` finalized** — extend `BandInfo` to `{ band, tone, shortLabel,
  title, message }` with the design's exact copy and thresholds (§4). Existing callers
  (B3 dashboard rows) keep working via `shortLabel`.
- **`Confetti` primitive** — a lightweight, dependency-free confetti burst (absolutely
  positioned coloured dots/squares in the four accents), **disabled under
  `prefers-reduced-motion`**. Added to `src/components` + the `/gallery`.
- **`CelebrationModal`** — design screen 7: 78px star-in-glow medallion, "ACHIEVEMENT
  UNLOCKED" mono eyebrow, goal title, italic "A promise kept" quote, a 3-up stat strip,
  and `Reflect on it` (outline) + `Keep the momentum →` (gold) actions.
- **Celebration trigger** — wired into the existing `ModalHost`: when toggling a
  milestone flips its topic to **100%**, the celebration opens (see **D1**).
- **Achievements page** — replaces the placeholder `routes/Achievements.tsx`. Fetches
  each current-year topic's goals (see **D3**), derives the achieved milestones, and
  renders them as a gold-accented list (empty / loading / error states included).
- **Topic-detail callout + dashboard rows** re-pointed at the finalized band util
  (see **D2**).
- **Tests** (Vitest + RTL + MSW): band copy/thresholds; Confetti reduced-motion;
  CelebrationModal render (stats + actions); achievements derivation + page states;
  the 100%-crossing trigger through `ModalHost`. Plus a **live** run against the real
  API + Mongo.

### Explicitly NOT in this step
- **No new backend endpoint.** There is no `/achievements` route (confirmed against
  `api/openapi.yaml` + the router); the Wins page is **derived client-side**.
- **No Reflections/Notes page** ("Reflect on it" links there but the page stays the
  B1 placeholder — it's a **B6** deliverable). No axe/Lighthouse a11y sweep (B6).
- No archive/reorder features.

---

## 2. Backend contract (verified — nothing new needed)

The data already exists; B5 only reads and derives:

- **`GET /api/v1/topics`** → `Topic[]`, each with `progress {total, completed, ratio}`,
  `motivation` (backend string), `color`, `createdAt`. A topic is "achieved" when
  `progress.total > 0 && progress.ratio === 1`.
- **`GET /api/v1/topics/{id}/goals`** → `Goal[]`; a milestone is a **win** when
  `status === 'done' && isAchievement === true`; `completedAt` (present when done)
  gives the date; `title` + parent topic give the label.
- **No bulk-goals endpoint exists.** The Wins page therefore fans out one goals request
  per current-year topic (§D3), reusing the `['goals', topicId]` cache B4 already
  populates, so opening a chapter first makes Wins instant.

No `gen:api` change (no schema edits).

---

## 3. Files (created ·/ changed)

```
web/src/
  lib/
    motivation.ts          ~ extend BandInfo → {band,tone,shortLabel,title,message}; design copy + thresholds
    motivation.test.ts     ~ add band-copy + threshold-boundary cases
    achievements.ts        + deriveAchievements(topics, goalsByTopic) + celebrationStats(topic, allTopics)
    achievements.test.ts   + derivation ordering/filter + stat math (months, ordinal)
  components/
    Confetti.tsx           + reduced-motion-aware confetti burst
    index.ts               ~ export Confetti
  features/
    celebration/
      CelebrationModal.tsx  + screen-7 modal (medallion, eyebrow, title, quote, 3-up stats, 2 actions)
      CelebrationModal.test.tsx  + renders title/quote/stats/actions; onReflect / onKeep fire
    achievements/
      useAchievements.ts    + useQueries fan-out over topics → { items, isLoading, isError }
      AchievementsPage.tsx  + the Wins page (header + list + empty/loading/error)
      AchievementCard.tsx    + one win row (gold ★, title, topic eyebrow, completed date)
      AchievementsPage.test.tsx  + list render, empty state, achieved-milestone filtering
  features/topics/
    TopicDetailModal.tsx   ~ callout uses band title+message+tone (D2); band line from full util
    ModalHost.tsx          ~ add 'celebrate' state; open it when a toggle crosses a topic to 100% (D1)
    MilestoneFormModal.tsx ~ (D1) align the ★-toggle helper copy with the chosen trigger
  features/dashboard/
    ChapterRow.tsx         ~ status word + tone from the finalized band util (design copy)
  routes/
    Achievements.tsx       ~ replace placeholder with <AchievementsPage/>
  test/
    msw.ts                 ~ multi-topic goals fixtures for the Wins fan-out (builders already exist)
```

Route table, providers, and auth are untouched — the Wins route and bottom-tab "Wins ★"
entry already exist from B1.

---

## 4. The motivation bands (design copy — the canonical table)

From design screens 3 & 7. `motivationBand(ratio, total)` returns:

| Band | Threshold | tone token | `shortLabel` (mono) | `title` (Newsreader) | `message` (italic) |
| --- | --- | --- | --- | --- | --- |
| `not-started` | `total===0 \|\| ratio<=0` | text-muted (grey) | Just started | The blank page | "A blank page isn't empty — it's full of everything that could be. Make the first mark today." |
| `in-progress` | `0 < ratio < 0.8` | blue | In progress | Building momentum | "Momentum is quiet. Keep showing up, and the small days compound into a remarkable year." |
| `almost-there` | `0.8 ≤ ratio < 1` | teal | Almost there | In sight | "You can see it from here. Don't trade the finish line for comfort — finish what you started." |
| `achieved` | `ratio ≥ 1` (and `total>0`) | gold | Achieved | A promise kept | "You kept a promise to yourself. This is exactly what that feels like. Take a moment to savour it." |

- Thresholds match the mock (`85%` bar sits in "almost there"; `0.8` is the boundary).
- `shortLabel` values are unchanged from B3 except **"Not started" → "Just started"**
  to match the design; the dashboard "N milestones to go" band line is preserved.
- `tone` maps to existing semantic/accent tokens — no new colours.

---

## 5. Celebration (design screen 7)

- **Trigger:** see **D1**. Fired from `ModalHost`, which already owns modal state and
  the `useToggleGoalDone` result; it detects a topic crossing to `ratio===1` and swaps
  to the `celebrate` state for that topic.
- **Visual:** centered ~480px card on a gold radial backdrop; `Confetti` overlay; 78px
  star-in-glow medallion (reuses the design's star SVG); mono gold "ACHIEVEMENT
  UNLOCKED"; topic title (Newsreader 30); the "A promise kept" italic quote; a 3-up
  stat strip over a hairline; two buttons.
- **Stat strip (D4):** `MILESTONES` = `completed/total`; `TO COMPLETE` = whole months
  between the topic's `createdAt` and now (`"5 mo"`, `"<1 mo"`); `#N OF {year}` =
  ordinal of this topic among the user's 100%-complete topics this year.
- **Actions:** `Reflect on it` → navigates to `/reflections` (B1 placeholder for now)
  and closes; `Keep the momentum →` → returns to the topic detail (or dashboard).
- **Motion:** confetti + medallion glow animate normally; under
  `prefers-reduced-motion` the confetti renders static (or not at all) and the modal
  just fades — no transform animation.

---

## 6. Achievements ("Wins") page

- **Data (`useAchievements`):** `useTopics()` for the current year, then `useQueries`
  to load `['goals', topicId]` for each — reusing B4's cache. `deriveAchievements`
  flattens to milestones where `status==='done' && isAchievement`, attaches the parent
  topic (title + `color`), and sorts by `completedAt` **desc** (undated last).
- **View:** editorial 680px column — mono "Wins" eyebrow, "Your achievements" H1, an
  italic sub-line ("N moments worth keeping" / design tone) — then a list of
  `AchievementCard`s (gold ★ medallion dot, milestone title, topic eyebrow in the
  topic's accent, completed date via `lib/date`). Mobile stacks to full-width cards.
- **States:** loading skeleton; error (inline + retry); **empty** — an on-brand "No
  wins yet" panel ("Star a milestone as an achievement, and finishing it lands here.")
  consistent with the dashboard empty state. Empty triggers when zero derived wins.

---

## 7. Small technical decisions (mine; for transparency)

| Decision | Choice | Why |
| --- | --- | --- |
| Confetti impl | Hand-rolled absolutely-positioned spans (no library) | Matches design's static dots; zero deps; trivially reduced-motion-aware. |
| Wins fan-out | `useQueries` over the topics list, cache-keyed `['goals', id]` | No bulk endpoint; reuses B4 cache so it's often warm; parallel + individually cached. |
| Band tone | Map to existing tokens (grey/blue/teal/gold) | No new colours; consistent light/dark. |
| Celebration ownership | Lives in `ModalHost` beside the other topic modals | It already holds the toggle mutation + live topic; one modal at a time. |

None of these touch a locked decision. The genuinely product-level calls are raised as
**D1–D4** below with recommendations — I proceed on the defaults unless you say otherwise.

---

## 8. Decisions for you (defaults recommended; flag any before I build)

**D1 — What triggers the celebration?** *(the one real ambiguity — the mock's screen 7
is topic-level "6/6 milestones · #1 of 2026", while the B4 milestone toggle helper says
"unlock a celebration when this is done".)*
- **(Recommended) Topic reaches 100%.** The celebration fires when the milestone you
  complete makes its big goal 100% done — exactly the screen-7 mock (topic title +
  topic stats). The per-milestone **★ `isAchievement`** flag then means "this is a Wins
  milestone" (gold star in the checklist + it appears on the Wins page). I'd soften the
  B4 toggle helper to "Show a gold star and add it to your Wins" to match.
- **(Alt) Per-starred-milestone.** Every `isAchievement` milestone celebrates on
  completion (screen 7 repurposed with the milestone's title). Simpler mental model for
  the toggle, but the stat strip ("#1 of 2026", "N/M milestones") is inherently
  topic-level and would need rework.

**D2 — Motivation source of truth in the topic-detail callout.**
- **(Recommended) Frontend band copy.** The callout shows the band `title` + `message`
  from `lib/motivation` (design copy), so it stays consistent with the dashboard, works
  under optimistic updates, and honours the system plan's "one motivation source of
  truth." The backend `topic.motivation` string becomes unused in the UI (kept in the
  type).
- **(Alt) Keep the backend string** in the callout; use the frontend util only for the
  band label/tone. Two voices, but zero client copy to maintain.

**D3 — What the Wins page lists.**
- **(Recommended) Achieved milestones** — completed `isAchievement` goals across all
  current-year topics (matches "derived from achieved milestones").
- **(Alt) Completed big goals** — only 100%-complete topics. Fewer, coarser entries.

**D4 — Celebration stat strip.**
- **(Recommended) Compute all three** — `N/M milestones`, `~mo to complete`
  (createdAt→now), `#N of {year}` (ordinal among completed topics), with graceful
  fallbacks ("<1 mo").
- **(Alt) Milestones only** — drop the two softer stats to avoid any "5 mo" feeling
  arbitrary.

---

## 9. Layout & responsive

- **Celebration:** ~480px card, centered; on ≤430 it becomes a near-full-width sheet
  (consistent with the other modals), confetti clipped to the backdrop.
- **Wins page:** 680px editorial column desktop; tightened at 768; stacked full-width
  cards at 390 (bottom tab bar already active, "Wins ★" highlighted).
- Both themes verified; all interactive elements keyboard-reachable (full a11y sweep is
  B6, but the celebration modal reuses `Modal`'s focus-trap/return-focus, and actions
  are real buttons/links).

## 10. Testing

- **Libs:** `motivation` — each band's copy + boundary thresholds (0, 0.79, 0.8, 0.99,
  1.0, `total===0`); `achievements` — filter (only done+starred), sort by `completedAt`
  desc, and stat math (months rounding, ordinal among completed topics).
- **Confetti:** renders pieces normally; renders nothing / static under a mocked
  `prefers-reduced-motion`.
- **CelebrationModal:** renders the topic title, quote, the three stats, and both
  actions; `Reflect on it` / `Keep the momentum →` invoke their handlers.
- **Trigger (ModalHost):** completing the last open milestone of a topic (MSW goals
  array) opens the celebration; completing a non-final milestone does not.
- **Wins page:** derives + renders cards from multi-topic goals fixtures; shows the
  empty state with none; loading/error states.
- **Live** (real API + Mongo): create a topic with 2 milestones (one starred); complete
  the first → no celebration; complete the last → **celebration fires** with correct
  stats; the Wins page lists the starred completed milestone; reopen one → Wins updates;
  check both themes at the three breakpoints.

## 11. Verification (definition of done)

`typecheck` · `lint` · `format:check` · `test` (all suites green) · `build` ·
`gen:api` a no-op diff. The live flow in §10 passes in both themes. `web/README.md`
and `docs/dev_run.md` (frontend section + changelog) updated. **Nothing committed.**

---

On approval I implement B5, verify per §11 (including the live run in both themes),
update the docs, and report back for your **"Task completed"** before B6.
```
