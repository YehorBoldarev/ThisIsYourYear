# Step B6 — Accessibility, responsive polish & states

> **Status:** Draft for approval
> **Parent:** [`00-system-plan.md`](00-system-plan.md) §6 (step B6)
> **Builds on:** B1–B5 — **done & approved**.
> **Approval gate:** No code is written until you approve this document.

---

## 1. Objective

The features are all in; B6 makes the SPA **accessible, resilient, and polished** across
every screen, theme, and size — the final Phase-B step. This is an **audit-and-harden**
pass, not new features: close WCAG gaps, guarantee `prefers-reduced-motion` everywhere,
make sure every async surface has loading / error / empty states, and do a
cross-theme / cross-breakpoint QA sweep with an objective, automated a11y gate.

### In scope
- **Keyboard & focus:** a **skip-to-content** link; verify the modal focus-trap +
  return-focus (B1 `Modal`, B5 `CelebrationModal`) on every popup; `aria-current` on the
  active nav item (header + bottom tab bar); visible focus ring audit on all interactive
  elements in both themes.
- **ARIA & semantics:** landmark check (`banner`/`nav`/`main`/`contentinfo`), accessible
  names for icon-only buttons (audit the theme toggle, close buttons, milestone circles,
  row edit/delete), `role`/`aria-labelledby` on the celebration dialog, `aria-live` for
  async status (form errors, optimistic failures) where a sighted user sees a change but
  a screen-reader user wouldn't.
- **Reduced motion:** a **global `prefers-reduced-motion` guard** (neutralize
  `transition`/`animation`, including the `animate-pulse` skeletons and hover transitions)
  layered on top of B5's component-level Confetti gate.
- **States everywhere:** audit loading / error / empty across dashboard, Wins, topic
  detail, the forms, and auth screens; fill any gap (e.g. a shared inline error/retry, a
  consistent empty pattern) and confirm the **Reflections** placeholder (already present)
  reads well and is on-brand.
- **Contrast:** verify text/UI contrast meets WCAG AA in **both** themes; adjust token
  usages (not the tokens themselves without your sign-off) where a combination falls short.
- **Automated a11y gate:** add an in-suite accessibility assertion (see **D1**) over the
  key screens so regressions are caught in CI alongside the existing 71 tests.
