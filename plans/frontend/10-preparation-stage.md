# Step D1 — Preparation stage for 2027, then year navigation

> **Status:** Draft for approval (2026-11-11)
> **Parent:** [`backlog.md`](../backlog.md) → *Year rollover & multi-year history* (time-sensitive).
> **Builds on:** C3 (self-hosting). Part A should ship **before family is invited**.
> **Approval gate:** No code is written until you approve this document. Parts A and B can be
> approved separately.
> **Note:** This step touches Go, the SPA and the OpenAPI contract, so it runs both quality gates.

---

## 1. Objective

You described it this way: *a state of the website with a countdown to the New Year
(01.01.2027), where users can log in and create and edit their goals. After the New Year this
state is no longer needed, and the app works as designed.* Year switching comes on top.

- **Part A — Preparation stage** (must ship before invitations):
  - Until **1 January 2027, 00:00 in each user's own timezone**, the app is about **2027**.
  - The dashboard shows a **live countdown** to 01.01.2027 instead of the "year elapsed" bar.
  - Every big goal and milestone created now **belongs to 2027**.
  - At midnight the app **switches by itself** to the normal flow: year 2027, year bar,
    countdown to 2028. No redeploy is needed, and the setting is inert afterwards.
- **Part B — Year navigation** (December, or early January):
  - a **year switcher** on the dashboard and the Wins page;
  - `?year=` on `GET /dashboard`;
  - a list of the years you have goals in.

**Why this order:** with Part A, family's data is created in 2027 from day one, so nothing goes
blank for them on 1 January. Part B mainly serves your own 2026 test data and every future
rollover, which makes it important but not urgent.

### Explicitly NOT in this step
- **A recurring "plan next year" window every December.** It's one-off for 2027, as you described
  (§6 notes how it could be generalised).
- **A "year in review"** for finished years, or carrying unfinished goals into the next year
  (backlog, *fuller fix*).
- **Monthly themes** ([`11-monthly-themes.md`](11-monthly-themes.md)).

---

## 2. Current state (audited 2026-11-11)

- **The server decides "the year" in UTC** (`domain.CurrentYear()`), and that drives three things:
  - the default `year` of a new topic (`NewTopic`, when the request omits it);
  - the default scope of `GET /topics`;
  - the dashboard, which summarises the current year's topics and counts down to the *next* New
    Year (`domain.Countdown`, UTC).
- **The SPA never chooses a year itself:**
  - `useTopics()` and `useDashboard()` use the server defaults.
  - The year bar (`YearProgress`) and the logo ring (`yearElapsedFraction`) are computed from the
    browser clock.
  - The create popup's eyebrow says `New for ${new Date().getFullYear()}`.
  - The optimistic topic stub (`mutations.ts`) uses the same browser year.
- **Today (November 2026) everything is "2026".** A goal created now is a 2026 goal, and it
  disappears from the dashboard on 1 January.
- **Timezones:** a user in UTC+1/UTC+2 sees the server's year flip 1–2 hours late. The countdown
  target is UTC midnight, not local midnight.
- **What already exists:** `GET /topics?year=` exists, and `PATCH /topics/{id}` accepts `year`, so a
  topic can already be moved between years through the API. The dashboard has no `?year=`.

---

## 3. Design

### 3.1 The "season": one rule for which year the app is about

A new pure domain function decides it from three inputs: *now*, the user's timezone, and the
configured preparation year.

```go
type Season struct {
    Year      int       // the year the app is about
    Preparing bool      // true while Year hasn't started yet (the preparation stage)
    StartsAt  time.Time // local midnight, 1 Jan of Year
    EndsAt    time.Time // local midnight, 1 Jan of Year+1
}

func SeasonAt(now time.Time, loc *time.Location, preparationYear int) Season
// local := now.In(loc)
// if preparationYear > local.Year()  → {Year: preparationYear, Preparing: true}
// otherwise                          → {Year: local.Year(),    Preparing: false}
```

- **The countdown target is one rule:** `StartsAt` while preparing, otherwise `EndsAt`.
- **Today** (`PREPARATION_YEAR=2027`, Kyiv time) → `{2027, preparing, starts 2027-01-01 00:00 +02:00}`.
- **One second after local midnight** → `{2027, not preparing, ends 2028-01-01 00:00 +02:00}`. The
  stage is over for good; the setting no longer has any effect.

**What uses it:**
- the default year for new topics;
- the default scope of `GET /topics`;
- the dashboard.

The season is resolved **once per request** in the transport layer, like the client IP, and handed
to the services as a plain year. The services stay clock- and timezone-free.

### 3.2 Configuration (D1)

