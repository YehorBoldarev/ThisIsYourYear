# Handoff: "This is Your Year" — Goal-tracking SPA

## Overview
"This is Your Year" is a single-page web app for setting and tracking annual goals. A user
creates **big goals** (called *chapters* / *topics*) and breaks each into **milestones**
(sub-goals). Completing a milestone can be marked as an **achievement** (★). Each goal shows a
**motivational message that changes with completion state**, and the dashboard shows an overall
**progress-through-the-year bar** and a **countdown to the New Year**.

The app ships in **two color themes** — a warm **dark** ("journal at night") theme and a warm
**light** ("paper") theme — switchable from a control in the header. Layouts are responsive:
**desktop (1320px)**, **tablet (768px)**, **mobile (390px)**.

## About the Design Files
The files in this bundle are **design references created in HTML** — prototypes that show the
intended look, layout, and behavior. They are **not production code to copy directly**. They use a
small in-house templating runtime (`support.js`, `<x-dc>`, `{{ }}` holes, `<sc-for>`/`<sc-if>`)
purely so the mock could render sample data — **ignore that runtime**. Your task is to **recreate
these designs in the target codebase's environment** (React, Vue, Svelte, SwiftUI, etc.) using its
established components, routing, and state patterns. If no codebase exists yet, choose the most
appropriate SPA framework (React + Vite is a safe default) and implement there.

The visual layout in each file lives inside positioned `<div>` "frames" on a design canvas. Each
frame is one screen/state at one breakpoint. Read the markup inside each frame for exact structure;
this README captures every value you need.

## Fidelity
**High-fidelity (hifi).** Colors, typography, spacing, radii, and shadows are final. Recreate the UI
pixel-accurately using the codebase's libraries. The sample copy and data (user "Nostradamus
Affterfluchell", the six goals, the dates) are **placeholder content** — wire them to real data.

## Files
- `This Is Your Year.dc.html` — **dark theme**. Contains: Editorial-Journal dashboard (desktop),
  Topic-detail popup, Create-big-goal popup, Add-milestone popup, Achievement celebration,
  Motivational-message state reference, Empty state, Tablet 768px, Mobile 390px.
  (Two earlier alternate dashboard layouts — "horizontal rail" and "sidebar+grid" — were explored
  then removed by the client; **the Editorial-Journal layout is the chosen direction**.)
- `This Is Your Year - Light.dc.html` — **light theme**, same screens as the dark file (dashboard,
  the four popups/states, empty state, tablet, mobile).

Open either file in a browser to view all frames on a pannable canvas. The small uppercase label
above each frame (e.g. "03 · Dashboard — Editorial journal") names the screen.

---

## Design Tokens

### Typography
Three Google Fonts:
- **Newsreader** (serif) — headings, goal titles, big numbers, motivational quotes (often *italic*).
  Weights 400/500/600; italic 400/500.
- **Hanken Grotesk** (sans) — body text, buttons, form inputs, avatar initials. Weights 400–700.
- **Spline Sans Mono** (mono) — small uppercase labels, meta text, footer, progress captions.
  Weights 400/500. Labels use `text-transform:uppercase` + `letter-spacing:.08em–.2em`.

Representative sizes (desktop): page H1 36px/1.22 (Newsreader 500); section H2 20–24px (600); goal
title (row) 19px (600); goal title (popup) 26–30px (600); body 14–15px; meta/labels 11px mono;
big stat numbers 22–34px (Newsreader 600).

### Color — Dark theme
| Role | Hex |
|---|---|
| App background (canvas behind cards) | `#0c0b08` / radial to `#1c1a14` |
| Card / panel surface | `#131210` |
| Header & footer bar | `#16140f` |
| Elevated popup surface | `#1a1813` |
| Inset input background | `#16140f` |
| Muted fill (nav pill, toggle track) | `#26231d` |
| Border (primary) | `#2a2722` |
| Border (subtle divider) | `#232019` |
| Border (on popups) | `#34302a` |
| Text primary | `#ece8e1` |
| Text secondary | `#b3ada3` |
| Text muted | `#8a847a` / `#7d776c` |
| Text faint / label | `#55504a` / `#6b6459` |
| Accent — teal (primary/CTA) | `#3ec5a0` |
| Accent — blue | `#5b8def` |
| Accent — gold (achievement/star) | `#e8b765` |
| Accent — rose | `#c2738f` |
| Progress track | `#2a2722` |

