# Step C2 — CI/CD pipeline (GitHub Actions)

> **Status:** Approved and implemented (2026-11-08); waiting on the first real Actions run. See §8.
> **Parent:** Phase C (production). Follows [`07-production-serving.md`](07-production-serving.md).
> Followed by [`09-self-hosting-raspberry-pi.md`](09-self-hosting-raspberry-pi.md) (C3).
> **Builds on:** the full repo — Go backend (Steps 1–9), the SPA (B1–B6), and the
> single-origin embed build (C1) — all green locally.
> **Approval gate:** No files are written until you approve this document.
> **Revision note (2026-11-07):** the first draft assumed no git remote and a `main` branch. The repo
> now lives on GitHub (`YehorBoldarev/ThisIsYourYear`, **public**), the production branch is
> **`master`**, and the golangci-lint config must move to the v2 format. Platform is locked to GitHub.
> **Revision note (2026-11-08):** the host is now chosen: a **Raspberry Pi 5 (arm64)**. The image is
> therefore published for **both `linux/amd64` and `linux/arm64`** (D7), cross-compiled so the build
> stays fast. §7 now describes the concrete Pi hand-off.

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
- **Multi-arch image** (D7): one image tag that works on a normal PC/server (`amd64`) and on the
  Raspberry Pi (`arm64`). The `Dockerfile` cross-compiles instead of emulating.
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
- **Architecture:** the `Dockerfile` hardcodes no architecture and all three base images
  (`node`, `golang`, distroless `static`) are multi-arch, so it already builds *natively* on the Pi.
  What's missing is a published `arm64` image: a plain `docker build` on the amd64 CI runner
  produces amd64 only.

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
                          platforms: linux/amd64, linux/arm64 (one multi-arch manifest)
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

### Multi-arch build (D7)
- **Cross-compile, don't emulate.** The two build stages run on the runner's own architecture
  (`FROM --platform=$BUILDPLATFORM …`):
  - the **web** stage builds once — its output (HTML/JS/CSS) is architecture-independent;
  - the **Go** stage compiles for the requested platform with `GOOS=$TARGETOS GOARCH=$TARGETARCH`
    (the build is already `CGO_ENABLED=0`, so cross-compiling needs nothing extra).
  - The runtime stage only `COPY`s the binary onto the matching distroless image, so no QEMU
    emulation is needed and the arm64 build costs seconds, not the ~10× of an emulated build.
- **Same `Dockerfile` everywhere:** a plain `docker build .` (no buildx flags) still works and still
  builds natively on the Pi — useful before the first CI image exists.
- **PRs** build both platforms too (no push), so an arm64-only breakage is caught before merge.
- **Not added:** native `ubuntu-24.04-arm` runners (free for public repos) — unnecessary while
  cross-compiling works; worth revisiting only if a cgo dependency ever appears.

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
| D7 | **Multi-arch image** (`linux/amd64` + `linux/arm64`) via cross-compilation (§3) — required by the Raspberry Pi host (C3) | Recommended |
| — | golangci-lint v2 migration (§3) | **Locked** (you asked for it) |

---

## 5. Files (created + / changed ~)

```
.github/workflows/ci.yml     + the pipeline (§3)
.github/dependabot.yml       + Actions updates, weekly (D6)
.golangci.yml                ~ v2 format (§3)
Makefile                     ~ pinned GOLANGCI_LINT_VERSION; `lint` runs it
Dockerfile                   ~ node:24 (D5); build stages on $BUILDPLATFORM, Go cross-compiles to
                               $TARGETOS/$TARGETARCH (D7)
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
- **Multi-arch:** `docker buildx build --platform linux/amd64,linux/arm64 .` succeeds locally; the
  arm64 binary is confirmed to be an aarch64 executable (`file` on the extracted binary); a plain
  `docker build .` still works. After the first published run,
  `docker buildx imagetools inspect ghcr.io/yehorboldarev/thisisyouryear:latest` lists both
  platforms, and `docker pull` on the Pi picks arm64 automatically.
- **Workflow lint:** `actionlint` over `ci.yml` and a YAML check of `dependabot.yml`.
- **Real run:** the definitive check is the first Actions run after you push. I don't push; once
  you do, I can read the run and fix anything runner-specific.
- **After the first green run (your repo settings):** Settings → Branches → add a rule for `master`
  requiring the `backend`, `backend-integration` and `frontend` checks; and Packages → the image →
  change visibility to public.
- Docs updated. **Nothing committed.**

---

## 7. How hosting plugs in (C3 — Raspberry Pi)

C2 ends at the published image; it adds **no `deploy` job**. The host is a Raspberry Pi on a home
network, so deployment is **pull-based**: the Pi periodically pulls the new image from GHCR and
restarts itself (designed in [`09-self-hosting-raspberry-pi.md`](09-self-hosting-raspberry-pi.md)).

- **Why not push from GitHub:** GitHub's runners can't reach a Pi behind a home router without
  opening SSH to the internet, and a self-hosted runner on the Pi is unsafe for a **public** repo
  (anyone's PR could run code on your home network).
- **What C3 needs from C2:** the multi-arch image (D7) and the package made **public** (§6), so the
  Pi pulls without credentials.
- **Config stays on the host:** `JWT_SECRET`, the invite code and (later) the tunnel token live in
  a `.env` on the Pi — never in the repo or the workflow. Rollback = run a previous `:sha-…` tag.

---

## 8. Implementation notes (2026-11-08)

Implemented as designed, with these additions found during verification:

- **Two lint/vet passes.** `-tags=integration` covers the default files plus the integration tests,
  and `-tags=embedui` covers the embedded-SPA file (`fs_noembed.go` is `!embedui`). Together they
  cover every Go file. CI calls `golangci-lint-action` twice; `make vet`/`make lint` mirror it.
- **No exclusion presets.** `golangci-lint migrate` carried over v1's implicit exclusions as four
  presets. Without them v2 reported only two missing doc comments, which were fixed, so the presets
  were dropped and revive's `exported` rule is really enforced. The refresh cookie's
  config-driven `Secure` flag (gosec G124) is a documented inline `//nolint`.
- **`go.mod` `ignore ./web/node_modules`.** An npm package (`flatted`) ships a `.go` file that
  `./...` picked up locally, so `go vet`, `go test` and the linter were scanning it. `make fmt`
  now uses `go fmt ./...`, which respects the directive; `gofmt -w .` doesn't.
- **Dockerfile copies `web/.npmrc`.** `npm ci` failed in the image because `legacy-peer-deps`
  (needed for TypeScript 6 vs openapi-typescript's TS 5 peer) wasn't present. This bug predates C2.
- **New `.dockerignore`.** It keeps `.git`, `.env` files and the host's `node_modules`/build output
  out of the build context. Before, `COPY web/ ./` would overwrite the image's `npm ci` result with
  host binaries.
- **Workflow hygiene.** Per-job `timeout-minutes`, `persist-credentials: false` on checkout, and
  `flavor: latest=false` so a version tag never moves `:latest`.
