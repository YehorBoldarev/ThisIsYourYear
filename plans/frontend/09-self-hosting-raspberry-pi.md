# Step C3 — Self-hosting on a Raspberry Pi 5

> **Status:** Draft for approval (2026-11-08)
> **Parent:** Phase C (production). Follows [`08-ci-cd-pipeline.md`](08-ci-cd-pipeline.md) (C2).
> **Depends on:** C2 implemented and pushed: the multi-arch image (C2 D7) is on GHCR and the package
> is public. (Until then, the Pi can build the image itself; see §7.)
> **Approval gate:** No files are written until you approve this document.
> **Note:** This step touches Go, the SPA, the OpenAPI contract and deployment files, so it runs the
> **Go and web** quality gates plus checks for the shell scripts and compose files.

---

## 1. Objective

Run the production app on your Raspberry Pi 5 for family and friends: safely, with automatic
backups and updates, and with an admin handbook that tells you what to check, when, and how.

It ships in **two phases**. Only phase 1 is executed in this step; phase 2 is written down so it's a
~30-minute switch once you have a domain:

| | How people reach it | Cookie | When |
|---|---|---|---|
| **Phase 1 — no domain** | `http://<pi-ip>:8080` on the home network, plus an **optional Quick Tunnel** (random `https://….trycloudflare.com`, changes on restart) for testing from outside | `COOKIE_SECURE=false` (plain HTTP at home) | Now: testing, a few trial users |
| **Phase 2 — domain** | `https://year.<your-domain>` through a **permanent Cloudflare Tunnel** | `COOKIE_SECURE=true` | When you buy/obtain a domain |

### In scope
1. **App changes**
   - **Real client IP behind the tunnel** (§3.3) so login rate-limiting is per visitor, not global.
   - **Invite-only registration** (§3.4) with an invite code and a shareable invite link.
   - **Admin commands** in the same binary (§3.5): `healthcheck`, `admin reset-password`,
     `admin list-users`.
2. **Pi deployment kit** `deployments/pi/` (§3.6): compose file, `.env` template, backup / restore /
   update scripts, and systemd timers that run backups and updates on schedule.
3. **Admin handbook** `docs/operations.md` (§3.8): install, maintenance schedule, logs for every
   component, backup & restore, updates & rollback, user admin, troubleshooting, phase-2 switch.
4. **Docs:** `docs/dev_run.md` (changelog + new settings), `README.md`, `CLAUDE.md`,
   `plans/backlog.md`.

### Explicitly NOT in this step
- **Buying or configuring a domain.** Phase 2 is documented only.
- **Self-service password change / "forgot password" e-mails** (D5 → backlog).
- **Remote admin from outside home.** SSH stays LAN-only.
- **Uptime monitoring / alerting services.** These only make sense with a permanent address
  (phase 2), so they go to the backlog.
- **The year-rollover fix.** It stays in the backlog, but see the risk in §8: it's due before
  1 January 2027.

---

## 2. Current state (audited 2026-11-08)

- **The Pi:** Raspberry Pi 5 (arm64), Raspberry Pi OS 64-bit.
  - Static IP; SSH works.
  - `ufw` allows SSH from the home LAN only.
  - Docker Engine + Compose installed, with log rotation.
  - `mongo:7.0.45` passed the smoke test (start, 20k inserts, restart).
  - No domain yet.
- **Image:**
  - The `Dockerfile` builds natively on arm64 as-is.
  - C2 adds the published multi-arch image `ghcr.io/yehorboldarev/thisisyouryear`.
- **Client IP:** `clientIP()` (`auth_handler.go`) uses `r.RemoteAddr` only, which is deliberate (see
  the RealIP note in `router.go`). Behind `cloudflared`, every visitor arrives from the tunnel
  container's address, which has three effects:
  - The auth rate limit (20 req/min) becomes **one shared bucket for everyone**: one person mistyping
    a password repeatedly locks out the whole family.
  - Session records show the tunnel's IP.
  - The request logs show the tunnel's IP.
- **Registration:** open to anyone who can reach the site.
- **Account recovery:** there is no password reset of any kind and no change-password endpoint.
  `SessionRepository.RevokeAllForUser` already exists.
