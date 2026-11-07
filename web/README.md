# This is Your Year — Web (Phase B frontend)

React + TypeScript + Vite + Tailwind CSS v4 single-page app for the Year Goals
tracker. Through **Step B6** (Phase B complete) it has the design-system foundation
(B1), a real **auth flow + typed API client** (B2), the **dashboard** wired to live
data (B3), the **topic & milestone CRUD modals** with optimistic updates (B4), the
**achievements page, 100% celebration & four-band motivation system** (B5), and an
**accessibility & polish** pass (B6). Phase C (production serving) is next.

## Prerequisites

- Node 20+ (developed on Node 24), npm 10+.

## Commands

```bash
npm install        # once
npm run dev        # dev server (http://localhost:5173), proxies /api → :8080
npm run build      # typecheck + production build to dist/
npm run preview    # serve the production build
npm run typecheck  # tsc -b, no emit
npm run gen:api    # regenerate src/api/schema.d.ts from ../api/openapi.yaml
npm run lint       # ESLint (incl. jsx-a11y)
npm run format     # Prettier write  (format:check to verify)
npm test           # Vitest (run once)  ·  npm run test:watch
```

## What's here (B1)

- **Design tokens** — `src/styles/tokens.css` holds the full dark + light token
  sets from the design handoff as CSS variables; `src/styles/global.css` maps
  them onto Tailwind utilities via `@theme inline`. One `data-theme` attribute on
  `<html>` swaps the whole app; components consume semantic tokens, never raw hex.
- **Theme** — `src/theme/` provides `ThemeProvider` + `useTheme`. The choice
  persists to `localStorage` and defaults from `prefers-color-scheme`. An inline
  script in `index.html` applies it before paint (no flash).
- **Fonts** — Newsreader / Hanken Grotesk / Spline Sans Mono, self-hosted via
  `@fontsource-variable/*` (imported in `src/main.tsx`); no runtime CDN.
- **Primitives** — `src/components/`: `Logo`, `Icon`, `Button`, `Input`,
  `Textarea`, `Label`, `ProgressBar`, `Ring`, `Modal` (backdrop + Esc +
  focus-trap + return-focus), `Chip`, `Toggle`, `SwatchGroup`, `Avatar`.
- **App shell** — `src/layout/`: `Header` (logo + nav + theme toggle + user
  block), `Footer`, `BottomTabBar` (mobile), composed by `AppShell`.
- **Routes** — `src/routes/`: `Dashboard`, `Achievements`, `Reflections`
  (placeholders), `Login`/`Register`, and a **dev-only** `/gallery`.

## What's here (B2 — auth & API client)

- **Typed client** — `src/api/`: `schema.d.ts` is generated from the OpenAPI
  contract (`npm run gen:api`, checked in). `client.ts` wraps `openapi-fetch`,
  injects the access token, and does **refresh-on-401 retry** (single-flight).
- **Auth** — `src/auth/`: `AuthProvider` holds the access token **in memory**
  (the durable credential is the HttpOnly refresh cookie), does a **silent refresh
  on boot** (so reloads stay logged in), and exposes `login/register/logout` via
  `useAuth`. `RequireAuth` gates protected routes and redirects to `/login`.
- **Screens** — on-brand `Login` / `Register` (auto-login after register) with
  inline field-error mapping from the `{error:{code,message,fields}}` envelope.
- **Server state** — `@tanstack/react-query` provider installed for data hooks.

## What's here (B3 — dashboard)

- **Data hooks** — `src/features/dashboard/useDashboard.ts` (`GET /dashboard`) and
  `src/features/topics/useTopics.ts` (`GET /topics`), typed via the client and cached
  by TanStack Query.
- **Dashboard** — `src/features/dashboard/Dashboard.tsx`: date eyebrow + greeting,
  the italic sub-message, the year-progress bar ("YOU ARE HERE · N% · D days left"),
  and "The chapters" list (per-topic accent bar, ✓ COMPLETE at 100%, status ·
  done/total, percent, mini bar). Loading / error / empty states included.
- **Derivations** — `src/lib/greeting.ts` (time-of-day greeting). Motivation copy comes
  from the API (see B5 below).
- Opening a chapter and creating goals/milestones are wired up in **B4** below.

## What's here (B4 — topic & milestone flows)

