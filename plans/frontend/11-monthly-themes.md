# Step D2 — Monthly themes

> **Status:** Draft **for discussion** (2026-11-11). This is not a plan to approve yet. It frames
> the idea, the constraints and the open questions so we can decide the scope together; an
> implementation plan follows that discussion.
> **Parent:** your request (2026-11-11): *themes of the application depending on the month.*
> **Related:** [`10-preparation-stage.md`](10-preparation-stage.md). The preparation stage runs in
> November/December, so a "New Year's Eve" look could be the first theme.

---

## 1. The idea

The app is about **one year passing**. Letting its look change with the month makes that passage
visible: a quiet signal that time is moving and the year is being lived. It would also make
opening the app on the first of the month feel a little special.

## 2. What already exists (and constrains the design)

- **Design tokens** in `web/src/styles/tokens.css`: two themes (`:root[data-theme='dark'|'light']`),
  surfaces, text, and the accents teal/gold/blue/rose.
  - Accent **text** uses separate `-ink` tokens; plain accents are for fills.
  - Tailwind v4 reads all of them via `@theme` in `global.css`.
- **Contrast is test-enforced** (`src/styles/contrast.test.ts`): every text token against every
  surface, in both themes, must meet WCAG AA. Any monthly palette has to pass the same test. The
  test would be extended to loop over the months.
- **The pre-paint theme script** (`public/theme-init.js`) sets `data-theme` before the first paint.
  It must stay an external file because of the strict CSP. A `data-month` attribute would be set
  the same way, with no flash.
- **Users pick topic colours** (a topic's stored accent). A monthly palette must not clash with
  them, or must leave them alone.
- **The light/dark toggle** belongs to the user, and monthly themes must work in both modes.
- **The design handoff** (`design_handoff_this_is_your_year/`) has one palette. Monthly looks need
  new mockups, from you or a designer, before implementation.

## 3. What could change with the month

From lightest to heaviest:

| Level | What changes | Effort | Risk |
|---|---|---|---|
| 1 · Accent | One "month accent" token, used by the year bar's gradient, the logo ring, the active nav, and the countdown | Small | Low |
| 2 · Palette | The accent set (and maybe surface tints) per month, both modes | Medium | Contrast must pass for 12 × 2 palettes |
| 3 · Motif | A decorative element per month (e.g. frost lines in January, blossom in April, fireworks on New Year's Eve), purely decorative and `aria-hidden` | Medium | Must stay subtle; respect `prefers-reduced-motion` |
| 4 · Words | A month eyebrow or line in the greeting ("November — the quiet month before the next chapter"), server-owned like the motivation copy | Small | Low |

## 4. Questions to discuss

1. **Scope:** which levels? A sensible start is **1 + 4** (accent + a line of copy), with 2 or 3
   later.
2. **12 months or 4 seasons?** Four seasons means half the design work (8 palettes instead of 24
   with both modes).
3. **Whose month?** The browser's local month, which is what users expect, rather than the server's.
4. **Hemisphere:** seasons are inverted in the southern hemisphere. Ignore it (family and friends
   are in Europe), or make it a setting?
5. **Opting out:** a "Classic" choice next to Light/Dark in the theme menu, for people who prefer
   the original look?
6. **Topic colours:** leave the user-chosen topic accents untouched (recommended), or tint them
   with the month?
7. **Special days:** a New Year's Eve / first-week-of-January look (it fits the preparation stage),
   and maybe the user's own "year anniversary"?
8. **Design source:** who designs the palettes and motifs? Mockups in the handoff format first, or
   iterate in the dev-only Gallery page?

## 5. A technical sketch (for later; no decisions yet)

- **Pre-paint attribute:** `theme-init.js` also sets `data-month="01".."12"` (or `data-season`)
  from the local date.
- **Token overrides:** `tokens.css` adds overrides such as
  `:root[data-theme='dark'][data-month='12'] { --accent-…: …; --accent-…-ink: … }`, and only
  tokens change, so components stay untouched.
- **Contrast test:** loops over every theme × month and fails the build on any AA miss.
- **Opt-out:** stored like the existing theme preference (`localStorage`), applied by the same
  pre-paint script.
- **Motifs:** inline SVG, `aria-hidden`, and static when `prefers-reduced-motion` is set.

## 6. Next step

Discuss §4 together. Then I write the implementation plan (`D2`), and you or a designer provide the
palettes or mockups for the agreed scope.
