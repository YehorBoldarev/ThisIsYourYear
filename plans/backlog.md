# Backlog — postponed features

Deliberately deferred work. Each item still needs its own approved plan (`plans/` or
`plans/frontend/`) before implementation starts.

## Postponed features

### Year rollover & multi-year history
> Added 2026-11-07. **Time-sensitive:** affects every user on 1 January 2027.
> **Now planned (2026-11-11):** [`frontend/10-preparation-stage.md`](frontend/10-preparation-stage.md)
> — a pre-launch "preparation stage" for 2027 plus the year switcher.

- **Problem.** The dashboard, chapter list and Wins page only show the current year, and the UI has
  no year switcher. `GET /topics` accepts `?year=`, but `GET /dashboard` doesn't. When the year
  rolls over, every user's dashboard goes blank and their previous year's goals and wins become
  unreachable in the app (the data stays in MongoDB).
- **Minimal fix.** A year switcher on the dashboard and Wins page, plus `?year=` on
  `GET /dashboard`.
- **Fuller fix.** A "year in review" for the finished year, and carrying unfinished topics into the
  new year.
- **Related.** The server computes the current year and the countdown target in UTC, so for a user
  in UTC+3 the countdown hits zero at 03:00 local and the year flips at the wrong hour. Fix
  alongside (the client sends its timezone or year).

### Reflections / Notes
- A new backend domain plus UI. The nav item ("Reflections" in the header, "Notes" in the mobile
  tab bar) currently leads to a "Coming soon" placeholder.
- Building it restores the celebration popup's "Reflect on it" action (removed 2026-11-07 because
  it led to the placeholder).

### Milestone reordering in the UI
- The API already supports `order`; the UI renders milestones in API order with no drag-and-drop
  (deferred in `05-goals-and-achievements.md` and `frontend/04-topics-goals.md` D4).

### Wins endpoint
- The Wins page fans out one `GET /topics/{id}/goals` per topic. Fine for a handful of topics,
  slow for dozens; a dedicated endpoint would make it one request.

### Deep links to topic popups
- Popups are local state, so a topic detail can't be linked or restored on reload (URL sync
  deferred in `frontend/04-topics-goals.md` D3).

### Self-service password change
- Deferred from C3 (D5). Today the operator resets a password with `api admin reset-password`, and
  the person keeps the temporary password. Needs `POST /me/password` (current + new) and a small
  account-settings form.

### Admin command to delete an account
- `docs/operations.md` §7.5 removes an account with a manual `mongosh` script; an
  `api admin delete-user --email X` command (with confirmation) would be safer.

### Uptime monitoring
- The site has a permanent address now (Tailscale Funnel), so an external check of
  `https://<machine>.<tailnet>.ts.net/healthz` (a free uptime service, or a phone notification)
  would tell you about an outage before family does.

### Off-site backups
- Backups go to the Pi (or a USB drive) and are copied to a computer by hand monthly. An automatic
  encrypted copy elsewhere (e.g. `rclone` to cloud storage) would remove the manual step.

### Custom domain
- Funnel only serves `*.ts.net`. For your own domain, use a Cloudflare Tunnel (the first C3 draft)
  in front of the same kit. The client-IP mechanism is header-configurable
  (`CLIENT_IP_HEADER=CF-Connecting-IP`).

### Monthly themes
- Planned for discussion: [`frontend/11-monthly-themes.md`](frontend/11-monthly-themes.md).

### End-to-end tests (optional)
- A few Playwright golden-path flows (sign up → create a goal → complete it → celebration),
  flagged as optional in `frontend/00-system-plan.md` §3.

## Known small issues

- **Footer placeholder links** (`web/src/layout/Footer.tsx`): GitHub points to
  `github.com/kibou/thisisyouryear` (the repo is `YehorBoldarev/ThisIsYourYear`), and
  `hello@thisisyour.year` isn't a deliverable address (`.year` isn't a real TLD).
- **Reflections nav dead end:** decide whether to hide the nav item until Reflections exists or
  keep it as a deliberate teaser.
