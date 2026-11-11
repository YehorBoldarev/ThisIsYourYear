# Step C3 — Self-hosting on a Raspberry Pi 5 (Tailscale Funnel)

> **Status:** Approved (2026-11-11: D1a **Funnel**, condition verified; D8 **not enabled**, guide in
> the handbook) and implemented (2026-11-11). See §9.
> **Parent:** Phase C (production). Follows [`08-ci-cd-pipeline.md`](08-ci-cd-pipeline.md) (C2).
> **Depends on:** C2: done and green. The multi-arch image `ghcr.io/yehorboldarev/thisisyouryear` is
> published and publicly pullable (`linux/amd64` + `linux/arm64`, checked 2026-11-11).
> **Approval gate:** No files are written until you approve this document.
> **Note:** This step touches Go, the SPA, the OpenAPI contract and deployment files, so it runs the
> **Go and web** quality gates plus checks for the shell scripts and compose files.
> **Revision note (2026-11-11):** the first draft exposed the app through Cloudflare (a Quick
> Tunnel now, a permanent tunnel once a domain existed). You chose **Tailscale** instead. **Tailscale
> Funnel** gives the Pi a permanent public HTTPS address on Tailscale's own domain
> (`https://<machine>.<tailnet>.ts.net`) with no domain to buy, so the two phases collapse into one.
> `cloudflared`, the Quick Tunnel and the plain-HTTP home address are gone. The cookie stays
> `Secure`, and the trusted client-IP source becomes Tailscale's proxy.

---

## 1. Objective

Run the production app on your Raspberry Pi 5 for family and friends at a permanent HTTPS address
that needs no domain. It should be safe, back itself up and update itself, and come with an admin
handbook that tells you what to check, when, and how.

```
family's browser ──https──► https://year.<tailnet>.ts.net
                              │  Tailscale Funnel: TLS terminates on the Pi (Let's Encrypt cert,
                              ▼  renewed automatically)
                 tailscaled (on the Pi's OS) ──http──► 127.0.0.1:8080 ──► api ──► mongo
```

### In scope
1. **App changes**
   - **Real client IP behind Tailscale** (§3.3), so login rate-limiting is per visitor, not global.
   - **Invite-only registration** (§3.4) with an invite code and a shareable invite link.
   - **Admin commands** in the same binary (§3.5): `healthcheck`, `admin reset-password`,
     `admin list-users`.
2. **Pi deployment kit** `deployments/pi/` (§3.6): compose file, `.env` template, backup / restore /
   update scripts, and systemd timers that run backups and updates on schedule.
3. **Tailscale setup** on the Pi (§3.2): install, Funnel, key expiry, auto-updates, and optional
   remote admin (D8). These are documented commands, not repo code.
4. **Admin handbook** `docs/operations.md` (§3.8): install, maintenance schedule, logs for every
   component, backup & restore, updates & rollback, user admin, troubleshooting.
5. **Docs:** `docs/dev_run.md` (changelog + new settings), `README.md`, `CLAUDE.md`,
   `plans/backlog.md`.

### Explicitly NOT in this step
- **A custom domain.** Funnel only serves `*.ts.net` names. If you want your own domain later,
  that's a separate step (e.g. a Cloudflare Tunnel). The client-IP mechanism (§3.3) is
  header-configurable, so it carries over.
- **Self-service password change / "forgot password" e-mails** (D5 → backlog).
- **Uptime monitoring / alerting services** (backlog; easier now that the address is permanent).
- **The year-rollover fix.** It stays in the backlog, but see the risk in §8: it's due before
  1 January 2027.

---

## 2. Current state (audited 2026-11-11)

- **The Pi:**
  - Raspberry Pi 5 (arm64), Raspberry Pi OS 64-bit.
  - Static IP; SSH works.
  - `ufw` allows SSH from the home LAN only.
  - Docker Engine + Compose installed, with log rotation.
  - `mongo:7.0.45` passed the smoke test.
  - Tailscale is not installed yet.
- **Image:** C2 is green. `ghcr.io/yehorboldarev/thisisyouryear:latest` is a multi-arch index
  (amd64 + arm64) and pulls **anonymously**, so the Pi needs no registry credentials.