- **Existing compose files don't fit real users.**
  - `deployments/docker-compose.yml` is for dev.
  - `docker-compose.customer.yml` (the local demo) has four problems:
    - it runs in **development mode**;
    - it uses the **dev JWT secret that's published in the repo**;
    - it publishes `8080` on all interfaces;
    - it uses floating `mongo:7`.
- **Docker vs firewall:** Docker-published ports **bypass `ufw`**.
- **Plain HTTP at home works:**
  - Production config doesn't reject `COOKIE_SECURE=false`.
  - The CSP has no `upgrade-insecure-requests`, and the server sends no HSTS.
- **Logs:** JSON in production (`internal/platform/logging`).

---

## 3. Design

### 3.1 Topology

```
Phase 1
  home device ──http──► 192.168.1.50:8080 ──┐   (published on the Pi's LAN IP only)
                                            ├──► api  (Go: SPA + API) ──► mongo (volume; no ports)
  internet ──https──► Cloudflare ◄── tunnel-quick (optional; outbound-only) ──┘
                                        └─ Docker network "tiyy" (fixed subnet, fixed tunnel IPs)

Phase 2
  internet ──https──► Cloudflare ◄── tunnel (named, token) ──► api ──► mongo
  Pi itself ──http──► 127.0.0.1:8080 (admin/health checks; or from your laptop via `ssh -L`)
```

- **No inbound ports are opened on the router** in either phase. The tunnel connects outward.
- **MongoDB is never published.** Only the `api` container can reach it.

### 3.2 Services (`deployments/pi/docker-compose.yml`)

| Service | Image | Notes |
|---|---|---|
| `mongo` | `${MONGO_IMAGE}`, default `mongo:7.0.45` (pinned) | Named volume and healthcheck. WiredTiger cache capped (default 1 GB) so Mongo can't take half the Pi's RAM. No `ports:`. |
| `api` | `${APP_IMAGE}`, default `ghcr.io/yehorboldarev/thisisyouryear:latest` | Settings come from `.env`, with `APP_ENV=production` forced. Waits for `mongo` to be healthy. Healthcheck via `/app/api healthcheck` (distroless has no curl). `ports: "${PUBLISH_ADDR}:8080:8080"`. |
| `tunnel-quick` | `${CLOUDFLARED_IMAGE}` (pinned) | Profile `quick-tunnel`. Runs `tunnel --no-autoupdate --url http://api:8080`. Fixed IP on `tiyy`. |
| `tunnel` | `${CLOUDFLARED_IMAGE}` (pinned) | Profile `tunnel` (phase 2). Runs `tunnel --no-autoupdate run` with `TUNNEL_TOKEN`. Fixed IP on `tiyy`. |

- **All services** use `restart: unless-stopped`, so the site comes back after a power cut or reboot.
- **Choosing the front door:** you pick it with **one line** in `.env`: `COMPOSE_PROFILES=` (home
  network only), `quick-tunnel`, or `tunnel`. Docker Compose reads this from `.env` natively.
- **`PUBLISH_ADDR`:**
  - It defaults to the Pi's LAN IP in phase 1, so the port isn't bound on IPv6 or other interfaces.
  - It's `127.0.0.1` in phase 2.
  - The handbook explains why this matters: Docker bypasses `ufw`.

### 3.3 Real client IP behind the tunnel

