# Step C2 — CI/CD pipeline (GitHub Actions)

> **Status:** Revised draft for approval (2026-11-07)
> **Parent:** Phase C (production). Follows [`07-production-serving.md`](07-production-serving.md).
> **Builds on:** the full repo — Go backend (Steps 1–9), the SPA (B1–B6), and the
> single-origin embed build (C1) — all green locally.
> **Approval gate:** No files are written until you approve this document.
> **Revision note:** the first draft assumed no git remote and a `main` branch. The repo now lives
> on GitHub (`YehorBoldarev/ThisIsYourYear`, **public**), the production branch is **`master`**,
> and the golangci-lint config must move to the v2 format. Platform is locked to GitHub.

---

## 1. Objective

Run every quality gate automatically on push / PR, and build and publish the production container
image — so `master` is always known-green and a deployable image exists for whatever hosting comes
next. **Host-agnostic:** this step automates the gates + image; deploying it to a host is the next
step (C3, see §7).

### In scope
- **One workflow** (`.github/workflows/ci.yml`) with parallel jobs (§3).
- **golangci-lint v2:** migrate `.golangci.yml` to the v2 format, pin one version for CI and
  `make lint`, and fix whatever the stricter v2 defaults surface.
- **Node alignment** (D5) and **Dependabot** for action versions (D6).
- **Docs:** a CI section in `docs/dev_run.md`, a status badge in `README.md`, and the lint-version
  note in CLAUDE.md.

### Explicitly NOT in this step
- **No deploy to a host** (no SSH, no provider API). Publishing the image is the boundary.
- No release automation/changelogs.
- No branch-protection change — that's a repo setting you apply in GitHub once the first run is
  green (instructions in §6).

---

## 2. Current state (audited 2026-11-07)

- **Remote:** `github.com/YehorBoldarev/ThisIsYourYear`, public → Actions minutes on standard
  runners are free; the GHCR image can be public, so hosts pull it without credentials.
- **Branch:** `master` is production. No `.github/` yet.
- **Gates exist locally and pass:** `gofmt`, `go vet`, `go test -race` (unit + `-tags=integration`
  via testcontainers), web `typecheck`/`lint`/`format:check`/`test`/`build`, `gen:api` no-diff, and
  `docker build` (multi-stage, `-tags embedui`).
- **golangci-lint is broken for current tooling:** `.golangci.yml` uses the v1 schema
  (`disable-all`, `linters-settings`, `issues.exclude-rules`, formatters listed as linters).
  golangci-lint v2 (current: **v2.15.0**) and `golangci-lint-action` v7+ reject v1 configs. It's
  also not installed locally, so it hasn't been run recently.
- **Toolchains:** Go **1.26.4** (`go.mod`); Node **22** in the Dockerfile, **24** locally.

---

## 3. Design

```
.github/workflows/ci.yml
  on:  push → master · pull_request → master · push tags v* · workflow_dispatch
  permissions: contents: read  (docker job adds packages: write)
  concurrency: one run per ref; PR runs cancel superseded ones, master runs never cancel

  jobs (parallel):
    backend               setup-go (go.mod version, cached)
                          gofmt -l (must be empty) · go mod tidy (no diff)
                          go vet ./... (default + integration + embedui tags)
                          golangci-lint (official action, pinned v2.15.0)
                          go test -race ./...
    backend-integration   go test -race -tags=integration ./...   (Docker on ubuntu-latest)
    frontend              setup-node (D5, npm cache) in web/
                          npm ci · typecheck · lint · format:check · test · build
                          gen:api drift check (regenerate schema.d.ts; fail on diff)

  docker (needs all three):
                          buildx + GHA layer cache; build the Dockerfile on every run
                          push to GHCR only on master and v* tags (never on PRs)
                          ghcr.io/yehorboldarev/thisisyouryear
                            :sha-<short>   every published build (rollback handle)
                            :latest        master
                            :X.Y.Z, :X.Y   version tags
```

- **Pinned actions (current majors):** `actions/checkout@v7`, `actions/setup-go@v7`,
  `actions/setup-node@v7`, `golangci/golangci-lint-action@v9`, `docker/setup-buildx-action@v4`,
  `docker/login-action@v4`, `docker/metadata-action@v6`, `docker/build-push-action@v7`.
