# Step C2 — CI/CD pipeline (GitHub Actions)

> **Status:** Draft for approval
> **Parent:** Phase C (production). Follows [`07-production-serving.md`](07-production-serving.md).
> **Builds on:** the full repo — Go backend (Steps 1–9), the SPA (B1–B6), and the
> single-origin embed build (C1) — all green locally.
> **Approval gate:** No files are written until you approve this document.
> **Heads-up:** the repo has **no git remote yet**. This plan targets **GitHub Actions**,
> so it assumes the repo will be pushed to GitHub (see **D1**). The workflow files are
> inert until then — nothing about them changes local development.

---

## 1. Objective

Run every quality gate automatically on push / PR, and build (and optionally publish) the
production container image — so `main` is always known-green and a deployable image exists
for whatever hosting comes later. **Host-agnostic:** this automates the gates + image; it
does not pick or provision a host (that's a separate Phase-C step).

### In scope
- **One workflow** (`.github/workflows/ci.yml`) triggered on push to `main` and on PRs,
  with **parallel jobs** and sensible caching/concurrency:
  - **backend** — Go 1.26: `gofmt` check, `go vet ./...`, **golangci-lint** (repo already
    has `.golangci.yml`), `go mod tidy` cleanliness check, and `go test -race ./...` (unit).
  - **backend-integration** — the `//go:build integration` suite (`go test -race
    -tags=integration ./...`) which spins up Mongo via testcontainers on the Docker-enabled
    Ubuntu runner (see **D2**).
  - **frontend** — Node 22 in `web/`: `npm ci`, `typecheck`, `lint`, `format:check`,
    `test` (Vitest, incl. the axe suite), `build`, plus a **`gen:api` drift check**
    (regenerate `schema.d.ts` and fail if it differs from the committed file).
  - **docker** — build the multi-stage image (`-tags embedui`) to prove the combined
    artifact compiles; **publish** it on push to `main` / tags per **D3**.
- **Caching:** Go build/module cache (`actions/setup-go` built-in) and npm cache
  (`actions/setup-node` with `cache: npm`, `cache-dependency-path: web/package-lock.json`).
- **Gating:** `docker` (and publish) run only after the test jobs pass (`needs:`).
- **Docs:** a CI section in `docs/dev_run.md` + a status badge note in `README.md`.

### Explicitly NOT in this step
- **No host provisioning / deploy** (no SSH, no `kubectl`, no PaaS API). Publishing an
  image is the boundary; actually running it somewhere is the next Phase-C step.
- No release automation/changelogs, no secret-manager setup beyond the registry token.
- No branch-protection or required-checks configuration (that's a repo setting you apply
  in GitHub once the workflow exists; I'll note it).

---

## 2. Current state (audited)

- **No remote, no `.github/`.** Fresh CI.
- **Gates already exist locally** and pass: Go (`gofmt`, `go vet`, `golangci-lint` via
  `.golangci.yml`, `go test -race`), web (`typecheck`/`lint`/`format:check`/`test`/`build`),
  and `go build -tags embedui`. CI just runs them on a runner.
- **Integration tests** are tag-gated (`//go:build integration`) and use testcontainers →
  they need a Docker daemon (present on `ubuntu-latest`). They're excluded from the default
  `go test`, so the unit job stays fast and Docker-free.
- **`web/.npmrc`** sets `legacy-peer-deps=true` (added in C1), so `npm ci` resolves cleanly
  on the runner.
- **Dockerfile** is the multi-stage embed build from C1 — CI builds it as-is.
- Toolchain versions to pin: **Go 1.26** (matches `go.mod` 1.26.4), **Node 22** (matches
  the Dockerfile + README's Node 20+).

---

## 3. Design (pending your decisions in §4)

```
.github/workflows/ci.yml
  on: push (main) + pull_request
  concurrency: cancel-in-progress per ref
  jobs:
    backend:              ubuntu-latest · setup-go 1.26 (cache) · gofmt -l · go vet ·
                          golangci-lint-action · go mod tidy check · go test -race ./...
    backend-integration:  ubuntu-latest · setup-go · go test -race -tags=integration ./...
                          (testcontainers → Docker on the runner)               [D2]
    frontend:             ubuntu-latest · setup-node 22 (npm cache) · working-directory web
                          npm ci · typecheck · lint · format:check · test · build ·
                          gen:api drift check
    docker:               needs: [backend, backend-integration, frontend]
                          build the image; on push to main/tags → login + push to GHCR [D3]
```

- **golangci-lint** runs via the official `golangci/golangci-lint-action` (handles its own
  caching) using the repo's `.golangci.yml`.
- **GHCR publish** (recommended) uses the built-in `GITHUB_TOKEN` with `packages: write` —
  **no extra secrets to configure**. Image tagged `ghcr.io/<owner>/thisisyouryear:<sha>`
  and `:latest` on `main` (and `:vX.Y.Z` on tags, via `docker/metadata-action`).

---

## 4. Decisions for you (defaults recommended; flag any before I build)

**D1 — CI platform.**
- **(Recommended) GitHub Actions.** The repo needs a GitHub remote (you create/push it);
  the workflow then runs automatically. Native GHCR publishing, zero extra secrets.
- **(Alt) GitLab CI / other.** I'd write a `.gitlab-ci.yml` (or equivalent) instead — say
  the word and I'll target that platform.

**D2 — Run integration tests in CI?**
- **(Recommended) Yes**, as their own job (`-tags=integration`, testcontainers + Docker on
  `ubuntu-latest`). Real end-to-end coverage on every change; ~1–2 min extra.
- **(Alt) Unit-only on PRs**, integration on a nightly schedule / manual dispatch — faster
  PR feedback, integration still covered but not per-push.

**D3 — Image publish.**
- **(Recommended) Build on every run; push to GHCR on `main` + version tags.** Uses
  `GITHUB_TOKEN`, no secret setup; gives you a pullable image immediately.
- **(Alt A) Build-only** (never push) — pure verification; you wire publishing when a host
  is chosen.
- **(Alt B) Docker Hub / other registry** — needs you to add registry credentials as repo
  secrets; I'll wire it to those.

**D4 — Workflow layout.**
- **(Recommended) One `ci.yml` with parallel jobs** — simplest to read; shared triggers.
- **(Alt) Split** (`backend.yml` / `frontend.yml` / `docker.yml`) — more granular status
  checks, more files to keep in sync.

---

## 5. Files (created ·/ changed) — indicative

```
.github/workflows/ci.yml   + the pipeline (jobs per §3, on the chosen platform D1)
docs/dev_run.md            ~ "Continuous integration" section + changelog entry
README.md                  ~ CI badge + one-line "what runs on push" note
```

No application code changes — this step only adds automation. (If **D3 = GHCR**, no repo
secrets are needed; if **D3 = Alt B**, you add registry credentials as secrets and I
reference them.)

---

## 6. Verification (definition of done)

Because CI runs *on GitHub*, "green" ultimately shows on a real push — but I verify as much
as possible locally and by construction:

- **Locally reproduce each job's commands** and confirm they pass: `gofmt -l` (empty),
  `go vet ./...`, `go test -race ./...`, `go test -race -tags=integration ./...` (Docker
  available here), `go mod tidy` (no diff), and the full web sequence incl. the `gen:api`
  drift check and `docker build`. This proves the workflow's steps are correct even before
  a remote exists.
- **Workflow lint:** validate `ci.yml` is well-formed (YAML parse + `actionlint` if
  available) so it won't fail to start.
- **Honest caveat:** I can't observe an actual Actions run without a GitHub remote, so I'll
  state that explicitly — the workflow is verified by locally mirroring every step and by
  static validation, not by a green checkmark I produced. Once you push to GitHub, the first
  run confirms it end-to-end (and I can iterate if a runner-specific issue appears).
- Docs updated. **Nothing committed.**

---

On approval I add the workflow on the chosen platform (D1–D4), reproduce every job locally,
validate the YAML, update docs, and report back for your **"Task completed."** Provisioning
an actual host + TLS remains the next Phase-C step whenever you want it.