- **QA sweep:** a documented cross-theme (dark/light) × breakpoint (desktop / 768 / 390)
  checklist, executed and recorded (see **D4** for how it's evidenced).

### Explicitly NOT in this step
- No new product features, no backend changes, no `schema.d.ts` change.
- No **Reflections** functionality (locked as "coming soon").
- No visual redesign — polish within the existing tokens/handoff, not a new look.
- Production serving of the SPA (embedding `dist` in the Go binary) is **Phase C**.

---

## 2. Current state (audited — what already passes)

So the plan targets real gaps, not solved ones:
- **Landmarks present:** `header` (banner), `nav aria-label="Primary"` (header + bottom
  tab bar), `main`, `footer`. Missing: a **skip link** and a stable `id` on `main`.
- **Focus management:** `Modal` traps focus, restores it on close, closes on Esc;
  `CelebrationModal` mirrors it. To re-verify per popup, not rebuild.
- **Focus ring:** a global `:focus-visible` teal outline exists. Audit coverage only.
- **Reduced motion:** only **Confetti** and its keyframe are gated. **Gap:** the
  `animate-pulse` skeletons (dashboard, Wins, topic-detail) and `transition-*` hovers
  are not — a global guard closes this.
- **Icon-only buttons:** most have `aria-label`/`title` (close, sign-out, milestone
  circle, edit/delete). Audit to confirm none slipped.
- **States:** dashboard and Wins have loading/error/empty; forms map field errors; topic
  detail has pending/error/empty for goals. **Gap:** no `aria-live` so screen-reader
  users hear optimistic failures / validation; nav active state lacks `aria-current`.

---

## 3. Small technical decisions (mine; for transparency)

| Decision | Choice | Why |
| --- | --- | --- |
| Skip link | A visually-hidden link that appears on focus, targets `#main` | Standard, cheap, high-value for keyboard users. |
| Reduced-motion guard | One scoped `@media (prefers-reduced-motion: reduce)` block in `global.css` shortening animations/transitions to ~0 | One rule covers skeletons + hovers + future motion; Confetti stays gated in JS too. |
| Live regions | A tiny `aria-live="polite"` region for form/status errors; `role="alert"` for the destructive-confirm error | Announces what sighted users already see; no visual change. |
| `aria-current` | Add `aria-current="page"` to the active `NavLink` (header + tabs) | Correct nav semantics for SR users. |

None touch a locked/system-wide decision. Any **token value** change (if a contrast fix
needs one) I'll raise to you rather than edit the palette unilaterally.

---

## 4. Files (created ·/ changed) — indicative

```
web/src/
  layout/
    AppShell.tsx        ~ add skip link + id="main" on <main>
    Header.tsx          ~ aria-current on active NavLink; focus-ring/name audit
    BottomTabBar.tsx    ~ aria-current on active tab
  components/
    (audit) Modal, Button, Toggle, Input, Textarea, Field — accessible-name / focus checks
    LiveRegion.tsx      + small polite aria-live announcer (shared)
  features/
    dashboard/*, topics/*, achievements/*  ~ wire LiveRegion for async errors; state audit
  styles/
    global.css          ~ global prefers-reduced-motion guard; skip-link + sr-only utilities
  test/
    a11y.test.tsx       + axe assertions over key screens (login, dashboard, Wins, a modal)
```

Exact touched files are whatever the audit turns up; the above is the expected shape.

---

## 5. Decisions for you (defaults recommended; flag any before I build)

**D1 — Automated a11y gate.**
- **(Recommended) Add `vitest-axe`** (jest-axe for Vitest) and assert **no violations** on
  a handful of rendered key screens (login, dashboard populated, Wins, an open modal +
  the celebration). Objective, runs in CI with the existing suite, catches regressions.
- **(Alt) Manual only** — no new dev-dependency; rely on the QA checklist + browser axe.
  Lighter footprint, but nothing guards against regressions.

**D2 — Reduced-motion scope.**
- **(Recommended) Global guard** — one `@media (prefers-reduced-motion: reduce)` rule
  neutralizes animations/transitions app-wide (skeletons show static, hovers snap), plus
  the existing JS gate for Confetti.
- **(Alt) Per-component** — gate each animation individually. More surgical, more code,
  easier to miss one.

**D3 — Contrast fixes, if any are needed.**
- **(Recommended) Adjust token _usage_** (swap a `text-text-faint` for `text-text-muted`
  in a low-contrast spot) and **bring any palette-value change to you** before touching
  `tokens.css`.
- **(Alt) Change token values** directly — I'd avoid this without your sign-off since the
  palette is a system-wide design decision.

**D4 — How the QA sweep is evidenced** *(the honesty point).*
- **(Recommended) Automated axe (D1) as the objective gate + a written manual checklist**
  in `docs/dev_run.md` (dark/light × desktop/768/390, per screen) that I run and record,
  driving the app with the browser tooling available in this environment where it works.
  I will **not** fabricate Lighthouse scores I can't produce headlessly here — if a full
  Lighthouse run isn't runnable, I'll say so and give the axe results + checklist instead.
- **(Alt) You run Lighthouse** on your machine from a checklist I provide, and I fold the
  numbers into the doc.

---

## 6. Testing & verification (definition of done)

- **Automated:** existing 71 tests stay green; **+ axe assertions** (D1) pass on the key
  screens; targeted unit/interaction tests for the new bits (skip link focuses `#main`;
  `aria-current` on the active route; a form error is announced via the live region).
- **Reduced motion:** a test (mocking `prefers-reduced-motion`) asserts the skeletons /
  animated elements carry no active animation; Confetti already covered.
- **Manual sweep:** the dark/light × desktop/768/390 checklist executed and recorded in
  `docs/dev_run.md`, plus a keyboard-only walkthrough (tab order, skip link, modal trap +
  return focus, Esc) of the core flows.
- **Gates:** `typecheck` · `lint` · `format:check` · `test` · `build` · `gen:api` no-op.
- Docs (`docs/dev_run.md` frontend section + changelog, `web/README.md`) updated.
  **Nothing committed.**

This is the last Phase-B step; on approval I audit, harden, add the automated a11y gate,
run the sweep, update docs, and report back for your **"Task completed"** — after which
Phase B is complete and Phase C (production serving) is the next conversation.

---

On approval I implement B6 per §4–§6 and report back with the axe results and the
completed cross-theme / cross-breakpoint checklist.