- **Tailscale Funnel facts that shape this design** (Tailscale docs and source, checked
  2026-11-11):
  - **Plans and requirements:** available on **all plans** (free included). It needs MagicDNS,
    HTTPS certificates and a `funnel` node attribute; the first `tailscale funnel` run walks you
    through all three.
  - **Address and ports:** public only on ports 443/8443/10000, only for the tailnet's `*.ts.net`
    name, and only over TLS.
  - **Bandwidth:** subject to non-configurable bandwidth limits. That's ample for a family app.
  - **Client IP:** when proxying, tailscaled **sets** (overwrites) `X-Forwarded-For` to the
    connection's source address. For Funnel connections that is the **visitor's public IP**, not
    the relay's (`ipn/ipnlocal/serve.go`: `addProxyForwardedHeaders`, `HandleIngressTCPConn`).
  - **Key expiry:** device keys expire after **180 days** by default. If the Pi's key expires, it
    drops off Tailscale and **the site goes offline** until you re-authenticate. Expiry can be
    disabled per device.
  - **Public names:** HTTPS certificates are recorded in public **Certificate Transparency
    logs**, so the machine name and tailnet name become public. Scanners will find the login
    page; that's normal for any website.
- **Docker networking (tested locally, Docker 29):** a request that reaches a container through
  a port published on `127.0.0.1` arrives from the **compose network's gateway address** (e.g.
  `172.30.42.1`), not from `127.0.0.1`. That address is what the app must trust (§3.3).
- **Client IP in the app:** `clientIP()` uses `r.RemoteAddr` only. Behind Tailscale every visitor
  would appear as the gateway, which has three effects:
  - The auth rate limit (20 req/min) becomes **one shared bucket for everyone**.
  - Session records show the gateway's address instead of the visitor's.
  - The request logs show the gateway's address instead of the visitor's.
- **Registration:** open to anyone who can reach the site.
- **Account recovery:** there is no password reset of any kind and no change-password endpoint.
  `SessionRepository.RevokeAllForUser` already exists.
- **Existing compose files don't fit real users:**
  - The demo file runs in **development mode** with the **dev JWT secret that's published in the
    repo**.
  - It publishes `8080` on all interfaces.
  - It uses floating `mongo:7`.
  - Docker-published ports also **bypass `ufw`**.

---

## 3. Design

### 3.1 Topology

```
                         ┌──────────────────────────── Raspberry Pi ───────────────────────────┐
internet ──https:443──►  │ tailscaled (OS service) ──http──► 127.0.0.1:8080                    │
 (Funnel relay)          │                                     │ published on loopback only     │
                         │                     Docker network "tiyy" (fixed subnet,            │
                         │                     gateway 172.30.42.1) ▼                          │
                         │                                    api ──► mongo (volume, no ports) │
your laptop/phone ─────► │ tailscaled ──► sshd :22   (optional remote admin, D8)               │
 (in your tailnet)       └─────────────────────────────────────────────────────────────────────┘
```

- **Nothing listens on the home network except SSH** (ufw, LAN only). The router opens no ports;
  Tailscale connects outward.
- **The app is published on `127.0.0.1` only**, so neither the LAN nor the internet can bypass
  Tailscale and reach it directly. MongoDB is never published.
- **Family at home use the same `ts.net` address** as everyone else. There's one address, always
  HTTPS, so the cookie stays `Secure` (`COOKIE_SECURE=true`, the default).

### 3.2 Tailscale on the Pi (D1)

Tailscale runs as a normal **OS service** on the Pi, not in a container (D1). All of this is
written up in the handbook's *Install* section:

1. **Install and log in:** `curl -fsSL https://tailscale.com/install.sh | sh`, then
   `sudo tailscale up`. Open the printed link and log in with the account that will own the
   tailnet.
