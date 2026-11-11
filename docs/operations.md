# Operations handbook — ThisIsYourYear on the Raspberry Pi

This is the admin guide for the self-hosted site: how to install it, what runs on its own, what
to check and when, where the logs are, and what to do when something goes wrong. The design
behind it is [`plans/frontend/09-self-hosting-raspberry-pi.md`](../plans/frontend/09-self-hosting-raspberry-pi.md).

Commands are run **on the Pi** (over SSH) unless a step says otherwise. Run anything that starts with
`docker compose` or `./` **from the kit directory**:

```bash
cd /opt/thisisyouryear/deployments/pi
```

---

## 0. At a glance

```
family's browser ──https──► https://<machine>.<tailnet>.ts.net
                                │  Tailscale Funnel (TLS ends here, on the Pi; certificate
                                ▼  renewed automatically)
                   tailscaled (OS service) ──http──► 127.0.0.1:8080 ──► api ──► mongo
```

| What | Where / how |
|---|---|
| Public address | `https://<machine>.<tailnet>.ts.net` (Tailscale Funnel) |
| App + database | Docker Compose project `tiyy`, run from `/opt/thisisyouryear/deployments/pi` |
| Settings & secrets | `/opt/thisisyouryear/deployments/pi/.env` (`chmod 600`, never committed) |
| Data | Docker volume `tiyy_mongo-data` |
| Backups | `BACKUP_DIR` in `.env` (default `/var/backups/thisisyouryear`), daily, newest 14 kept |
| App reachable from | the Pi itself only (`127.0.0.1:8080`); everyone else comes through Tailscale |
| Open to your home network | SSH (port 22), nothing else (`ufw`) |
| Open on the router | nothing; Tailscale connects outward |

**Runs on its own:**

| When | What | Check it with |
|---|---|---|
| Daily 03:30 | Database backup (`tiyy-backup.timer` → `backup.sh`) | `journalctl -u tiyy-backup -n 20` |
| Daily 04:00 | App update: backup, new image, **automatic rollback** if unhealthy (`tiyy-update.timer` → `update.sh`) | `journalctl -u tiyy-update -n 30` |
| Continuously | Containers restart after crashes and reboots (`restart: unless-stopped`) | `docker compose ps` |
| Automatically | HTTPS certificate renewal (Tailscale) | `tailscale funnel status` |
| Automatically | Tailscale updates (`--auto-update`) | `tailscale version` |
| Daily | Debian security updates (`unattended-upgrades`) | `cat /var/log/unattended-upgrades/unattended-upgrades.log` |

---

## 1. Install (one time)

**Already done earlier:** Raspberry Pi OS 64-bit, static IP, SSH, `ufw` allowing SSH from the home
LAN only, Docker Engine + Compose with log rotation (`"log-driver": "local"`), and the MongoDB
smoke test.

### 1.1 Clear away the early test (only if you ran it)

If you tried Funnel before this kit existed (`tiyy-test-*` containers), remove that test, data
included. Its network uses the same address range as the real one, so it must go first:

```bash
sudo docker rm -fv tiyy-test-api tiyy-test-mongo 2>/dev/null; sudo docker network rm tiyy-test 2>/dev/null
sudo tailscale funnel reset   # if you had turned Funnel on for the test
```

### 1.2 Get the kit

```bash
sudo apt install -y git
sudo git clone https://github.com/YehorBoldarev/ThisIsYourYear.git /opt/thisisyouryear
cd /opt/thisisyouryear/deployments/pi
```

The clone is only a source of the kit files. The app itself comes from the published image, and
the Pi never builds code.

### 1.3 Settings (`.env`)

```bash
sudo cp .env.example .env && sudo chmod 600 .env
openssl rand -hex 32          # copy the output…
sudo nano .env                # …paste it after JWT_SECRET=, and choose REGISTRATION_INVITE_CODE
```

- **`REGISTRATION_INVITE_CODE`:** at least 8 characters, easy to type, not guessable
  (e.g. `sunflower-meadow-2027`). Avoid `$`.
