# ThisIsYourYear — Frontend (Phase B) System Plan

> **Status:** Draft for approval
> **Parent:** [`../00-system-plan.md`](../00-system-plan.md) §8 Phase B (expands steps 10–11 into a
> detailed, ordered roadmap). Backend Step 9 ([`../09-design-data-extensions.md`](../09-design-data-extensions.md))
> is a prerequisite and lands first.
> **Scope of this document:** the tech stack, architecture, design-system approach, and the
> **ordered, step-by-step roadmap** for building the React SPA from the `design_handoff_this_is_your_year/`
> designs. Each numbered step below gets its own detailed `plans/frontend/NN-*.md` plan that must be
> **approved before any code is written for it** — exactly as we did for the backend.

---

## 1. What we're building

A single-page app for the "This is Your Year" goal tracker, recreating the **Editorial-Journal**
design direction (the client's chosen layout) pixel-accurately in React. Core screens, from the
handoff:

- **Dashboard (Editorial Journal)** — date/greeting, a year-progress bar + New-Year countdown, an
  italic motivational sub-message, and "The chapters": a list of the user's big goals with per-goal
  progress.
- **Topic detail** (modal) — one big goal, its motivational message (adapts to progress), and its
  milestone checklist with ✓ done / ★ achievement / open states.
- **Create big goal**, **Add milestone** (modals).
- **Achievement celebration** (100% state) + the **motivational-message** system (four bands).
- **Empty state**, **Achievements** ("Wins") page, and **two themes** (warm dark + warm light,
  toggle persisted), responsive across **desktop / tablet (768) / mobile (390)**.

The design is **high-fidelity**: colors, type, spacing, radii, and shadows are final and captured in
`design_handoff_this_is_your_year/README.md` (the source of truth for token values). Sample copy and
data are placeholders — everything wires to the real API.

---

## 2. Confirmed decisions (locked by you)

| Area | Decision |
| --- | --- |
| Framework | **React + TailwindCSS** (from the master plan §2) |
| Language / build | **TypeScript + Vite** |
| API contract | The typed client is derived from the checked-in **`api/openapi.yaml`** (OpenAPI 3.1) |
| Design-gap strategy | **Extend the backend first, then build.** The missing API fields are added as a **backend step** ([`../09-design-data-extensions.md`](../09-design-data-extensions.md)) that lands **before** Phase B; the frontend then builds against them. |
| Reflections / Notes | **Deferred.** The nav destination is shown as a "coming soon" placeholder in v1; no backend Reflections feature is built now. |
| Achievements page | **Derived from existing data** (achieved milestones/goals) — no dedicated endpoint in v1. |

**Prerequisite (backend, tracked outside this plan):** Step 9 —
[`../09-design-data-extensions.md`](../09-design-data-extensions.md) — adds the fields the design
needs before any screen is built: Topic `category` + `targetDate`, Goal `dueDate` +
`isAchievement` (the "Celebrate as an achievement ★" flag that distinguishes a ✓ done milestone from
a ★ one). It belongs to backend development and is **not** a Phase B step; the roadmap below assumes
it is done.

Anything that would change one of the locked rows above is a **system-wide decision** and comes back
to you. Library-level choices below are small technical decisions, listed for transparency.

---

## 3. Tech stack (proposed detail)

| Concern | Choice | Rationale |
| --- | --- | --- |
| Build tool | **Vite** | Fast dev server + HMR; first-class React + TS. |
| Language | **TypeScript** (strict) | Compile-time safety against API drift. |
| Styling | **Tailwind CSS** + CSS variables for theme tokens | Utility styling; tokens as CSS vars let one `data-theme` swap both themes without forking markup. |
| Routing | **React Router** | Standard SPA routing (Dashboard / Achievements / Reflections / auth). |
| Server state | **TanStack Query** (React Query) | Caching, mutations, optimistic updates, refetch — fits the goal/milestone CRUD cleanly. |
| Client/UI state | React Context (auth, theme, open-modal) | Small, no heavyweight store needed. |
| API types | **`openapi-typescript`** (types) + a thin typed `fetch` wrapper (`openapi-fetch`) | Generates types straight from `api/openapi.yaml`; stays close to fetch, no codegen sprawl. |
| Fonts | Newsreader, Hanken Grotesk, Spline Sans Mono | Per handoff; self-hosted via the build (or Google Fonts) — decided in Step B2. |
| Icons | Inline SVG / a lightweight icon set | Handoff uses text glyphs (✓ ★ ✕ ＋ ▦ ✎ ●); recreate as SVG components. |
| Unit/component tests | **Vitest + React Testing Library** + **MSW** (API mocking) | Same "precise and verified" bar as the backend. |
| E2E (optional) | **Playwright** | A couple of golden-path flows; flagged, not required for every step. |
| Lint/format | **ESLint + Prettier** (+ `eslint-plugin-jsx-a11y`) | Enforced like `golangci-lint` was. |

**Location:** the app lives in **`web/`** in this repo (monorepo), so backend and frontend evolve
together and the OpenAPI contract is one `../api/openapi.yaml` away.

**Dev wiring:** Vite dev server proxies `/api` and `/openapi.yaml` to the Go API on `:8080`, so the
browser sees a **single origin** in development — the HttpOnly refresh cookie "just works" and we
avoid CORS entirely locally. (Production serving is Phase C.)

---

## 4. Architecture

```
web/
  index.html
  vite.config.ts            # React plugin + dev proxy to :8080
  tailwind.config.ts        # tokens -> theme-aware utilities
  tsconfig.json
  package.json
  src/
    main.tsx                # app bootstrap (providers: Query, Theme, Auth, Router)
    App.tsx                 # routes + app shell
    routes/                 # Dashboard, Achievements, Reflections(placeholder), Login, Register
    components/             # reusable primitives: Button, Input, Textarea, Label, Modal,
                            #   ProgressBar, Ring, Logo, Chip, Toggle, Swatch, Avatar, Confetti
    features/
      auth/                 # auth context, login/register forms, silent refresh
      dashboard/            # dashboard view + hooks
      topics/               # topic detail modal, create-goal modal, topic hooks
      goals/                # milestone list, add-milestone modal, goal mutations
      achievements/         # derived "Wins" view + celebration modal
    api/                    # generated openapi types + typed client + query hooks
    theme/                  # tokens.css (both themes), ThemeProvider, useTheme (persisted)
    lib/                    # date/countdown, motivation bands, formatters
    styles/                 # global css, tailwind entry, @font-face
    test/                   # setup, MSW handlers
```

**Principles (mirroring the backend's layering):**
- **Design tokens are the single source of truth.** Colors/type/spacing/radii/shadows live as CSS
  variables in two theme sets; components consume tokens, never raw hexes. One `data-theme` attribute
  on `<html>` (persisted to `localStorage`) swaps the whole app — no forked markup.
- **API boundary is typed and generated.** `src/api` exposes typed functions + React Query hooks; no
  component builds URLs or hand-types responses. Types come from `api/openapi.yaml`, so a backend
  contract change surfaces as a TypeScript error.
- **Feature-first folders.** Each screen/feature owns its components, hooks, and tests; shared UI
  lives in `components/`. New features = a new folder + a route, without touching existing ones
  (the same expandability rule the backend follows).
- **Derived, not stored, UI truth.** Progress %, motivation band, year-elapsed %, and achievement
  status are computed from API data in `lib/` — matching the backend's "computed, never stored".
- **Accessibility is not a final coat.** Semantic HTML, focus management, and contrast are built into
  the primitives (Modal, Button, Toggle) from Step B2, then audited in B7.

---

## 5. Design system (from the handoff)

The exhaustive token values are in `design_handoff_this_is_your_year/README.md` §"Design Tokens"; we
implement them as two CSS-variable sets. Summary of what Step B2 encodes:

- **Type:** Newsreader (serif — headings, titles, big numbers, italic quotes), Hanken Grotesk (sans —
  body, buttons, inputs), Spline Sans Mono (mono — uppercase labels/meta, letter-spaced).
- **Color:** dark ("journal at night") and light ("paper") sets — surfaces, borders, text tiers, and
  four accents (teal primary/CTA, blue, gold achievement, rose). Same layout, swapped tokens.
- **Shape:** card radius 18px, small 10–14px, pills/bars 999px; card padding 26–32px; header 15×28;
  input 12×14. Shadows per theme. Progress bars 8px (year) / 6px (per-goal).
- **Logo:** SVG timer-ring + gold star; the ring's `stroke-dasharray` binds to overall year progress.
- **Motion:** hover lightening on cards/CTAs; confetti on the celebration (respect
  `prefers-reduced-motion`).

---

## 6. Roadmap (ordered steps)

> One step at a time. For each I'll write a detailed `plans/frontend/NN-*.md`, get your approval,
> implement, verify (build / typecheck / lint / unit+component tests / a11y / live run against the
> real API), update docs, and wait for your **"Task completed"** before the next.

> **Prerequisite:** backend Step 9 ([`../09-design-data-extensions.md`](../09-design-data-extensions.md))
> lands the design-gap fields first. The steps below are Phase B proper.

**B1 — App scaffold & design system** · `frontend/01-scaffold-design-system.md`
Create `web/` (Vite + React + TS + Tailwind). Encode both **theme token sets** + `data-theme` toggle
(persisted) + fonts. Build the **core primitives** (Logo, Button, Input, Textarea, Label,
ProgressBar, Ring, Modal shell w/ backdrop+Esc+focus-trap, Chip, Toggle, Swatch, Avatar) and the
**app shell** (Header with nav + theme toggle + user block, Footer, mobile bottom tab bar), all
responsive. Routing skeleton with placeholder pages. *Outcome: the shell and a token-accurate
component gallery render in both themes at all breakpoints, on mock data.*

**B2 — API client & auth flow** · `frontend/02-api-client-auth.md`
Generate the typed client from `api/openapi.yaml`; build the **auth context** (access token in memory
+ **silent refresh** via the HttpOnly cookie, protected routes, `GET /me`, logout) and the **login /
register** screens (on-brand — these aren't in the handoff, so I design them from the tokens). Dev
proxy wired. *Outcome: a real user can register, log in, stay logged in across reloads, and log out.*

**B3 — Dashboard (Editorial Journal)** · `frontend/03-dashboard.md`
The home view wired to `GET /dashboard` + `GET /topics`: date eyebrow, greeting (name + time of day),
italic sub-message (dashboard `message`), **year progress bar** (elapsed % + "YOU ARE HERE" from the
`countdown`), and "The chapters" list (accent bar, title, ★ ACHIEVED tag at 100%, `status ·
done/total`, percent, mini bar). Empty state. Responsive incl. mobile stacked cards + bottom tab bar.
*Outcome: the dashboard reflects real data pixel-accurately in both themes.*

**B4 — Topic & milestone flows (CRUD popups)** · `frontend/04-topics-goals.md`
Topic-detail modal (motivational callout by band, progress ring, milestone list with ✓/★/open + due
dates), **Create big goal** modal (title, why, target date, category, accent swatch), **Add
milestone** modal (title, due date, notes, "Celebrate as an achievement ★" toggle), and the
mutations: add/edit/delete goal & milestones, mark milestone done/undone, mark goal achieved — with
optimistic updates and live progress recompute. Uses the Step-9 fields. *Outcome: full create/read/
update/delete for topics and milestones through the designed modals.*

**B5 — Achievements page, celebration & motivation system** · `frontend/05-achievements-motivation.md`
The **100% celebration** modal (confetti, medallion, stat strip, actions), triggered when a goal
reaches 100%; a shared **four-band motivation** util + the design's copy, used by both dashboard rows
and topic detail; and the **Achievements ("Wins")** page derived from achieved milestones/goals.
*Outcome: completing a goal celebrates it; the Wins tab lists achievements; one motivation source of
truth.*

**B6 — Accessibility, responsive polish & states** · `frontend/06-a11y-polish.md`
WCAG pass (modal focus-trap + return focus, keyboard nav, ARIA roles/labels, contrast in **both**
themes, `prefers-reduced-motion`), loading / error / empty states everywhere, the Reflections "coming
soon" placeholder, and a cross-breakpoint / cross-theme QA sweep (axe + Lighthouse). *Outcome: the
SPA is accessible, resilient, and polished across every screen, theme, and size.*

---

## 7. Cross-cutting concerns

- **Theming:** two CSS-variable sets; `data-theme` on `<html>`; toggle persists to `localStorage` and
  respects `prefers-color-scheme` on first visit.
- **Responsive:** desktop (centered 680px editorial column), tablet 768 (tightened, inline nav),
  mobile ≤430 (top nav → bottom tab bar; goals become stacked cards; modals → near-full-width sheets).
- **Data derivations (`lib/`):** progress %, motivation band, year-elapsed % / days-to-New-Year (from
  `countdown`), achievement status — all computed, unit-tested.
- **Testing:** Vitest + RTL + MSW for components/hooks; a11y assertions in-suite; optional Playwright
  golden paths. Every step ships with its tests, like the backend.
- **Error/loading UX:** consistent skeleton/spinner + error envelope handling (the backend's stable
  `{error:{code,message,fields}}` maps to inline field errors and toasts).

---

## 8. Open questions deferred to you (not blocking this plan)

Raised at the relevant step, not decided unilaterally:
- **Auth screen visuals:** login/register aren't in the handoff. I'll design minimal on-brand screens
  from the tokens in B3 — flag if you'd rather provide designs.
- **Motivation copy & thresholds:** the design's four-band wording differs slightly from the backend's
  Step-6 copy. Decide in B6 whether the frontend uses the design's copy (likely) and the exact
  band thresholds (~80% for "almost there").
- **Achievement celebration stats** ("5 mo to complete", "#1 of 2026"): which are computed vs dropped
  — decided in B6.
- **Production serving (Phase C):** Go server serving the built SPA (embedded `dist`) vs. a separate
  static host — deferred to Phase C.

---

## 9. Definition of done (per step)

A step is complete when: it typechecks (`tsc`) and builds (`vite build`); ESLint/Prettier pass; the
step's Vitest component/unit tests pass; the feature is reachable end-to-end **against the real Go
API** (run locally) and verified in **both themes** at the relevant breakpoints; and docs
(`docs/dev_run.md` frontend section / a `web/README.md`) are updated. Backend steps additionally keep
`go` build/vet/lint/tests and the OpenAPI spec green, as before.