2. **Pick the public name before sharing any link**, because renaming later breaks every link.
   - **Machine name:** in the admin console, rename the Pi, e.g. to `year`.
   - **Tailnet name:** optionally choose one of Tailscale's generated tailnet names instead of the
     default `tail1234.ts.net`.
   - **Result:** something like `https://year.<tailnet>.ts.net`. Choose names that don't identify
     you (they're public, §2).
3. **Disable key expiry for the Pi** (admin console → the Pi → *Disable key expiry*). Otherwise
   the site goes offline after 180 days.
4. **Turn on Funnel:** `sudo tailscale funnel --bg 8080`.
   - The first run asks you to approve enabling HTTPS and Funnel for the tailnet.
   - `--bg` keeps it on across reboots.
   - It proxies `https://year.<tailnet>.ts.net` → `http://127.0.0.1:8080`.
5. **Keep Tailscale updated:** `sudo tailscale set --auto-update`.

### 3.3 Real client IP behind the proxy (D6)

**New settings:**
- `TRUSTED_PROXIES`: a comma-separated list of IPs/CIDRs allowed to tell us the client's IP.
  Default empty, which means today's behaviour exactly.
- `CLIENT_IP_HEADER`: default `X-Forwarded-For`. Only a **single IP** is accepted; a
  comma-separated chain counts as invalid and falls back, by design. That fits Tailscale, which
  overwrites the header with exactly one address. It also fits Cloudflare's `CF-Connecting-IP`
  later, should you add a domain.

**Behaviour:** one middleware, first in the chain, resolves the client IP once.
- **Trusted:** if the connection comes from an address in `TRUSTED_PROXIES` and the header holds
  exactly one valid IP, that header value is the client IP.
- **Otherwise:** the client IP is the connection's own address.

The result goes into the request context. These all use it:
- the auth rate limiter (one bucket per real visitor);
- the IP recorded on sessions;
- the request log, which logs both `client_ip` and the raw `remote_ip`.

**On the Pi:**
- The compose network has a fixed subnet (`172.30.42.0/24`), so its gateway is always
  `172.30.42.1`.
- `.env` sets `TRUSTED_PROXIES=172.30.42.1`.
- The `api` port is bound to `127.0.0.1`.

**Why this is safe:** only processes on the Pi itself can reach the app through that gateway:
tailscaled, plus you running `curl` while logged in over SSH. Tailscale replaces any
`X-Forwarded-For` a visitor sends, so a visitor can't fake their IP. Nothing on the LAN can
reach the app at all.

The `router.go` comment about `middleware.RealIP` gets updated to point to this vetted mechanism.

### 3.4 Invite-only registration (D2)

- **Setting:**
  - New setting `REGISTRATION_INVITE_CODE`. Empty means open registration, which is today's
    behaviour and the dev default.
  - When set, it must be at least 8 characters, and every registration must present it.
  - It's compared in constant time. The per-IP auth rate limit makes guessing impractical.
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
    `https://year.<tailnet>.ts.net/register?invite=<code>` and family only fill in name, e-mail and
    password.
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
  docker-compose.yml        mongo + api on network "tiyy" (fixed subnet); api published on
                            127.0.0.1:8080 only; restart: unless-stopped
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

| Service | Image | Notes |
|---|---|---|
| `mongo` | `${MONGO_IMAGE}`, default `mongo:7.0.45` (pinned) | Named volume and healthcheck. WiredTiger cache capped (default 1 GB). No `ports:`. |
| `api` | `${APP_IMAGE}`, default `ghcr.io/yehorboldarev/thisisyouryear:latest` | Settings from `.env` (`APP_ENV=production`, `TRUSTED_PROXIES=172.30.42.1`). Waits for `mongo` to be healthy. Healthcheck via `/app/api healthcheck`. `ports: "127.0.0.1:8080:8080"`. |

- **Install location:** `/opt/thisisyouryear` is a **git clone** of the public repo (D7). You run
  everything from `deployments/pi/`. Scripts change only when *you* run `git pull`; the timers never
  pull repo code.
- **What updates automatically:**
  - **The app image** updates nightly (D3).
  - **Tailscale** auto-updates itself (§3.2).
  - **OS security updates** come from Debian's `unattended-upgrades`.
  - **MongoDB** stays pinned in `.env` and is bumped by hand, following the handbook's monthly
    checklist.
- **Scripts quality bar:**
  - `set -euo pipefail` and readable output.
  - They log to the journal, so `journalctl -u tiyy-update` shows every update and rollback.

### 3.7 Secrets

All secrets live only in `/opt/thisisyouryear/deployments/pi/.env` (`chmod 600`, git-ignored):
- `JWT_SECRET` (generated on the Pi with `openssl rand -hex 32`);
- `REGISTRATION_INVITE_CODE`.

None of them goes in the repo, the workflow or a chat message.

**Tailscale needs no secret in the repo or `.env`.** The Pi's Tailscale identity lives in
tailscaled's own state on the OS. Protect it like SSH access: anyone with root on the Pi controls
it. The handbook explains how to rotate each secret and what rotating does. For example, a new
`JWT_SECRET` means everyone's next page load silently refreshes; it doesn't lose any data.

### 3.8 Admin handbook — `docs/operations.md`

The handbook is written last, against the finished kit, so every command in it is real.

- **Map:** what runs where (§3.1, in plain words).
- **Install:** from a fresh Pi to a running site:
  - clone the repo, write `.env`, first start, enable the timers;
  - `unattended-upgrades`;
  - Tailscale (§3.2);
  - check from a phone **on mobile data** that the address works;
  - optional remote admin (D8).
- **Maintenance schedule**

  | When | What |
  |---|---|
  | Weekly | Glance at the timers and last update result; check disk space; `tailscale funnel status`. |
  | Monthly | Copy a backup off the Pi. Bump Mongo if a new 7.0.x exists. Check temperature and throttling. Run `apt` upgrades if anything is held back. |
  | Every 3 months | **Restore drill:** restore the latest backup into a scratch container. |
  | Yearly | Review accounts and the invite code; rotate secrets if needed. Confirm key expiry is still disabled for the Pi. |
- **Logs, per component:**
  - app (`docker compose logs`, with filters for errors and 5xx in the JSON logs);
  - MongoDB;
  - **Tailscale** (`tailscale status`, `tailscale funnel status`, `journalctl -u tailscaled`);
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
  - rebuilding on a new SD card or SSD, including re-adding the Pi to Tailscale under the same
    name.
- **Updates & rollback:**
  - how automatic updates work;
  - how to pin a known-good `:sha-…` tag;
  - how to update Mongo, Tailscale and the OS.
- **User admin:**
  - invite links;
  - changing the code;
  - listing users;
  - resetting a password;
  - removing an account (manual `mongosh` steps until there's a command for it).
- **Troubleshooting:** common symptoms and their fixes:
  - site unreachable: is it Tailscale/Funnel or the app?
  - can't log in;
  - "too many requests";
  - certificate errors;
  - Pi dropped off Tailscale (key expiry);
  - disk full;
  - Pi overheating;
  - Pi won't boot.

---

## 4. Decisions

| # | Decision | Status |
|---|---|---|
| D1 | **Exposure via Tailscale Funnel**, with Tailscale installed **on the Pi's OS** and Funnel proxying to `127.0.0.1:8080`. *(Alt: a Tailscale container in the compose file; it keeps everything in Docker, but needs auth-key handling and gives no SSH access to the Pi itself.)* | **Locked** (Tailscale, your choice); host install recommended |
| D1a | **Funnel (public)** rather than **Serve (private)**. With Serve, only devices in your tailnet could open the site, so every family member would install the Tailscale app and you'd share the Pi with them. That's stricter, but much harder for non-technical people. With Funnel, anyone with the link sees the sign-in page; invite-only sign-up (D2) and the per-visitor rate limit (D6) protect it. | Recommended |
| D2 | **Invite code** (`REGISTRATION_INVITE_CODE`) with a prefilled invite link; field shown only when required. *(Alt: e-mail allowlist; stricter, but you'd edit `.env` and restart for every new person.)* | **Locked** (default you accepted) |
| D3 | **Automatic app updates nightly** (04:00): back up first, roll back automatically if the new version isn't healthy within 2 minutes. Mongo is bumped by hand monthly; Tailscale and OS security patches update themselves. *(Alt: no timer, you run `update.sh` yourself.)* | Recommended |
| D4 | **Backups:** daily at 03:30, keep 14, in `BACKUP_DIR`. Default is on the Pi; **recommended:** point it at a USB drive, plus a monthly copy to your computer, since an SD card or SSD failure would take the database *and* on-Pi backups with it. | Recommended |
| D5 | **Password help** = the admin `reset-password` command. Self-service "change password" goes to the backlog. *(Alt: add `POST /me/password` + a settings form in this step.)* | Recommended |
| D6 | **Client IP:** trust `X-Forwarded-For` (single IP only) **only** from the compose network's gateway, with the app bound to `127.0.0.1` (§3.3) | Recommended (security) |
| D7 | **Install as a git clone** at `/opt/thisisyouryear`; scripts update only when you `git pull` | Recommended |
| D8 | **Remote admin over Tailscale** (optional): add one `ufw` rule allowing SSH on the `tailscale0` interface. You can then SSH into the Pi from anywhere, but only from devices logged into *your* tailnet; it's never public. If you enable it, switch SSH to **key-only** first: SSH is then reachable beyond your home network, which is the condition we agreed makes keys necessary. | Recommended (optional) |

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
                                         delete-account admin command; custom domain
```

Existing behaviour is unchanged when the new settings are empty: same rate limiting, open
registration, server starts with no arguments.

---

## 6. Verification (definition of done)

- **Go:** `make fmt vet lint test test-int`.
  - New unit tests:
    - client-IP resolution: a trusted peer uses the header; an untrusted peer's header is ignored;
      an invalid header or a comma chain falls back; empty config = `RemoteAddr`;
    - rate limiting per resolved IP;
    - invite: open, required+correct, required+wrong/missing;
    - `ResetPassword` revokes all sessions;
    - config validation.
  - New integration tests: the register and registration-info endpoints, and the user repository
    methods.
- **Web:** `typecheck`, `lint`, `format:check`, `test` (invite field hidden/shown/prefilled, field
  error rendering, axe), `build`, and the `gen:api` drift check. CI runs all of this again on push.
- **Kit, checked statically:**
  - `docker compose config`;
  - `shellcheck` on the scripts;
  - `systemd-analyze verify` on the units.
- **Kit, rehearsed end-to-end on the dev machine** with a locally built image:
  - start the stack;
  - register with and without the code;
  - simulate Tailscale (a request from the host through `127.0.0.1:8080` with `X-Forwarded-For`)
    and confirm that IP lands in the session and the log;
  - the same header from a container on another network is ignored;
  - backup → delete data → restore;
  - `update.sh` with a deliberately broken image → automatic rollback.
- **On the Pi:** you follow the handbook's *Install* section, then open the `ts.net` address from a
  phone on mobile data and sign up with the invite link. I can't reach the Pi, so you send me any
  output that looks wrong.
- Docs updated. **Nothing committed.**

---

## 7. Order of work

1. **C3:** approve → implement → you push → CI publishes an image that includes the C3 app
   changes.
2. **On the Pi:** install by following `docs/operations.md` (Docker kit + Tailscale).
3. **Invite a couple of people first**, then everyone.

You can install Tailscale and try Funnel on the Pi right away, before C3 is implemented, using the
current GHCR image. **Never put the demo stack (`docker-compose.customer.yml`) behind Funnel.** It
runs in development mode with the JWT secret that's published in the repo, so anyone could forge a
login. Instead, run the image in **production mode with a freshly generated secret**:

```bash
docker network create --subnet 172.30.42.0/24 tiyy-test
docker run -d --name tiyy-test-mongo --network tiyy-test mongo:7.0.45
docker run -d --name tiyy-test-api --network tiyy-test -p 127.0.0.1:8080:8080 \
  -e APP_ENV=production -e MONGO_URI=mongodb://tiyy-test-mongo:27017 \
  -e JWT_SECRET="$(openssl rand -hex 32)" ghcr.io/yehorboldarev/thisisyouryear:latest
sudo tailscale funnel --bg 8080
```

Even this way, until C3 lands sign-up is open to anyone who finds the address, and every visitor
shares one login rate-limit bucket. So treat it as a **test only**: no real data.
- **Turn it off:** `sudo tailscale funnel reset`.
- **Remove it, test data included:** `docker rm -fv tiyy-test-api tiyy-test-mongo && docker
  network rm tiyy-test`.

---

## 8. Risks

- **Year rollover (time-sensitive).** If family start using the app in November/December, on
  **1 January 2027** their dashboard and Wins go blank and last year's goals become unreachable in the
  UI (see `plans/backlog.md`). Recommended: schedule the backlog's *minimal fix* (year switcher +
  `?year=` on the dashboard) right after C3, before invitations go out widely.
- **Key expiry.** If expiry isn't disabled, the site disappears after 180 days. It's in the install
  steps and in the yearly checklist.
- **Public discoverability.** Certificate Transparency logs publish the address, so bots will probe
  the sign-in page. Mitigated by invite-only sign-up, per-visitor rate limiting, bcrypt and
  non-identifying names.
- **Dependency on Tailscale.** If Tailscale has an outage or changes Funnel's terms, the site is
  unreachable from outside. Your data is unaffected, and a domain + Cloudflare Tunnel remains a
  fallback (backlog).
- **Storage failure.** SD cards wear out. Off-Pi backup copies (D4) are what make this survivable.
- **Power cuts.** Containers and tailscaled restart on boot, and MongoDB's journal makes crash
  recovery safe. A small UPS is optional.
- **16 KB page kernel.** Already ruled out: Mongo 7.0.45 passed the smoke test on your Pi. The
  handbook keeps the `kernel=kernel8.img` note in case an OS upgrade changes it.

---

## 9. Implementation notes (2026-11-11)

**Decisions as approved:**
- **D1a — Funnel (public).** You approved it on one condition: a random visitor must be unable to
  do anything without signing in. That condition was verified in code:
  - The SPA's only public routes are `/login` and `/register`; everything else is behind
    `RequireAuth`.
  - Every data endpoint is behind the access-token middleware.
  - The only unauthenticated API surface is `healthz`/`readyz`/`openapi.yaml` and the auth
    endpoints. Sign-up needs the invite code, and sign-in is bcrypt plus per-visitor rate
    limiting.
- **D8 — remote admin not enabled.** `docs/operations.md` §10 documents how to turn it on later
  (key-only SSH first).

**Differences from the draft, and findings:**
- **`AdminService`, not `AuthService`, owns the operator commands.** It has its own unscoped port,
  `AdminUserRepository`, plus `SessionRevoker`. That keeps the user-scoped auth ports untouched and
  makes the one unscoped port explicit and CLI-only.
- **The temporary password isn't four words.** It's four groups of four characters from an
  unambiguous alphabet (`k7pq-m3xd-r9tw-h2nf`, ~79 bits). That needs no embedded wordlist and is
  still easy to read out.
- **`api healthcheck` checks `/readyz`, not `/healthz`.** Being "healthy" then also means MongoDB is
  reachable, which is the right signal for `update.sh`'s rollback decision.
- **New `domain.FieldError`.** A business rule (the invite check) can now return a per-field 422
  through the single `respondError` mapping.
- **Bug found in rehearsal: `update.sh` must tag the rollback image *before* pulling.** With Docker's
  containerd image store (the default on new installs, including the Pi), an image that loses its
  last tag is deleted at once. Pulling first left nothing to roll back to. If tagging is impossible
  (someone pulled by hand), the script now stops without changing anything and says what to do.
- **Compose project name is `tiyy`.** It doesn't collide with the Windows demo (`thisisyouryear`).
  The kit passes only the variables the app needs into the container (no `env_file`), and refuses
  to start without `JWT_SECRET`.
- **The pre-C3 Funnel test recipe** used the same subnet, so the handbook's install begins by
  removing it.

**Verification:**
- **Go:** `make fmt vet lint` (0 issues), `make test`, and new integration tests (invite flow,
  session IP behind a trusted proxy, spoofed header ignored, `SetPasswordHash`/`List`).
- **Web:** typecheck, ESLint, Prettier, Vitest (182, incl. invite field + axe), build, `gen:api`.
- **Kit:**
  - shellcheck, `docker compose config`, and `systemd-analyze verify`; the only complaint is that
    `/opt/…` doesn't exist off the Pi;
  - a **full local rehearsal** with a local registry:
    - invite-only sign-up;
    - `client_ip` = forwarded IP via the `127.0.0.1` port, while a container on the network
      faking the header was recorded as its real address;
    - `admin list-users`/`reset-password` (old password 401, temporary 200, sessions revoked);
    - backup rotation, restore drill, live restore;
    - `update.sh`: no-op, broken release → automatic rollback (exit 1, previous version healthy),
      retry the next night, fixed release → updated.

