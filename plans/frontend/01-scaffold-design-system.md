# Step B1 — App scaffold & design system

> **Status:** Draft for approval
> **Parent:** [`00-system-plan.md`](00-system-plan.md) §6 (step B1) · Master: [`../00-system-plan.md`](../00-system-plan.md) §8 Phase B
> **Prerequisite:** Backend Step 9 (design-data extensions) — **done & approved**.
> **Approval gate:** No frontend code is written until you approve this document.

---

## 1. Objective

Stand up the `web/` React app and the **design-system foundation** every later screen builds on:
the two-theme token layer, the theme toggle, the shared UI primitives, the app shell (header /
footer / mobile tab bar), and a routing skeleton with placeholder pages. The proof it works is a
**component gallery** that renders every primitive in every state, pixel-accurate to the handoff, in
**both themes** at **desktop / tablet 768 / mobile 390**.

This step wires **no real API** — auth and data land in B2/B3. The shell renders on **mock data**
(placeholder user, sample chapters), exactly as the roadmap specifies.

### In scope
- `web/` project: **Vite + React 18 + TypeScript (strict)** + **Tailwind CSS v4**.
- **Design tokens** as CSS variables — full dark + light sets from the handoff — mapped into Tailwind
  utilities; one `data-theme` on `<html>` swaps the whole app.
- **Theme toggle** (sun/moon segmented control): persisted to `localStorage`, first-visit default
  from `prefers-color-scheme`, applied before paint (no flash).
- **Fonts:** Newsreader / Hanken Grotesk / Spline Sans Mono, self-hosted (no external CDN).
- **Primitives:** `Logo` (SVG timer-ring + star, progress-bindable), `Button` (teal/gold/outline
  variants + hover), `Input`, `Textarea`, `Label` (mono uppercase), `ProgressBar` (8px & 6px),
  `Ring` (dasharray %), `Modal` shell (backdrop + Esc + backdrop-click + focus-trap + return-focus),
  `Chip`, `Toggle` (on/off switch), `Swatch` (accent color picker), `Avatar` (gradient + initials),
  plus an `Icon` set (check, star, x, plus, grid, pencil, sun, moon, dot) as inline SVG.
- **App shell:** `Header` (logo + nav pills + theme toggle + user block/avatar), `Footer`
  (support email · GitHub · "Since 2026"), **mobile bottom tab bar** (Goals ▦ / Wins ★ / Notes ✎ /
  New ＋). Responsive across the three breakpoints.
- **Routing skeleton** (React Router): `/` Dashboard, `/achievements`, `/reflections` (coming-soon),
  and a **dev-only `/gallery`**. All are placeholders except the gallery.
- **Tooling:** ESLint (+ `jsx-a11y`) + Prettier, strict `tsconfig`, Vitest + RTL + jsdom, npm scripts.
- **Docs:** `web/README.md` (run/verify) + a frontend pointer in `docs/dev_run.md`.