### Color — Light theme
| Role | Hex |
|---|---|
| App background (canvas behind cards) | `#e3d9c8` / radial to `#efe7da` |
| Page/card background (paper) | `#faf6ee` |
| Elevated surface / white card | `#ffffff` |
| Header & footer bar | `#ffffff` |
| Inset input background | `#faf6ee` |
| Muted fill (nav pill, toggle track) | `#f0e9dc` / `#ece4d6` |
| Border (primary) | `#e7dfd2` |
| Border (subtle divider) | `#ece4d6` / `#f1ece3` |
| Border (input) | `#ddd3c4` |
| Text primary | `#2b2620` |
| Text secondary | `#6e665a` |
| Text muted | `#9c9384` |
| Text faint / label | `#9a9184` / `#b7ad9c` |
| Accent — teal (primary/CTA) | `#2f9e7e` |
| Accent — blue | `#3f6fd6` |
| Accent — gold (achievement/star) | `#d29b3f` (deep gold text `#b8841f`) |
| Accent — rose | `#c2738f` |
| Progress track | `#ece4d6` |

> Themes are the **same layout** with swapped tokens. Implement as CSS variables / a theme object
> and toggle a `data-theme` attribute (or context) — do not fork the markup.

### Spacing, radius, shadow
- Card radius **18px**; inner popup card **18–20px**; small cards/inputs **10–14px**; pills/toggles
  **999px**; progress bars **999px**.
- Card padding ~**26–32px**; header padding **15px 28px**; input padding **12px 14px**.
- Shadows: dark cards `0 16px 50px rgba(0,0,0,.45)`; dark popups `0 30px 80px rgba(0,0,0,.6)`;
  light cards `0 16px 50px rgba(120,100,70,.13)`; light popups `0 30px 80px rgba(90,72,40,.22)`.
- Progress bar height: **8px** (dashboard year bar), **6px** (per-goal bars).

### Logo
Star + timer mark: an SVG circle acting as a **timer ring** (a partial teal stroke over a faint
full ring, `stroke-linecap:round`, rotated -90°) with a **gold 5-point star** centered inside.
This encodes "achievement earned through the year." Recreate as an SVG component; the ring's
`stroke-dasharray` can be bound to overall year progress. Wordmark "This is Your Year" in Newsreader
600, 19px.

---

## Screens / Views

### 1. Dashboard — Editorial Journal (default, desktop)
**Purpose:** home overview; read the day's greeting, see year progress, scan all goals, open any.
**Layout:** full-width card. Sticky-style **header** (logo + nav left; theme toggle + user identity
+ avatar right). Centered **editorial column, max-width 680px**: date eyebrow (mono, centered) → thin
rule → large greeting H1 (centered) → italic sub-message → **year progress bar** with JAN / "YOU ARE
HERE · 48%" / DEC captions → section row "The chapters" + "＋ Begin a new chapter" link → a
**list of goal rows**, each a flex row: colored 3px accent bar | title + `★ ACHIEVED` tag + mono
"status · done/total milestones" | right side percent (Newsreader) + 96px mini progress bar.
Full-width **footer**: support email · GitHub link · "Since 2026".
**Components / values:**
- Header nav tabs: `Dashboard` (active — filled pill), `Achievements`, `Reflections`.
- User block: name **"Nostradamus Affterfluchell"** (13px 600) + role "Dreamer-in-Chief" (11px muted).
- Avatar: 38px circle, gradient `135deg` teal→blue, initials "NA" 700.
- Year bar: 48% filled, gradient teal→gold. Caption center text in gold.
- Goal rows use the six sample goals (see **Sample data** below).

### 2. Header theme switch
A **segmented sun/moon toggle**: rounded-999px track (muted fill), two 26px circular options.
Active option has a raised/filled background (dark: `#3a352d` bg gold ☾ icon; light: white bg with
soft shadow, gold ☀ icon). Clicking flips the whole app theme. On tablet options are 24px, on mobile
22px. Place it left of the user identity block in the header (mobile: left of the avatar).