- **GHCR auth** uses the built-in `GITHUB_TOKEN` — no secrets to configure.

### golangci-lint v2 migration
- Convert `.golangci.yml` to `version: "2"`: `linters.default: none` + the same enabled set;
  `gofmt`/`goimports` move to the new `formatters` section (`goimports.local-prefixes:
  [kibou/thisisyouryear]`); `issues.exclude-rules` → `linters.exclusions.rules` (test files keep
  the errcheck/gosec exemption); keep the revive `exported` rule.
- Start from `golangci-lint migrate`, then review the result by hand.
- **Expect new findings:** v2 folds `gosimple`/`stylecheck` into `staticcheck` and replaces v1's
  implicit default exclusions with explicit presets. Real issues get fixed in code; anything
  deliberately excluded is listed in the config with a comment saying why.
- **One version, two consumers:** `Makefile` gets `GOLANGCI_LINT_VERSION := v2.15.0` and `make lint`
  runs exactly that version (no global install needed); the workflow pins the same value.

---

## 4. Decisions

| # | Decision | Status |
|---|---|---|
| D1 | **GitHub Actions**, production branch `master` | **Locked** |
| D2 | Integration tests run in CI as their own job on every push/PR (~1 min) | Recommended |
| D3 | Build the image on every run; **push to GHCR on `master` + `v*` tags**; make the package public once in GitHub (one click) so hosts pull without credentials | Recommended |
| D4 | One `ci.yml` with parallel jobs | Recommended |
| D5 | **Node 24 everywhere** (Dockerfile, CI, a new `web/.nvmrc`) — matches local dev; Node 22 leaves LTS in April 2027. *(Alt: keep 22 and use 22 locally.)* | Recommended |
| D6 | **Dependabot** (`.github/dependabot.yml`) for GitHub Actions only, weekly, grouped into one PR — keeps the pinned action majors current. Go/npm updates stay manual for now. | Recommended |
| — | golangci-lint v2 migration (§3) | **Locked** (you asked for it) |

---

## 5. Files (created + / changed ~)

```
.github/workflows/ci.yml     + the pipeline (§3)
.github/dependabot.yml       + Actions updates, weekly (D6)
.golangci.yml                ~ v2 format (§3)
Makefile                     ~ pinned GOLANGCI_LINT_VERSION; `lint` runs it
Dockerfile                   ~ node:24 (D5)
web/.nvmrc                   + 24 (D5)
<Go sources>                 ~ only if v2 lint findings need code fixes
docs/dev_run.md              ~ "Continuous integration" section + changelog
README.md                    ~ CI badge + what runs on push
CLAUDE.md                    ~ lint version note
```

No application behaviour changes.

---

## 6. Verification (definition of done)

- **Reproduce every job locally:** `gofmt -l` (empty), `go mod tidy` (no diff), `go vet` (all tags),
  `make lint` (v2.15.0, clean), `make test`, `make test-int`, the full web sequence incl. the
  `gen:api` drift check, and `docker build`.
- **Workflow lint:** `actionlint` over `ci.yml` and a YAML check of `dependabot.yml`.
- **Real run:** the definitive check is the first Actions run after you push. I don't push; once
  you do, I can read the run and fix anything runner-specific.
- **After the first green run (your repo settings):** Settings → Branches → add a rule for `master`
  requiring the `backend`, `backend-integration` and `frontend` checks; and Packages → the image →
  change visibility to public.
- Docs updated. **Nothing committed.**

---

## 7. How hosting plugs in later (C3)

C2 is complete without choosing a host: every container host consumes the same GHCR image. Hosting
adds **one `deploy` job** after `docker` (on `master` only), and its shape is the host's choice:
SSH + `docker compose pull && up -d` on a VPS, or a provider CLI/deploy hook on a platform. Deploy
credentials (SSH key or API token) go in a GitHub `production` environment; app config
(`JWT_SECRET`, `MONGO_URI`, `COOKIE_SECURE=true`, `APP_ENV=production`) lives on the host, never in
the repo or workflow. Rollback = redeploy a previous `:sha-…` tag.