| Setting | Default | Meaning |
|---|---|---|
| `PREPARATION_YEAR` | empty (off) | e.g. `2027`: before 1 Jan 2027 (user's local time) the app is in the preparation stage for 2027. After that it's inert. |
| `APP_TIMEZONE` | `UTC` | Fallback timezone when a request doesn't say which one the user is in (D2). |

- **Validation:** `PREPARATION_YEAR` must be within 2000–2100, and `APP_TIMEZONE` must be a valid
  IANA name.
- **Timezone data:** the binary embeds the timezone database (`import _ "time/tzdata"`, about
  450 KB), because the distroless image has none.
- **On the Pi:** `.env.example` and the compose file gain `PREPARATION_YEAR=2027`, and the handbook
  gets a line saying it can be removed after New Year (or left; it does nothing then).

### 3.3 Timezone per request (D2)

- **The SPA sends the browser's IANA timezone** on every API call, as
  `X-Client-Timezone: Europe/Kyiv` (from `Intl.DateTimeFormat().resolvedOptions().timeZone`),
  added once in the `openapi-fetch` client.
- **The server validates it** with `time.LoadLocation`. If it's missing or invalid, it uses
  `APP_TIMEZONE`.
- **The countdown therefore hits zero at local midnight for everyone.** That matters if, for
  example, family are in UTC+2 and you're in UTC+1.
- **Spoofing the header only changes that user's own view**, so it isn't a security concern.

### 3.4 API changes (Part A)

- **`GET /dashboard`:**
  - gains `phase: "preparation" | "year"`;
  - `year` is the season year (2027 during preparation);
  - `countdown.target` is the season's countdown target (local midnight as an absolute instant);
  - `secondsRemaining` is derived from that target, as today.
- **`POST /topics` without `year`** → the season year. **`GET /topics` without `year`** → the
  season year. Explicit `year` values behave as today.
- **Dashboard message during preparation** comes from a new server-owned copy table, in
  `domain/dashboard.go`, next to the motivation copy, for one voice. Draft wording, yours to edit:
  - **No goals yet:** "Your 2027 is a blank page. Plan the first goal before the year begins."
  - **With goals:** "Every goal you plan now is a head start. 2027 is almost here."
- **OpenAPI:** documents the header (a reusable parameter) and `phase`; run `gen:api`.

### 3.5 The dashboard during preparation (Part A)

```
   Wednesday, 11 November
   ──
   Good evening, Yehor.
   Every goal you plan now is a head start. 2027 is almost here.

        ┌──────────────── 2027 begins in ────────────────┐
        │     51        07        42        18           │
        │    days      hours     min       sec           │
        └────────────────────────────────────────────────┘
          You're in the preparation stage · plan your year

   Chapters · 2027                                   + New
   ...  (the normal chapter list: create / edit / delete / milestones)
```

- **A `NewYearCountdown` component replaces `YearProgress`** while `phase === "preparation"`:
  days, hours, minutes and seconds, ticking every second.
  - It's computed locally from `countdown.target`, so it's smooth and needs no polling.
  - The digits are hidden from screen readers. A visually hidden text, updated **once a minute**,
    says e.g. "2027 begins in 51 days and 7 hours", so screen readers don't hear every second.
  - With `prefers-reduced-motion` it ticks without animation.
- **Everything else works as now** (D3): create, edit and delete big goals and milestones, mark
  milestones done, and Wins.
- **Wording follows the season year:**
  - the create popup's eyebrow becomes "New for 2027";
  - the empty state becomes "Your 2027 is a blank page…";
  - the chapters header says "2027".
- **The logo's year ring stays empty during preparation** (D5): the year hasn't started.
- **At midnight, with the page open** (D4):
  - the countdown reaches zero and shows a brief "Happy New Year — your 2027 starts now" moment;
  - it refetches the dashboard and topics;
  - the view switches to the normal year view without a reload.
- **Design:** the handoff has no mock for this state. It reuses the dashboard's existing type scale
  and the gold accent, and gets a Gallery entry (`/gallery`, dev only) so you can review it in both
  themes before it ships.

### 3.6 Part B — Year navigation

- **API:**
  - `GET /dashboard?year=YYYY` (2000–2100), default = season year.
  - New `GET /years` → `{ "season": 2027, "preparing": true, "years": [2026, 2027] }`: the years
    you have topics in, plus the season year (always present).
  - Both are user-scoped.
- **Switcher:** a compact year switcher (e.g. `‹ 2026 · 2027 ›` or a select) on the **dashboard**
  and the **Wins** page.
  - The selection lives in the URL (`/?year=2026`, `/achievements?year=2026`), so it survives
    reloads and can be linked.
  - Without `?year=` you get the season year.
- **Viewing a past year:**
  - The same layout, with no countdown. The year bar shows "2026 · complete", and the message
    reflects that year's final progress.
  - Editing stays allowed (people backfill), and new goals created while viewing 2026 go to 2026:
    the client sends `year` explicitly.
- **Moving a goal to another year** (D6): the edit-topic popup gains a "Year" select (season year
  ± 1, plus the topic's current year). That's how you'd move your own 2026 test goals into 2027.
  The API already supports it.

---

## 4. Decisions

| # | Decision | Status |
|---|---|---|
| D1 | **Turned on by setting**, `PREPARATION_YEAR=2027`, ending by itself at local midnight. *(Alt: hard-code 2027, which is simpler but needs a code change to reuse; or an automatic every-December window, which you said isn't needed.)* | Recommended |
| D2 | **Per-user timezone**: the browser sends its IANA zone, with `APP_TIMEZONE` as the fallback. *(Alt: one timezone for the whole site, which is simpler but makes the countdown 1–2 h off for family in another timezone.)* | Recommended |
| D3 | **Everything works during preparation**: create, edit, delete, and mark milestones done. *(Alt: plan only, with no completing until 1 January.)* | Recommended |
| D4 | **Midnight moment**: a short "Happy New Year" state, then a live switch without reload | Recommended |
| D5 | **Logo ring empty** during preparation *(alt: fill it as the countdown progresses through December)* | Recommended |
| D6 | **Part B includes a "Year" field** in the edit-topic popup, to move goals between years | Recommended |
| D7 | **Ship Part A first** (before invitations), Part B in December or early January | Recommended |

---

## 5. Files (created + / changed ~)

**Part A**
```
internal/domain/season.go (+test)        + Season, SeasonAt (pure; DST/timezone boundary tests)
internal/domain/dashboard.go (+test)     ~ preparation-stage copy
internal/config/config.go (+test)        ~ PREPARATION_YEAR, APP_TIMEZONE (+ validation)
cmd/api/main.go                          ~ wire season resolver; import _ "time/tzdata"
internal/transport/http/season.go (+test) + resolve the season once per request (header → context)
internal/transport/http/{topic,dashboard}_handler.go, dashboard_dto.go ~ season year + phase
internal/service/{topic,dashboard}_service.go (+tests) ~ take the year/season explicitly
api/openapi.yaml · web/src/api/schema.d.ts ~ X-Client-Timezone, Dashboard.phase
web/src/api/client.ts                     ~ send X-Client-Timezone on every request
web/src/features/dashboard/NewYearCountdown.tsx (+test) + the countdown
web/src/features/dashboard/Dashboard.tsx, layout/AppShell.tsx, topics/TopicFormModal.tsx,
  topics/EmptyState.tsx, topics/mutations.ts, lib/year.ts ~ season-aware wording/year
web/src/routes/Gallery.tsx · test/a11y.test.tsx ~ preparation-stage examples + axe
deployments/pi/{docker-compose.yml,.env.example} · docs/operations.md ~ PREPARATION_YEAR
docs/dev_run.md · CLAUDE.md · plans/backlog.md ~
```

**Part B**
```
internal/service + transport: GET /years, GET /dashboard?year= (+ tests, integration)
web: YearSwitcher component; ?year= on Dashboard + Wins; Year field in the edit popup (+ tests, axe)
api/openapi.yaml · schema.d.ts · docs
```

---

## 6. Verification

- **Domain tests at the boundaries:**
  - 31 Dec 23:59:59 vs 1 Jan 00:00:00 in UTC, UTC+2, UTC−5 and UTC+14;
  - `PREPARATION_YEAR` unset, past, or equal to the current year (all inert);
  - an invalid timezone falls back to `APP_TIMEZONE`.
- **Server-side checks:**
  - Service and handler tests: default topic year = season year, explicit year untouched, and
    the dashboard's phase, target and copy.
  - Integration tests over HTTP: the timezone header changes the countdown target, and a topic
    created during preparation is a 2027 topic.
- **Web:**
  - countdown rendering and ticking with fake timers;
  - the midnight switch;
  - wording follows the season year;
  - the screen-reader text updates once a minute;
  - axe;
  - the full gate.
- **Manual:** run locally with `PREPARATION_YEAR` set to next year, check both themes and mobile,
  then fake midnight with a short countdown in the dev tools.
- **A note for later generalisation:** a recurring December planning window would be `SeasonAt`
  with a "preparation starts on" date instead of a fixed year. That's a small change if you ever
  want it.
- Docs updated. **Nothing committed.**

---

## 7. Timeline

| When | What |
|---|---|
| Now | Approve Part A → implement → PR → CI → the Pi updates overnight. Set `PREPARATION_YEAR=2027` in the Pi's `.env`. |
| Then | Invite family: they plan 2027 with the countdown in front of them. |
| December | Part B (year switcher), or early January at the latest. |
| 1 Jan 2027, 00:00 local | The app switches to the normal 2027 flow by itself. |