### 3. Topic (big-goal) detail — popup
**Purpose:** view one big goal, its motivational message, and its milestone checklist; mark
milestones/goal done.
**Layout:** centered modal card (width ~720px) over a dimmed radial backdrop; internal sections
divided by hairline borders.
- Header block: mono category eyebrow ("Big goal · Endurance") + ✕ close button (30px, bordered);
  goal title H2 (30px); one-line description; then a **motivational callout** — left accent border,
  tinted background (teal @ ~8–10% alpha), italic Newsreader quote.
- Progress block: 84px **ring** (accent stroke, `stroke-dasharray` = percent, centered % label) +
  "6 of 8 milestones complete" + mono "Almost there · 2 milestones to go".
- Milestones: label row "Milestones" + "＋ Add milestone" link, then a `<ul>` of rows. Each row =
  status token (22px circle) + label + trailing tag/date:
  - **Achievement done:** gold-outlined circle with ★, label struck-through, mono `ACHIEVED` tag (gold).
  - **Normal done:** solid teal/green circle with ✓ (white check), label struck-through.
  - **Open:** empty outlined circle, primary-text label, mono due date (e.g. "Due Oct 4").
- Footer actions (right-aligned): `Edit goal` (outline button) + `★ Mark goal achieved` (gold solid).

### 4. Create a big goal — popup
**Purpose:** create a new top-level goal.
**Layout:** modal card ~560px. Eyebrow "New for 2026" + H2 "Create a big goal" + ✕.
Fields: **What is the goal?** (text input) · **Why does it matter?** (textarea) · row of **Target date**
+ **Category** (two inputs) · **Accent colour** (four 28px swatch circles: teal, blue, gold, rose;
selected has a double-ring). Footer: `Cancel` (outline) + `Create goal` (teal solid).
Inputs: inset background, 1px border, 10px radius, 12×14 padding; labels are 11px mono uppercase.