- **Popups** — `src/features/topics/`: `TopicDetailModal` (progress ring +
  milestone checklist with ✓/★/open states, due dates, per-row edit/delete),
  `TopicFormModal` (create/edit a big goal — title, why, target date, category,
  accent swatch), `MilestoneFormModal` (add/edit a milestone + the "Celebrate as an
  achievement ★" toggle), and `ConfirmDelete` (inline destructive confirm). A
  feature-local `ModalHost` tracks which popup is open (one at a time; child popups
  return to the detail view on close).
- **Mutations** — `src/features/topics/mutations.ts`: typed create/update/delete
  hooks for topics and goals with **optimistic update → rollback → settle-invalidate**;
  toggling a milestone recomputes the topic's progress and the dashboard aggregate in
  the cache so the UI never waits on the round-trip. `useTopicGoals` loads a chapter's
  milestones (`GET /topics/{id}/goals`).
- **Helpers** — `src/lib/date.ts` (DD-MM-YYYY ⇄ native `<input type="date">` +
  "Due Oct 4" label); `ACCENT_HEX`/`accentFromHex` map the swatch picker to the API's
  stored `color` hex. Chapter rows are now clickable and open the detail popup.

## What's here (B5 — achievements, celebration & motivation)

- **Motivation system** — the API computes the copy for the design's four bands (Just
  started / Building momentum / In sight / A promise kept) and returns it on every topic as
  `motivation: { band, label, title, message }` (source: `internal/domain/motivation.go`).
  The chapter rows and topic-detail callout render it as-is; `src/lib/motivation.ts` only
  maps `band` → tone colour via `bandToneClass`.
- **Celebration** — `src/features/celebration/CelebrationModal.tsx` (design screen 7):
  gold star-in-glow medallion, the `Confetti` primitive (`src/components/Confetti.tsx`,
  skipped under `prefers-reduced-motion`), a congratulatory quote, and a computed stat
  strip (`src/lib/achievements.ts` → `celebrationStats`: milestones · months-to-complete
  · rank). `ModalHost` opens it when completing a milestone brings its chapter to 100%.
- **Wins page** — `src/features/achievements/`: `useAchievements` fans out
  `GET /topics/{id}/goals` across the year's chapters (there is **no** achievements
  endpoint) and `deriveAchievements` filters to completed, starred (★) milestones,
  newest first; `AchievementsPage` + `AchievementCard` render them with loading / error /
  empty states. Wired to the `/achievements` route (bottom-tab "Wins ★").

## What's here (B6 — accessibility & polish)

- **Keyboard & landmarks** — `src/layout/AppShell.tsx` adds a **skip-to-content** link
  targeting the `#main` landmark; the active nav item carries `aria-current="page"`
  (react-router default, verified by test). Modals (topic-detail, forms, confirm,
  celebration) trap focus and restore it on close.
- **Announcements** — form-level and confirm-delete errors use `role="alert"`; field
  errors are tied to their input via `aria-describedby`.
- **Reduced motion** — a global `@media (prefers-reduced-motion: reduce)` guard in
  `src/styles/global.css` neutralizes skeleton pulses and hover transitions, on top of
  the JS gate that removes confetti entirely.
- **Automated a11y gate** — `src/test/a11y.test.tsx` runs **axe** (`vitest-axe`) over the
  login screen, a populated dashboard, the Wins page, and an open modal. jsdom can't compute
  colour contrast, so `src/styles/contrast.test.ts` checks it analytically: every text token
  against every surface in `tokens.css`, both themes, at WCAG AA. Accent-coloured text uses the
  `-ink` utilities (`text-teal-ink`, `text-gold-ink`, `text-blue-ink`); the plain accent
  utilities are for fills, strokes and borders.

## Component gallery

Run `npm run dev` and open **http://localhost:5173/gallery** to see every
primitive in every state. Flip the theme toggle in the header and resize to
768 / 390 to verify both themes and all breakpoints.

## Dev proxy

`vite.config.ts` proxies `/api` and `/openapi.yaml` to the Go API
(`http://localhost:8080`, override with `VITE_API_TARGET`). Not needed for B1
(no API calls yet); it's used from Step B2. Start the backend with
`make compose-up && make run` from the repo root.

## Production (single origin)

In production the Go binary serves this SPA **and** the API from the same origin —
no CORS, no separate static host. From the repo root, `make build-all` builds `web/`,
stages `dist` under the Go embed directory, and compiles a self-contained binary
(`go build -tags embedui`); the multi-stage `Dockerfile` does the same. The dev proxy
above is development-only. The pre-paint theme script lives in `public/theme-init.js`
(external, not inline) so production can keep a strict `script-src 'self'` CSP. See
`../docs/dev_run.md` §6 for details.