### Explicitly NOT in this step
- No API client, no auth, no data fetching (B2). No dashboard/topic/goal screens with real content
  (B3–B5). No login/register UI (B2). No confetti/celebration (B5). No dev proxy is *required* to run
  the gallery (it's added in B2 with the API), though `vite.config.ts` will already declare it.

---

## 2. Small technical decisions (mine to make; listed for transparency)

These are library/implementation choices, not system-wide decisions. I'll proceed with these unless
you object:

| Decision | Choice | Why |
| --- | --- | --- |
| Tailwind version | **v4** (`@tailwindcss/vite`, CSS-first `@theme`) | Current major; maps our CSS-variable tokens straight to utilities with no `tailwind.config.ts` fork. (Master plan named a `tailwind.config.ts`; v4 replaces it with a CSS `@theme` block — same outcome, less config.) |
| Fonts | **Self-hosted via `@fontsource*`** npm packages | No runtime CDN dependency; works offline and in CI/tests; deterministic. (Mock used Google Fonts CDN — we localize it.) |
| Token delivery | **CSS variables** in `:root` (dark) + `:root[data-theme="light"]` | One attribute swaps both themes; Tailwind utilities reference the vars. Matches handoff guidance exactly. |
| No-flash theme | Tiny inline script in `index.html` sets `data-theme` before React mounts | Prevents a wrong-theme flash on first paint. |
| Component gallery | A **dev-only route** `/gallery`, tree-shaken out of prod builds | Gives us the "token-accurate gallery" acceptance artifact without shipping it. |
| State for theme | React Context (`ThemeProvider`) + `localStorage` | Small; no store needed (per system plan §3). |
| Test env | **Vitest + RTL + jsdom**, `@testing-library/jest-dom` matchers | Mirrors the backend's "every step ships tests" bar. |

If any of these should be a locked decision instead, tell me and I'll treat it as such.

---

## 3. Files created (B1)

```
web/
  index.html                     # #root + no-flash theme script + font preloads
  package.json                   # deps + scripts (dev/build/preview/lint/format/test/typecheck)
  vite.config.ts                 # React + Tailwind plugins; dev proxy /api,/openapi.yaml -> :8080
  tsconfig.json / tsconfig.node.json
  .eslintrc.cjs (+ jsx-a11y) / .prettierrc / .gitignore
  vitest.config.ts / src/test/setup.ts
  src/
    main.tsx                     # providers: Theme, Router (Query/Auth added in B2)
    App.tsx                      # <AppShell> + <Routes>
    styles/
      tokens.css                 # CSS-variable token sets (dark + light)
      global.css                 # @tailwind + base element styles + @font-face via @fontsource
    theme/
      ThemeProvider.tsx  useTheme.ts  theme.ts (Theme type, storage key, resolve helper)
    components/
      Logo.tsx  Icon.tsx  Button.tsx  Input.tsx  Textarea.tsx  Label.tsx
      ProgressBar.tsx  Ring.tsx  Modal.tsx  Chip.tsx  Toggle.tsx  Swatch.tsx  Avatar.tsx
      index.ts                   # barrel
    layout/
      AppShell.tsx  Header.tsx  Footer.tsx  ThemeToggle.tsx  BottomTabBar.tsx  UserBlock.tsx
    routes/
      Dashboard.tsx  Achievements.tsx  Reflections.tsx   # placeholders (real content later)
      Gallery.tsx                                        # dev-only component gallery
    lib/
      cn.ts                      # className merge helper
      mockData.ts                # placeholder user + sample chapters (shell/gallery only)
    __tests__/                   # co-located or here: primitive + toggle + modal tests
```

Everything is additive under `web/`; no existing Go files change. `web/dist/`, `web/node_modules/`
are git-ignored (root `.gitignore` gets `/web/node_modules/` + `/web/dist/`).

---

## 4. Design-token layer

`tokens.css` encodes the handoff's **Design Tokens** section verbatim as CSS variables. Two sets:

- **Dark** (default, `:root`): bg `#0c0b08`→radial `#1c1a14`, card `#131210`, header/footer `#16140f`,
  popup `#1a1813`, muted fill `#26231d`, borders `#2a2722`/`#232019`/`#34302a`, text
  `#ece8e1`/`#b3ada3`/`#8a847a`/`#55504a`, accents teal `#3ec5a0` (hover `#4fd3ae`), blue `#5b8def`,
  gold `#e8b765` (hover `#f0c47e`), rose `#c2738f`, track `#2a2722`; shadows
  `0 16px 50px rgba(0,0,0,.45)` (card) / `0 30px 80px rgba(0,0,0,.6)` (popup).
- **Light** (`:root[data-theme="light"]`): bg `#e3d9c8`→`#efe7da`, paper `#faf6ee`, white card
  `#ffffff`, muted `#f0e9dc`, borders `#e7dfd2`/`#ece4d6`/`#ddd3c4`, text
  `#2b2620`/`#6e665a`/`#9c9384`/`#9a9184`, accents teal `#2f9e7e` (hover `#36b08c`), blue `#3f6fd6`,
  gold `#d29b3f` (hover `#dcab57`, deep text `#b8841f`), rose `#c2738f`, track `#ece4d6`; shadows
  `0 16px 50px rgba(120,100,70,.13)` / `0 30px 80px rgba(90,72,40,.22)`.

Naming: semantic roles, not raw colors — `--bg`, `--surface`, `--surface-elevated`, `--bar`,
`--fill-muted`, `--border`, `--border-subtle`, `--border-popup`, `--text`, `--text-2`, `--text-muted`,
`--text-faint`, `--accent-teal`(+`-hover`), `--accent-blue`, `--accent-gold`(+`-hover`,`-deep`),
`--accent-rose`, `--track`, `--shadow-card`, `--shadow-popup`, plus radii (`--r-card:18px`,
`--r-sm:12px`, `--r-pill:999px`) and the two bar heights.

Tailwind v4 `@theme` in `global.css` maps these to utilities (`bg-surface`, `text-muted`,
`border-subtle`, `rounded-card`, `shadow-card`, `text-accent-teal`, …). **Components consume tokens
only** — no raw hex in component files (an ESLint note / review check enforces it).

Type scale (desktop, from handoff): H1 36/1.22 Newsreader 500; H2 20–24/600; row title 19/600; popup
title 26–30/600; body 14–15; mono label 11 uppercase `letter-spacing:.08–.2em`; big stat 22–34
Newsreader 600. Encoded as Tailwind font-size/family utilities (`font-serif`/`font-sans`/`font-mono`).

---

## 5. Theme toggle

- `ThemeProvider` resolves initial theme: `localStorage["tiyy-theme"]` → else
  `matchMedia('(prefers-color-scheme: dark)')` → default dark. Writes `data-theme` to
  `document.documentElement`, persists on change, exposes `{ theme, toggle, setTheme }`.
- `index.html` runs the same resolution in a tiny inline `<script>` before the bundle loads, so the
  correct `data-theme` is set pre-paint (no flash). `ThemeProvider` then adopts it.
- `ThemeToggle` = the segmented sun/moon control (26px options desktop, 24 tablet, 22 mobile),
  `role="switch"` / `aria-checked`, keyboard-operable.

---

## 6. Primitives (props, states, a11y)

Each ships with the states the handoff shows and is exercised in the gallery + a unit test where it
carries logic.

- **Logo** — SVG faint full ring + teal partial ring (`stroke-linecap:round`, rotated −90°) + gold
  5-point star; `progress` prop (0–1) drives `stroke-dasharray`; optional wordmark (Newsreader 600).
- **Icon** — inline SVG set: check, star, x, plus, grid, pencil, sun, moon, dot; `size`/`color`
  (token) props; `aria-hidden` unless labeled.
- **Button** — variants `teal` (primary CTA), `gold` (achievement), `outline`; hover lightens fill
  per tokens; sizes; `disabled`; renders `<button>`; focus-visible ring.
- **Input / Textarea** — inset bg, 1px border, 10–12px radius, 12×14 padding; focus ring; paired with
  **Label** (11px mono uppercase); error slot (styled now, wired in B2).
- **ProgressBar** — `value` 0–1, `height` 8|6, optional gradient (teal→gold year bar); track token;
  `role="progressbar"` + aria values.
- **Ring** — `value` 0–1, `size` (84 default), accent stroke, centered % label; SVG dasharray.
- **Modal** — portal; dimmed radial backdrop; **Esc closes**, **backdrop-click closes**,
  **focus-trap** + **return focus** to opener; `role="dialog"` `aria-modal` labelled by title;
  body-scroll lock; near-full-width sheet on mobile. (The reusable shell; concrete popups are B4/B5.)
- **Chip** — pill for "TRY:" examples / tags; optional accent dot.
- **Toggle** — on/off switch (accent track + white knob when on); `role="switch"` `aria-checked`;
  label + helper text row layout (the "Celebrate as an achievement ★" pattern).
- **Swatch** — accent color circle (teal/blue/gold/rose), selected = double-ring; radiogroup
  semantics for the picker.
- **Avatar** — 38px gradient circle (135° teal→blue) + initials 700; size prop.

---

## 7. App shell & routing

- **Header** — logo+wordmark left, nav pills (`Dashboard` active/filled, `Achievements`,
  `Reflections`), then `ThemeToggle` + `UserBlock` (name 13/600 + role 11 muted) + `Avatar`.
  Padding 15×28. Tablet: inline, tightened (logo 27). Mobile: nav hidden, toggle left of avatar.
- **Footer** — support email · GitHub link · "Since 2026", mono, full width.
- **BottomTabBar** — mobile-only fixed bar: Goals ▦ (active/accent) · Wins ★ · Notes ✎ · New ＋,
  each a route/action target (New opens create-goal in B4; placeholder handler now).
- **AppShell** composes Header + `<main>` (centered 680px editorial column) + Footer + (mobile)
  BottomTabBar. Uses `mockData` for the user block.
- **Routes:** `/` Dashboard, `/achievements`, `/reflections` (visible "Coming soon" placeholder),
  `/gallery` (dev-only — excluded from prod build via `import.meta.env.DEV`). Placeholder route
  bodies are simple titled sections; real content arrives in later steps.

---

## 8. Component gallery (`/gallery`)

A single scrollable page grouping every primitive and shell piece with all variants/states side by
side, plus a breakpoint note. It's the **acceptance artifact**: open it, flip the theme, resize to
768 and 390 — everything must match the handoff. Dev-only.

---

## 9. Testing & tooling

- **Vitest + RTL + jsdom.** Tests this step: `ThemeProvider` (persist + prefers-color-scheme +
  toggle), `Modal` (Esc / backdrop-click / focus-trap / return-focus / aria), `Toggle` & `Swatch`
  (aria-checked + selection), `ProgressBar`/`Ring` (aria values + dasharray math), `Button` (variant
  classes + disabled). Render smoke test for `AppShell` in both themes.
- **ESLint** (typescript + react-hooks + **jsx-a11y**) + **Prettier**; `tsc --noEmit` strict.
- **Scripts:** `dev`, `build`, `preview`, `lint`, `format`, `typecheck`, `test`, `test:watch`.

---

## 10. Verification (definition of done for B1)

1. `npm run typecheck` — clean (strict). 2. `npm run lint` — clean (incl. a11y). 3. `npm run format
-- --check` — clean. 4. `npm test` — all green. 5. `npm run build` — `vite build` succeeds; `/gallery`
absent from the prod bundle. 6. **Live check:** `npm run dev`, open `/gallery` and the shell — verify
in **dark & light** at **1320 / 768 / 390**: tokens/type/spacing/radii/shadows match the handoff; the
theme toggle swaps instantly and persists across reload with **no flash**; Modal traps focus and
closes on Esc/backdrop; mobile shows the bottom tab bar. 7. `web/README.md` + `docs/dev_run.md`
frontend pointer updated. Nothing committed (per workflow).

---

## 11. Open questions (non-blocking; I proceed with the stated default)

- **Footer GitHub/support values** — I'll use the repo's existing support email / a placeholder GitHub
  URL for now; correct them anytime.
- **Reflections placeholder** — shown as an on-brand "Coming soon" per the locked decision; no nav
  removal. Confirmed by the system plan, so I proceed.

Nothing here changes a locked (system-wide) decision. On your **approval**, I implement B1, verify per
§10, update docs, and report back for your **"Task completed"** before B2.