- **Keep a copy of `.env` somewhere safe** (a password manager). You need it to rebuild the Pi.
- **Backups to a USB drive** (recommended): see [§5.2](#52-backups-on-a-usb-drive), then set
  `BACKUP_DIR` accordingly.

### 1.4 Start the site

```bash
sudo docker compose up -d
sudo docker compose ps                    # both services "Up … (healthy)"
curl -s http://127.0.0.1:8080/readyz      # {"status":"ready"}
```

The first start downloads the images, which takes a minute or two.

### 1.5 Turn on the timers (backups + updates)

```bash
sudo cp systemd/tiyy-*.service systemd/tiyy-*.timer /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now tiyy-backup.timer tiyy-update.timer
systemctl list-timers 'tiyy-*'            # shows the next run of each
sudo systemctl start tiyy-backup.service  # take a first backup now
journalctl -u tiyy-backup -n 5            # "backup ok"
```

### 1.6 Automatic security updates for the OS

```bash
sudo apt install -y unattended-upgrades
sudo dpkg-reconfigure -plow unattended-upgrades   # answer "Yes"
```

### 1.7 Tailscale and Funnel

1. **Install and log in.** Open the printed link and log in with the account that will own the
   tailnet:
   ```bash
   curl -fsSL https://tailscale.com/install.sh | sh
   sudo tailscale up
   ```
2. **Choose the public name *before* sharing any link.** Renaming later breaks every link.
   - **Machine name:** in the [admin console](https://login.tailscale.com/admin/machines), open
     **Machines**, then the Pi's **⋯** menu → **Edit machine name…**, e.g. `year`.
   - **Tailnet name (optional):** the **DNS** page shows the tailnet name (e.g. `tail1234.ts.net`).
     You can rename it once to one of Tailscale's generated names.
   - **These names become public** (HTTPS certificates are logged publicly), so don't use your
     name or anything identifying.
3. **Disable key expiry for the Pi:** **Machines** → the Pi's **⋯** menu → **Disable key
   expiry**. Otherwise the Pi drops off Tailscale after 180 days **and the site goes offline**.
4. **Turn on Funnel:**
   ```bash
   sudo tailscale funnel --bg 8080
   ```
   The first time, it prints a link to approve HTTPS and Funnel for your tailnet; approve, then
   run the command again if it asks. `--bg` keeps Funnel on across reboots. Check it:
   ```bash
   tailscale funnel status      # https://year.<tailnet>.ts.net (Funnel on) → proxy http://127.0.0.1:8080
   ```
5. **Keep Tailscale updated automatically:**
   ```bash
   sudo tailscale set --auto-update
   ```

### 1.8 First check, then invite people

1. **On your phone, with Wi-Fi off** (so the request really comes from the internet), open
   `https://year.<tailnet>.ts.net`. You should see the sign-in page with a valid padlock.
2. **Register yourself** with the invite link:
   `https://year.<tailnet>.ts.net/register?invite=<your code>`. The code field is prefilled.
3. **Check that sign-ups are recorded:**
   ```bash
   sudo docker compose exec api /app/api admin list-users
   ```
4. **Invite one or two people first, then everyone.** Send them the invite link.

---

## 2. Rules of thumb

- **Never run `docker compose pull` by hand.** Use `sudo ./update.sh`, which keeps a rollback copy
  first.
- **Don't open ports on the router**, and don't add `ports:` entries to the compose file. Docker's
  published ports bypass `ufw`.
- **Change settings in `.env`, then apply them** with `sudo docker compose up -d`. That recreates
  only what changed; data is kept.
- **Take a backup before anything risky:** `sudo ./backup.sh`.

---

## 3. Maintenance schedule

### Weekly (2 minutes)

```bash
systemctl --failed                                  # should list 0 units
systemctl list-timers 'tiyy-*'                      # both scheduled, with a recent "LAST"
journalctl -u tiyy-update -n 5 --no-pager           # "already up to date" or "update ok"
sudo docker compose ps                              # both "(healthy)"
tailscale funnel status                             # Funnel on
df -h /                                             # disk use well below 80 %
```

### Monthly (10 minutes)

1. **Copy the newest backup off the Pi** ([§5.3](#53-copy-a-backup-to-your-computer)).
2. **Check the Pi's health:**
   - `vcgencmd measure_temp`: should be under ~70 °C.
   - `vcgencmd get_throttled`: `throttled=0x0` means no overheating and no weak power supply.
3. **Pending OS updates:**
   ```bash
   sudo apt update && apt list --upgradable
   sudo apt full-upgrade
   ```
   If `/var/run/reboot-required` exists afterwards, reboot during a quiet hour (`sudo reboot`);
   everything restarts by itself.
4. **MongoDB patch release** ([§6.3](#63-mongodb)): is there a newer `7.0.x` than the one in `.env`?
5. **Docker housekeeping:** `sudo docker system df`. If images take a lot of space, run
   `sudo docker image prune -f`.

### Every 3 months — restore drill (5 minutes)

A backup you've never restored is only a hope:

```bash
ls -1t /var/backups/thisisyouryear | head -1          # newest archive
sudo ./restore.sh --drill /var/backups/thisisyouryear/<that file>
```

It restores into a throwaway database and prints the document counts. Live data is untouched. The
counts should look plausible: users ≈ your family, topics and goals > 0.

### Yearly

- **Accounts and access:**
  - Review the accounts: `sudo docker compose exec api /app/api admin list-users`.
  - Rotate the invite code if it has spread too far ([§7.2](#72-change-or-close-sign-up)).
- **Tailscale:** in the admin console, confirm **key expiry is still disabled** for the Pi.
- **Backup copy:** check that the off-Pi backup copy exists and opens.

---

## 4. Logs and status

### 4.1 Quick status

```bash
cd /opt/thisisyouryear/deployments/pi
sudo docker compose ps; tailscale funnel status; systemctl list-timers 'tiyy-*'; df -h /
```

### 4.2 The app

The logs are JSON, one line per request or event.

```bash
sudo docker compose logs --tail 100 api              # recent
sudo docker compose logs -f api                      # follow live (Ctrl+C to stop)
sudo docker compose logs --since 24h api | grep '"level":"ERROR"'     # errors today
sudo docker compose logs --since 24h api | grep '"status":5'          # server errors (5xx)
sudo docker compose logs --since 24h api | grep '"status":429'        # rate-limited sign-in attempts
```

- **Each request line** has `client_ip` (the visitor's real IP, from Tailscale) and `remote_ip`
  (always `172.30.42.1:…`, the Docker gateway Tailscale's requests come through).
- **Nicer filtering:** `sudo apt install jq`, then:
  ```bash
  sudo docker compose logs --no-log-prefix --since 24h api | jq -c 'select(.status? >= 500)'
  ```

### 4.3 MongoDB

```bash
sudo docker compose logs --tail 50 mongo
```

### 4.4 Tailscale

```bash
tailscale status                                     # connected? (the Pi listed, no "offline")
tailscale funnel status                              # public address and where it forwards
journalctl -u tailscaled --since today --no-pager | tail -50
```

### 4.5 Backups and updates (the timers)

```bash
systemctl list-timers 'tiyy-*'
journalctl -u tiyy-backup --since "7 days ago" --no-pager
journalctl -u tiyy-update --since "7 days ago" --no-pager
systemctl status tiyy-update.service                 # result of the last run
```

### 4.6 The rest of the Pi

| What | Command |
|---|---|
| Docker daemon | `journalctl -u docker --since "1 hour ago" --no-pager` |
| SSH logins | `journalctl -u ssh --since today --no-pager` · `last -n 20` |
| Firewall rules | `sudo ufw status verbose` |
| Blocked connections | `sudo journalctl -k --since today \| grep UFW` |
| Temperature / throttling | `vcgencmd measure_temp` · `vcgencmd get_throttled` (`0x0` = fine) |
| Disk / memory / load | `df -h /` · `sudo docker system df` · `free -h` · `uptime` |
| OS security updates | `cat /var/log/unattended-upgrades/unattended-upgrades.log` |
| Anything that failed | `systemctl --failed` |

Container logs are rotated by Docker (the `local` log driver, about 100 MB per container), so they
can't fill the disk.

---

## 5. Backups and restore

### 5.1 What, where, how often

- **What:** the whole database, as a gzip-compressed `mongodump` archive.
- **Where:** `BACKUP_DIR`, named `tiyy-YYYYMMDD-HHMMSS.archive.gz` (readable by root only).
- **How often:** daily at 03:30, before every app update, and before every restore. The newest
  `BACKUP_KEEP` (14) are kept.
- **Back up right now:** `sudo ./backup.sh`.

### 5.2 Backups on a USB drive

If the Pi's SD card or SSD dies, backups stored on it die too. A USB stick or drive fixes that:

```bash
lsblk -f                                   # find the drive, e.g. sda1, and its UUID
sudo mkfs.ext4 -L tiyy-backup /dev/sda1    # ERASES the drive — double-check the name
sudo mkdir -p /mnt/tiyy-backup
echo 'LABEL=tiyy-backup /mnt/tiyy-backup ext4 defaults,nofail 0 2' | sudo tee -a /etc/fstab
sudo mount -a && df -h /mnt/tiyy-backup
```

Then set `BACKUP_DIR=/mnt/tiyy-backup/thisisyouryear` in `.env` and run `sudo ./backup.sh`.
`nofail` means the Pi still boots if the drive is unplugged; the backup then fails loudly, and
`systemctl --failed` shows it.

### 5.3 Copy a backup to your computer

Run this **on your computer**; backups are root-only on the Pi, hence `sudo cat`:

```bash
ssh pi@192.168.1.50 'sudo ls -1t /var/backups/thisisyouryear | head -1'      # newest name
ssh pi@192.168.1.50 'sudo cat /var/backups/thisisyouryear/<that file>' > <that file>
```

### 5.4 Restore after a mistake

For example, someone deleted something important yesterday:

```bash
sudo ./restore.sh /var/backups/thisisyouryear/<archive from before the mistake>
```

It asks you to type `restore`, takes a safety backup of the current data, stops the app, replaces
the database, restarts the app and prints the counts. **Everything written after that backup is
lost**, including accounts created since.

### 5.5 Rebuild on a new SD card or SSD

1. Repeat the earlier setup: OS, static IP, SSH, `ufw`, Docker.
2. **In the Tailscale admin console, remove the old Pi** (Machines → ⋯ → Remove). Its name is then
   free for the new one.
3. Follow [§1.2](#12-get-the-kit)–[§1.7](#17-tailscale-and-funnel), restoring your saved `.env`
   instead of creating a new one, and giving the machine **the same name** as before, so all links
   keep working.
4. Copy your newest backup onto the Pi, then
   `sudo ./restore.sh --yes /path/to/tiyy-….archive.gz`.

---

## 6. Updates and rollback

### 6.1 The app (automatic)

- **What happens:** every night `update.sh` checks for a new `:latest` image. A new image appears
  each time a pull request is merged to `master` and CI is green. If there's a new image:
  1. it backs up the database;
  2. it starts the new version;
  3. it waits up to 2 minutes for it to report healthy;
  4. if it doesn't, it **rolls back** to the previous version and the run is marked failed.
- **After a failed update** (`systemctl --failed` shows `tiyy-update.service`):
  - The site is still running the previous version; nobody is affected.
  - See why: `journalctl -u tiyy-update -n 60`.
  - It will retry and roll back every night until a fixed version is published. To stop that, pin
    the working version (§6.2).
- **Update right now:** `sudo ./update.sh`.

### 6.2 Pin or roll back to a specific version

Every published build has a tag `sha-<first 7 characters of the commit>`. They're listed on GitHub
under **Packages → thisisyouryear**.

```bash
sudo nano .env      # APP_IMAGE=ghcr.io/yehorboldarev/thisisyouryear:sha-4271996
sudo docker compose up -d api
```

While pinned, nightly updates do nothing ("already up to date"). To follow `:latest` again, set
`APP_IMAGE=ghcr.io/yehorboldarev/thisisyouryear:latest` and run `sudo ./update.sh`.

### 6.3 MongoDB

MongoDB is pinned on purpose; it's updated by hand within **7.0.x** only.
1. Look up the newest `7.0.x` tag on [Docker Hub](https://hub.docker.com/_/mongo/tags?name=7.0.).
2. Apply it:
   ```bash
   sudo ./backup.sh
   sudo nano .env          # MONGO_IMAGE=mongo:7.0.<new>
   sudo docker compose up -d
   sudo docker compose ps  # both healthy again
   ```
3. If it doesn't come back healthy, put the old tag back and run `sudo docker compose up -d`.

Moving to MongoDB 8 is a project of its own: it needs a plan. On the Pi 5, some 8.0.x releases have
known problems.

### 6.4 Tailscale and the OS

- **Tailscale** updates itself (`sudo tailscale set --auto-update`). Manually: `sudo tailscale update`.
- **The OS:** `unattended-upgrades` installs security updates daily. Everything else is in the
  monthly checklist (`sudo apt full-upgrade`).
- **If an OS upgrade ever switches the kernel to 16 KB pages**, MongoDB may crash on start.
  `getconf PAGESIZE` shows `16384`. The fix: add `kernel=kernel8.img` under `[all]` in
  `/boot/firmware/config.txt` and reboot.

---

## 7. Users and access

### 7.1 Invite someone

Send them `https://<machine>.<tailnet>.ts.net/register?invite=<REGISTRATION_INVITE_CODE>`.
The code is prefilled; they only choose a name, email and password.

### 7.2 Change or close sign-up

- **New code:** edit `REGISTRATION_INVITE_CODE` in `.env`, then run `sudo docker compose up -d`.
  Existing accounts are unaffected; old links stop working.
- **Close sign-up:** set the code to a long random value you never share (`openssl rand -hex 16`).
- **Never leave it empty** while the site is public. Empty means anyone who finds the address can
  register.

### 7.3 See who has an account

```bash
sudo docker compose exec api /app/api admin list-users
```

### 7.4 Someone forgot their password

```bash
sudo docker compose exec api /app/api admin reset-password --email someone@example.com
```

It prints a temporary password, such as `k7pq-m3xd-r9tw-h2nf`, once, and signs that person out on
every device. Send it to them privately (not in a group chat). There's no "change password" screen
yet (it's in the backlog), so they keep using the temporary one.

### 7.5 Remove an account

There's no command for this yet. This deletes the account **and all of its goals**, so take a
backup first:

```bash
sudo ./backup.sh
sudo docker compose exec -T mongo mongosh thisisyouryear --quiet --eval '
  const u = db.users.findOne({ email: "someone@example.com" });   // lower-case email
  if (!u) { print("no such account"); quit(1); }
  print("goals",    db.goals.deleteMany({ userId: u._id }).deletedCount);
  print("topics",   db.topics.deleteMany({ userId: u._id }).deletedCount);
  print("sessions", db.sessions.deleteMany({ userId: u._id }).deletedCount);
  print("user",     db.users.deleteOne({ _id: u._id }).deletedCount);'
```

---

## 8. Secrets

| Secret | Where | If it leaks / to rotate |
|---|---|---|
| `JWT_SECRET` | `.env` | Generate a new one (`openssl rand -hex 32`), put it in `.env`, `sudo docker compose up -d`. Nobody is logged out for long: their next page load re-authenticates silently. |
| `REGISTRATION_INVITE_CODE` | `.env` | Change it ([§7.2](#72-change-or-close-sign-up)). |
| The Pi's Tailscale identity | tailscaled's state on the OS | Remove the machine in the admin console, then `sudo tailscale up` again (same name). |
| SSH access to the Pi | your computer's SSH key / the Pi password | Anyone with root on the Pi controls all of the above. Keep SSH LAN-only. |

`.env` is git-ignored and `chmod 600`. Never paste it into chats, issues or AI tools. Keep one copy
in a password manager.

---

## 9. Troubleshooting

| Symptom | Check | Fix |
|---|---|---|
| Site unreachable from outside | `tailscale status` (Pi online?) · `tailscale funnel status` (Funnel on?) · `curl -s 127.0.0.1:8080/readyz` | Tailscale offline: `sudo systemctl restart tailscaled`, check key expiry in the admin console. Funnel off: `sudo tailscale funnel --bg 8080`. App down: next row. |
| `readyz` fails / containers not healthy | `sudo docker compose ps` · `sudo docker compose logs --tail 50 api mongo` | `sudo docker compose up -d`. A bad release: pin the previous version ([§6.2](#62-pin-or-roll-back-to-a-specific-version)). |
| Certificate warning in the browser | `tailscale funnel status` | Usually temporary while Tailscale renews it; `sudo systemctl restart tailscaled`. |
| Pi dropped off Tailscale | admin console: "Expired" badge | `sudo tailscale up`, then **Disable key expiry** for the Pi. |
| "Too many requests" when signing in | logs: `grep '"status":429'` | Wait one minute (20 sign-in attempts per minute per person). If everyone gets it at once, check `client_ip` in the logs isn't the same for all; if it is, `TRUSTED_PROXIES` is wrong. |
| Someone can't sign up: "invite code is not valid" | the link they used | Send the current link ([§7.1](#71-invite-someone)). |
| Someone can't sign in | `admin list-users` (right email?) | [§7.4](#74-someone-forgot-their-password). |
| `tiyy-update.service` failed | `journalctl -u tiyy-update -n 60` | Rolled back automatically; see [§6.1](#61-the-app-automatic). If it says it "can't keep a rollback copy": `sudo docker compose up -d`, then `sudo ./update.sh`. |
| `tiyy-backup.service` failed | `journalctl -u tiyy-backup -n 30` | Stack running? USB drive mounted (`df -h`)? Disk full? |
| Disk almost full | `df -h /` · `sudo docker system df` · `sudo du -sh /var/backups/thisisyouryear` | `sudo docker image prune -f`; lower `BACKUP_KEEP`; move backups to USB. |
| Pi hot or slow | `vcgencmd measure_temp` · `vcgencmd get_throttled` | Add a heatsink or fan (the official Pi 5 Active Cooler). A non-zero throttled value can also mean a weak power supply; use the official 27 W USB-C one. |
| MongoDB keeps restarting after an OS upgrade | `getconf PAGESIZE` | `16384` → [§6.4](#64-tailscale-and-the-os) (`kernel=kernel8.img`). |
| Pi won't boot | — | Rebuild on a new card ([§5.5](#55-rebuild-on-a-new-sd-card-or-ssd)). That's what the off-Pi backups are for. |

---

## 10. Not enabled: remote admin over Tailscale

> **Not enabled.** You chose (plan C3, D8) to manage the Pi only from home. This section is
> here so you can turn it on later if you want to fix things while away.

**What it gives you:** SSH into the Pi from anywhere, from your own devices only. Each one must be
logged into **your** tailnet. It's never public: Funnel exposes only port 443, the website.

**What it costs:** SSH becomes reachable from beyond your home network. So **switch SSH to
key-only first**, as we agreed.

1. **Key-only SSH.** On your computer:
   ```bash
   ssh-keygen -t ed25519            # only if you don't have a key yet
   ssh-copy-id pi@192.168.1.50
   ssh pi@192.168.1.50              # must log in WITHOUT asking for the Pi password
   ```
   Then on the Pi, **only after** the key login above worked:
   ```bash
   echo 'PasswordAuthentication no' | sudo tee /etc/ssh/sshd_config.d/10-no-passwords.conf
   sudo systemctl restart ssh
   ```
   Keep your current session open and test a new one before closing it.
2. **Install Tailscale on your laptop or phone** and log in with the same account.
3. **Allow SSH through the Tailscale interface** (on the Pi):
   ```bash
   sudo ufw allow in on tailscale0 to any port 22 proto tcp comment 'SSH via Tailscale (own devices)'
   sudo ufw status verbose
   ```
4. **Connect** from anywhere: `ssh pi@year` (the machine name, via MagicDNS) or
   `ssh pi@<the Pi's 100.x address>` from `tailscale status`.

**To turn it off again:**
`sudo ufw delete allow in on tailscale0 to any port 22 proto tcp`.

---

## 11. Reference

| Path | What |
|---|---|
| `/opt/thisisyouryear/deployments/pi/` | the kit: `docker-compose.yml`, `.env`, `backup.sh`, `restore.sh`, `update.sh`, `lib.sh`, `systemd/` |
| `/etc/systemd/system/tiyy-*.{service,timer}` | installed timer units (copies of `systemd/`) |
| `BACKUP_DIR` (default `/var/backups/thisisyouryear`) | backup archives |
| Docker volume `tiyy_mongo-data` | the database files |
| Docker network `tiyy_tiyy` (`172.30.42.0/24`, gateway `172.30.42.1`) | private network between api and mongo |

- **Updating the kit itself:** `cd /opt/thisisyouryear && sudo git pull` covers the scripts and the
  compose file. After a pull, re-copy the systemd units if they changed (`sudo cp …` + `daemon-reload`,
  [§1.5](#15-turn-on-the-timers-backups--updates)) and run `sudo docker compose up -d`. The nightly
  timers never pull repo code.
- **The `api` binary's admin commands:** `sudo docker compose exec api /app/api help`.