**New settings** (both default to empty or off, which means today's behaviour exactly):
- `TRUSTED_PROXIES`: a comma-separated list of IPs/CIDRs allowed to tell us the client's IP.
- `CLIENT_IP_HEADER`, default `CF-Connecting-IP`. Only single-value headers are supported (for
  example `CF-Connecting-IP`, `X-Real-IP`); `X-Forwarded-For` chains are deliberately not parsed.

**Behaviour:** one middleware, first in the chain, resolves the client IP once.
- **Trusted:** if the connection comes from an address in `TRUSTED_PROXIES` and the header holds a
  valid IP, that header value is the client IP.
- **Otherwise:** the client IP is the connection's own address.

The result goes into the request context. These all use it:
- the auth rate limiter (one bucket per real visitor);
- the IP recorded on sessions;
- the request log, which logs both `client_ip` and the raw `remote_ip`.

**Why this is safe:** the Pi compose file pins the tunnel containers to fixed addresses, and
`TRUSTED_PROXIES` lists exactly those addresses.
- **Home devices** reach the app through the published port with their own LAN address, which is
  not trusted. So nobody on your Wi-Fi can fake the header to dodge the rate limit.
- **The `router.go` comment** about `middleware.RealIP` gets updated to point to this vetted
  mechanism.

### 3.4 Invite-only registration (D2)

- **Setting:**
  - New setting `REGISTRATION_INVITE_CODE`. Empty means open registration, which is today's
    behaviour and the dev default.
  - When set, it must be at least 8 characters, and every registration must present it.
  - It's compared in constant time. The existing per-IP auth rate limit makes guessing impractical.
- **API changes** (`api/openapi.yaml`, then `npm run gen:api`):
  - `POST /api/v1/auth/register` accepts an optional `inviteCode`. If it's missing or wrong, the
    response is the standard `422` validation envelope with a field error on `inviteCode`, so the
    form shows it under the field like any other error.
  - New public endpoint `GET /api/v1/auth/registration` returns `{ "inviteRequired": true|false }`,
    so the sign-up page shows the field only when needed.
- **Where the check lives:** in `AuthService.Register`, which receives the configured code through
  its constructor, like the other auth settings.
- **Sign-up page:**
  - Gains an **"Invite code"** field when `inviteRequired` is true.
  - The field is **prefilled from the link**, so you can share
    `https://<address>/register?invite=<code>` and family only fill in name, e-mail and password.
- **Changing or closing:**
  - **Change the code:** edit `.env`, then `docker compose up -d`. Existing accounts are unaffected.
  - **Close sign-up completely:** set a long random code that you share with nobody.

### 3.5 Admin commands (same binary, no extra image)

Run on the Pi with `docker compose exec api /app/api <command>`; distroless needs no shell for this.

| Command | What it does |
|---|---|
| `healthcheck` | Calls `http://localhost:8080/healthz` and exits 0 or 1. Used by the compose healthcheck and by `update.sh`. |
| `admin list-users` | Prints e-mail, display name and sign-up date for every account, so you can see who joined. |
| `admin reset-password --email X` | Generates a readable temporary password (e.g. four words), sets it and **signs that user out everywhere** (`RevokeAllForUser`). Prints the password **once** for you to pass on. |

- **New repository methods:** `UserRepository` gains `SetPasswordHash` and `List`.
- **Where the logic lives:** the reset logic lives in `AuthService`, so the hashing and session
  rules stay in one place. `cmd/api/main.go` gains a small subcommand switch, and no arguments
  starts the server exactly as today.

### 3.6 Pi deployment kit

```
deployments/pi/
  docker-compose.yml        §3.2
  .env.example              every setting with a one-line comment; the real .env exists only on
                            the Pi (git-ignored, chmod 600)
  backup.sh                 mongodump --archive --gzip → $BACKUP_DIR/tiyy-YYYYMMDD-HHMM.archive.gz;
                            checks the archive isn't empty; deletes archives beyond $BACKUP_KEEP (14)
  restore.sh <archive>      asks for confirmation, takes a safety backup first, then
                            mongorestore --drop
  update.sh                 backup → pull app image → up -d → wait up to 2 min for healthy;
                            if not healthy, roll back to the previous image and exit non-zero
  systemd/
    tiyy-backup.service/.timer   daily 03:30
    tiyy-update.service/.timer   daily 04:00 (D3)
```

- **Install location:** `/opt/thisisyouryear` is a **git clone** of the public repo (D7). You run
  everything from `deployments/pi/`. Scripts change only when *you* run `git pull`; the timers never
  pull repo code.
- **What updates automatically:**
  - **The app image** updates automatically (D3).
  - **Mongo and cloudflared** stay pinned in `.env` and are bumped by hand, following the handbook's
    monthly checklist.
  - **OS security updates** come from Debian's `unattended-upgrades`; setup steps are in the
    handbook.
- **Scripts quality bar:**
  - `set -euo pipefail` and readable output.
  - They log to the journal, so `journalctl -u tiyy-update` shows every update and rollback.

### 3.7 Secrets

All secrets live only in `/opt/thisisyouryear/deployments/pi/.env` (`chmod 600`, git-ignored):
- `JWT_SECRET` (generated on the Pi with `openssl rand -hex 32`);
- `REGISTRATION_INVITE_CODE`;
- `TUNNEL_TOKEN` (phase 2).

None of them goes in the repo, the workflow or a chat message. The handbook explains how to rotate
each one and what rotating does. For example, a new `JWT_SECRET` means everyone's next page load
silently refreshes; it doesn't lose any data.

### 3.8 Admin handbook — `docs/operations.md`

The handbook is written last, against the finished kit, so every command in it is real.

- **Map:** what runs where (the diagram from §3.1, in plain words).
- **Install:** from a fresh Pi to running site (clone, `.env`, first start, enable timers,
  `unattended-upgrades`), including the phase-1 Quick Tunnel and how to find its current address.
- **Maintenance schedule**

  | When | What |
  |---|---|
  | Weekly | Glance at the timers and last update result; check disk space. |
  | Monthly | Copy a backup off the Pi. Bump Mongo/cloudflared if new patch versions exist. Check temperature and throttling. Run `apt` upgrades if anything is held back. |
  | Every 3 months | **Restore drill:** restore the latest backup into a scratch container. |
  | Yearly | Review accounts and the invite code; rotate secrets if needed. |
- **Logs, per component:**
  - app (`docker compose logs`, with filters for errors and 5xx in the JSON logs);
  - MongoDB;
  - tunnel;
  - backup and update timers (`journalctl -u …`);
  - Docker daemon;
  - SSH logins;
  - firewall (`ufw` and kernel log);
  - system health (`vcgencmd measure_temp`, `vcgencmd get_throttled`, `df -h`, `docker system df`,
    `free -h`).
- **Backups & restore:**
  - where the backups are;
  - how to copy one to your computer;
  - restoring after a mistake;
  - rebuilding on a new SD card or SSD.
- **Updates & rollback:**
  - how automatic updates work;
  - how to pin a known-good `:sha-…` tag;
  - how to update Mongo, cloudflared and the OS.
- **User admin:**
  - invite links;
  - changing the code;
  - listing users;
  - resetting a password;
  - removing an account (manual `mongosh` steps until there's a command for it).
- **Troubleshooting:** common symptoms and their fixes:
  - site down;
  - can't log in;
  - "too many requests";
  - Quick Tunnel address changed;
  - disk full;
  - Pi overheating;
  - Pi won't boot.
- **Phase 2 switch:** a domain checklist (§3.1, phase 2).

---

## 4. Decisions

| # | Decision | Status |
|---|---|---|
| D1 | **Two phases:** no domain now (home address plus optional Quick Tunnel), permanent tunnel when a domain exists | **Locked** (your choice) |
| D2 | **Invite code** (`REGISTRATION_INVITE_CODE`) with a prefilled invite link; field shown only when required. *(Alt: e-mail allowlist; stricter, but you'd edit `.env` and restart for every new person.)* | **Locked** (default you accepted) |
| D3 | **Automatic app updates nightly** (04:00): back up first, roll back automatically if the new version isn't healthy within 2 minutes. Mongo and cloudflared are bumped by hand monthly; OS security patches come via `unattended-upgrades`. *(Alt: no timer, you run `update.sh` yourself.)* | Recommended |
| D4 | **Backups:** daily at 03:30, keep 14, in `BACKUP_DIR`. Default is on the Pi; **recommended:** point it at a USB drive, plus a monthly copy to your computer, since an SD card or SSD failure would take the database *and* on-Pi backups with it. | Recommended |
| D5 | **Password help** = the admin `reset-password` command. Self-service "change password" goes to the backlog: resets will be rare for a family app, and the temporary password is a readable phrase people can keep. *(Alt: add `POST /me/password` + a settings form in this step.)* | Recommended |
| D6 | **Client IP:** trust `CF-Connecting-IP` **only** from the tunnel containers' fixed addresses (§3.3) | Recommended |
| D7 | **Install as a git clone** at `/opt/thisisyouryear`; scripts update only when you `git pull` | Recommended |

---

## 5. Files (created + / changed ~)

```
internal/config/config.go              ~ TRUSTED_PROXIES, CLIENT_IP_HEADER, REGISTRATION_INVITE_CODE
                                         (+ validation) · config_test.go ~
internal/transport/http/clientip.go    + resolve-once middleware + context accessor (+ test)
internal/transport/http/router.go      ~ wire the middleware; RealIP comment; GET /auth/registration
internal/transport/http/ratelimit.go   ~ key = resolved client IP
internal/transport/http/auth_handler.go ~ session IP from context; inviteCode; registration info
internal/transport/http/middleware.go  ~ log client_ip alongside remote_ip
internal/transport/http/dto.go         ~ RegisterRequest.inviteCode; RegistrationInfoResponse
internal/service/auth_service.go       ~ invite check; ResetPassword (+ tests)
internal/repository/users.go           ~ SetPasswordHash, List (+ integration tests)
cmd/api/main.go                        ~ subcommand switch (no args = server, as today)
cmd/api/admin.go                       + healthcheck, admin list-users, admin reset-password
api/openapi.yaml                       ~ inviteCode; GET /auth/registration
web/src/api/schema.d.ts                ~ regenerated (gen:api)
web/src/routes/Register.tsx            ~ invite field (conditional, prefilled from ?invite=) (+ tests)
web/src/test/msw.ts, a11y.test.tsx     ~ fixtures/handler for /auth/registration; a11y with field
deployments/pi/…                       + §3.6
docs/operations.md                     + §3.8
docs/dev_run.md                        ~ changelog; new env vars; client-IP + invite notes
README.md · CLAUDE.md                  ~ self-hosting pointer; client-IP + invite conventions
plans/backlog.md                       ~ change password; uptime monitoring; off-site backups;
                                         delete-account admin command
```

Existing behaviour is unchanged when the new settings are empty: same rate limiting, open
registration, server starts with no arguments.

---

## 6. Verification (definition of done)

- **Go:** `make fmt vet test test-int` and `make lint` (v2.15.0 from C2).
  - New unit tests:
    - client-IP resolution: a trusted peer uses the header; an untrusted peer's header is ignored;
      an invalid header falls back; empty config = `RemoteAddr`;
    - rate limiting per resolved IP;
    - invite: open, required+correct, required+wrong/missing;
    - `ResetPassword` revokes all sessions;
    - config validation.
  - New integration tests: the register and registration-info endpoints, and the user repository
    methods.
- **Web:** `typecheck`, `lint`, `format:check`, `test` (invite field hidden/shown/prefilled, field
  error rendering, axe), `build`, and the `gen:api` drift check.
- **Kit, checked statically:**
  - `docker compose config` for every profile;
  - `shellcheck` on the scripts;
  - `systemd-analyze verify` on the units.
- **Kit, rehearsed end-to-end on the dev machine** with a locally built image:
  - start the stack;
  - register with and without the code;
  - via the Quick Tunnel, confirm the real IP lands in the session and the log;
  - a spoofed `CF-Connecting-IP` sent to the home door is ignored;
  - backup → delete data → restore;
  - `update.sh` with a deliberately broken image → automatic rollback.
- **On the Pi:** you follow the handbook's *Install* section; I can't reach the Pi, so you send me any
  output that looks wrong.
- Docs updated. **Nothing committed.**

---

## 7. Order of work

1. **C2:** approve → implement → you push → first green Actions run → make the GHCR package public.
2. **C3:** approve → implement → you push → CI publishes an image that includes the C3 app changes.
3. **On the Pi:** install by following `docs/operations.md`.

**If you want the Pi running before C2 is done:** build on the Pi (`docker build -t
thisisyouryear:local .` in the clone) and set `APP_IMAGE=thisisyouryear:local`. The kit supports
this; automatic updates simply have nothing to pull until you switch back to the GHCR image.

---

## 8. Risks

- **Year rollover (time-sensitive).** If family start using the app in November/December, on
  **1 January 2027** their dashboard and Wins go blank and last year's goals become unreachable in the
  UI (see `plans/backlog.md`). Recommended: schedule the backlog's *minimal fix* (year switcher +
  `?year=` on the dashboard) right after C3, before invitations go out widely.
- **Quick Tunnel is for testing.** Cloudflare offers it with no uptime guarantee, and its address
  changes on every restart, which signs people out. That's fine for phase 1 with a few people; move
  to phase 2 before relying on it.
- **Storage failure.** SD cards wear out. Off-Pi backup copies (D4) are what make this survivable.
- **Power cuts.** Containers restart on boot, and MongoDB's journal makes crash recovery safe. A small
  UPS is optional.
- **16 KB page kernel.** Already ruled out: Mongo 7.0.45 passed the smoke test on your Pi. The
  handbook keeps the `kernel=kernel8.img` note in case an OS upgrade changes it.