### 5. Add a milestone (sub-goal) — popup
**Purpose:** add a milestone under a specific big goal.
**Layout:** modal card ~520px. Eyebrow shows parent goal with its accent dot ("● Run a
Half-Marathon") + H2 "Add a milestone" + ✕. Fields: **Milestone** (text) · **Target date — optional**
· **Notes — optional** (textarea) · a **toggle row** "Celebrate as an achievement ★" (label + helper
text + on/off switch, switch shown ON = accent track, white knob right). Footer: `Cancel` +
`Add milestone` (teal solid).

### 6. Achievement unlocked — celebration state (100%)
**Purpose:** reward completing a goal.
**Layout:** centered card ~480px on a gold-tinted radial backdrop sprinkled with small confetti
dots/squares (teal/blue/gold/rose, rotated). Card: 78px gold **star-in-glow** medallion → mono gold
eyebrow "ACHIEVEMENT UNLOCKED" → goal title H2 → italic congratulatory quote → a 3-up stat strip
(e.g. `6/6` milestones · `5 mo` to complete · `#1` of 2026) separated by a top border → two buttons:
`Reflect on it` (outline) + `Keep the momentum →` (gold solid).

### 7. Motivational messages — state reference
**Purpose:** documents the four message states so behavior is unambiguous. **This is a spec board,
not a runtime screen** — implement its content as the logic in screens 1 & 3.
Four cards, each = mono state label (colored by tone) + a mini progress bar + a Newsreader title +
an italic message:
- **0% · Just started** (tone muted/grey) — *"The blank page"* — "A blank page isn't empty — it's
  full of everything that could be. Make the first mark today."
- **In progress** (tone blue) — *"Building momentum"* — "Momentum is quiet. Keep showing up, and the
  small days compound into a remarkable year."
- **Almost there** (tone teal) — *"In sight"* — "You can see it from here. Don't trade the finish
  line for comfort — finish what you started."
- **Achieved** (tone gold) — *"A promise kept"* — "You kept a promise to yourself. This is exactly
  what that feels like. Take a moment to savour it."

### 8. Empty state — no goals yet
**Purpose:** first-run / zero goals.
**Layout:** standard header + footer; centered column: faint large logo mark (ring nearly empty) →
H1 "Your year is a blank page." → supporting paragraph → primary CTA `＋ Create your first big goal`
→ a "TRY:" row of example goal chips ("Run a marathon", "Read 24 books", "Learn a language").

### 9. Tablet — 768px
The Editorial-Journal dashboard inside browser chrome. Same structure, tightened: logo 27px, nav
padding reduced, greeting H1 30px, editorial column padded 30×36. Nav stays inline in the header.

### 10. Mobile — 390px
The dashboard inside a narrow browser frame. Header collapses the top nav; greeting H1 25px (line
break after "Good evening,"). **Each goal becomes a stacked card**: left 3px accent border, title +
★ + percent on one line, mono status line, then a **full-width** 6px progress bar. A fixed **bottom
tab bar** replaces top nav: **Goals ▦** (active, accent) · **Wins ★** · **Notes ✎** · **New ＋**.

---

## Interactions & Behavior
- **Open goal:** clicking a goal row/card opens the Topic-detail popup (screen 3) for that goal.
- **New big goal:** "＋ Begin a new chapter" / "＋ New big goal" / empty-state CTA opens Create-goal
  popup (screen 4).
- **Add milestone:** "＋ Add milestone" inside a topic opens Add-milestone popup (screen 5).
- **Complete milestone:** toggling a milestone's status circle marks it done (✓) or, if flagged as
  an achievement, ★. When **all** milestones of a goal are done → goal reaches 100% → show the
  Achievement celebration (screen 6).
- **Motivational message** on a goal is derived from its completion ratio, using the four bands in
  screen 7: 0% → "just started"; >0 and <~80% → "in progress"; ~80–99% → "almost there"; 100% →
  "achieved". (Pick exact thresholds during implementation; ~80% used for "almost there" in the mock.)
- **Theme toggle** flips light/dark for the whole app; persist the choice (e.g. localStorage) and
  respect it on reload.
- **Popups:** open over a dimmed backdrop; ✕ or Cancel closes; primary action saves. Backdrop click
  and Esc should also close (standard modal behavior).
- **Countdown / year progress:** the dashboard bar shows year elapsed (mock: Day 175/365 ≈ 48%) and
  days remaining to the New Year ("YOU ARE HERE"). Compute from the current date.
- **Hover states:** cards lighten their border on hover; CTA buttons lighten their fill on hover
  (teal `#3ec5a0`→`#4fd3ae` dark / `#2f9e7e`→`#36b08c` light; gold `#e8b765`→`#f0c47e` dark /
  `#d29b3f`→`#dcab57` light). Nav/links are pointer-cursor.

## State Management
Suggested state:
- `theme`: `'light' | 'dark'` (persisted).
- `user`: `{ name, role, initials }`.
- `goals[]`: each `{ id, title, description, why, category, accentColor, targetDate,
  milestones[] }`.
- `milestone`: `{ id, title, dueDate, notes, done: boolean, isAchievement: boolean }`.
- Derived per goal: `done`/`total` counts, `percent`, `status` label, `motivationState`,
  `isAchieved` (percent === 100).
- Derived app-wide: year progress %, days-to-New-Year (from current date), counts for active goals /
  achievements / milestones done.
- UI: which popup is open and its target goal/milestone.
- Data fetching: goals load from and persist to the backend/store (out of scope for the mock).

## Responsive Behavior
Three breakpoints demonstrated: **≥ ~1024px** desktop (centered 680px editorial column), **768px**
tablet (nav stays inline, sizes tightened), **≤ ~430px** mobile (top nav → bottom tab bar; goals
become stacked cards with full-width bars; reduced type scale). Popups should become near-full-width
sheets on mobile.

## Assets
- **Fonts:** Newsreader, Hanken Grotesk, Spline Sans Mono (Google Fonts) — load via the codebase's
  font pipeline.
- **Logo:** inline SVG (star + timer ring) — recreate as a component; no external file.
- **Icons:** the mock uses text glyphs (☀ ☾ ✓ ★ ✕ ＋ ▦ ✎ ●). Replace with the codebase's icon set
  (e.g. a check, star, x, plus, grid, pencil). No raster image assets are required.
- **Avatar:** gradient circle with initials — no image; swap for real user photo when available.
