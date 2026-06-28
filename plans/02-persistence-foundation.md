# Step 2 — Persistence Foundation

> **Phase:** A (Backend) · **Status:** Draft for approval
> **Parent:** [`00-system-plan.md`](./00-system-plan.md) §8.2
> **Goal of this step:** Establish the **shared persistence machinery** every future
> repository will build on — domain error types, document/entity conventions, a generic
> Mongo repository base, idempotent index management, Mongo-error translation, and a real
> **testcontainers** integration harness — and prove it works against a live MongoDB.
>
> **Out of scope:** Any real domain entity (`User`, `Topic`, `Goal`) or its repository.
> The first concrete repository (users) is built on this foundation in **Step 3**.

---

## 1. Why this step exists (and the "no empty packages" rule)

Steps 3–5 each add an entity + repository. Without a shared foundation, each would
re-implement ID conversion, timestamps, CRUD boilerplate, duplicate-key handling, and
index registration — divergently. This step builds that foundation **once**.

To honor our "no empty/placeholder packages" rule, the foundation is **validated by a real
integration test** that uses a small *test-only* sample type to exercise every helper against
a live Mongo container. We do **not** ship a fake entity in production code — the sample type
lives only in `_test.go` files. Production code here is generic and genuinely used the moment
Step 3 lands.

---

## 2. Outcome / Definition of Done

- A generic repository base provides `Insert / FindByID / FindOne / Update / Delete / List`
  with `context`, `ObjectID` handling, and automatic timestamps.
- Domain error types exist and are returned by the base repo: not-found → `ErrNotFound`,
  duplicate key → `ErrConflict`.
- Index management is registry-driven and idempotent; `EnsureIndexes` (called at startup)
  creates everything registered, and is safe to run repeatedly.
- A testcontainers harness spins up real MongoDB for integration tests behind the
  `integration` build tag; `make test-int` runs them green (Docker required).
- Unit tests (no Docker) cover error translation and conversion helpers.
- `go build`, `go vet`, `gofmt`, `golangci-lint`, and `make test` (unit) all pass.

---

## 3. Files Created / Changed

```
internal/
  domain/
    errors.go              # sentinel domain errors + helpers (errors.Is friendly)
    errors_test.go         # unit: classification helpers
  platform/mongo/
    mongo.go               # CHANGED: index registry + EnsureIndexes wired to it
    document.go            # base document (ID, timestamps) + hex<->ObjectID helpers
    document_test.go       # unit: id/timestamp conversion
    errors.go              # translate Mongo driver errors -> domain errors
    errors_test.go         # unit: E11000 -> ErrConflict, ErrNoDocuments -> ErrNotFound
    repository.go          # generic Repository[T] base (CRUD on a collection)
    index.go               # IndexSpec type + collection index registration API
    repository_inttest.go  # //go:build integration — exercises base repo on real Mongo
    harness_inttest.go     # //go:build integration — testcontainers Mongo setup/teardown
```

> Naming: integration files use the `_inttest.go` suffix **and** a `//go:build integration`
> tag so they never compile into unit-test runs or the binary. (`make test-int` adds the tag.)

---

## 4. Component Details

### 4.1 Domain errors (`internal/domain/errors.go`)
Sentinel errors usable with `errors.Is`, plus small classifier helpers. These are the
*vocabulary* the transport layer will later map to HTTP status codes (mapping itself is added
when the first HTTP feature lands).

```go
var (
    ErrNotFound   = errors.New("resource not found")
    ErrConflict   = errors.New("resource conflict")   // e.g. unique-key violation
    ErrValidation = errors.New("validation failed")
)
```
- Helpers: `IsNotFound(err)`, `IsConflict(err)` (thin `errors.Is` wrappers) for readable
  call sites.
- Lives in `domain` because these are business-level concepts, not Mongo-specific.

### 4.2 Base document (`internal/platform/mongo/document.go`)
- `Document` struct embedded by every BSON model:
  ```go
  type Document struct {
      ID        primitive.ObjectID `bson:"_id,omitempty"`
      CreatedAt time.Time          `bson:"createdAt"`
      UpdatedAt time.Time          `bson:"updatedAt"`
  }
  ```
- Helpers: `HexFromID(primitive.ObjectID) string`, `IDFromHex(string) (primitive.ObjectID,
  error)` (invalid hex → wrapped `ErrValidation`), and timestamp stamping used by the base repo.
- **Clock injection:** a `Clock` func type (default `time.Now`) so timestamp behavior is
  deterministic in tests.

### 4.3 Mongo-error translation (`internal/platform/mongo/errors.go`)
- `translate(err) error`:
  - `mongo.ErrNoDocuments` → `domain.ErrNotFound`
  - write error with code `11000` (duplicate key) → `domain.ErrConflict`
  - otherwise → the original error wrapped with context.
- Keeps Mongo specifics from leaking past the persistence layer.

### 4.4 Generic base repository (`internal/platform/mongo/repository.go`)
- `Repository[T any]` wrapping a `*mongo.Collection`. **Generics** chosen to eliminate
  per-entity CRUD boilerplate while staying type-safe. (Small technical decision — flag if
  you'd prefer hand-written repos per entity instead.)
- Methods (all `context`-first, all funnel errors through `translate`):
  - `Insert(ctx, *T) error` — stamps `CreatedAt/UpdatedAt`, sets `_id` if empty.
  - `FindByID(ctx, id string) (*T, error)`
  - `FindOne(ctx, filter bson.M) (*T, error)`
  - `Update(ctx, id string, update bson.M) error` — bumps `UpdatedAt`.
  - `Delete(ctx, id string) error` — hard delete; soft-delete/archival is handled per-entity
    later (e.g. Topics' `archivedAt`), not forced here.
  - `List(ctx, filter bson.M, opts...) ([]T, error)`
- Concrete entity repositories (Step 3+) **embed or wrap** `Repository[T]` and add only their
  entity-specific queries. This is the expandability lever: new entity = new thin wrapper.

### 4.5 Index management (`internal/platform/mongo/index.go` + `mongo.go`)
- `IndexSpec` describes one index (keys, unique, TTL, name).
- A registry: collections register their specs (`RegisterIndexes(collection string, specs
  ...IndexSpec)`); `Client.EnsureIndexes(ctx)` iterates the registry and applies them via
  `Indexes().CreateMany`, which is idempotent (existing identical indexes are no-ops).
- Step 2 registers **no** indexes itself (no collections yet); it delivers the mechanism.
  Step 3 registers `users.email unique`, the sessions TTL index, etc.
- Integration test proves: registering a unique index then inserting a duplicate yields
  `ErrConflict`.

### 4.6 Testcontainers harness (`harness_inttest.go`)
- Uses `github.com/testcontainers/testcontainers-go` (+ the `mongodb` module) to start a real
  `mongo:7` container, returning a connected `*Client`/`*mongo.Database` and a cleanup func.
- Per-test isolation: each test uses a uniquely named database (or drops collections on
  cleanup) so tests don't interfere.
- Guarded by `//go:build integration` so the default `make test` (and CI without Docker)
  never requires Docker; `make test-int` opts in.

---

## 5. Testing Strategy

| Layer | Tooling | Runs in |
| --- | --- | --- |
| Error translation, id/timestamp helpers, error classifiers | stdlib `testing` + `testify` | `make test` (no Docker) |
| Base repo CRUD, not-found, duplicate→conflict, index creation, list filtering | testcontainers (real `mongo:7`) | `make test-int` (Docker) |

The integration test is the proof the whole foundation is sound on a real database — this is
the substitute for the manual Mongo check we couldn't fully automate in Step 1.

---

## 6. Dependencies Added (go.mod)

- `github.com/testcontainers/testcontainers-go` and its `modules/mongodb` (test-only; pulled
  in under the integration build tag).
- No new production dependencies beyond the existing official Mongo driver.

---

## 7. Decisions Within This Step (my discretion — flag to change)

- **Generics** for the base repository (vs. hand-written repos per entity).
- Integration tests gated by `//go:build integration` + `_inttest.go` suffix.
- Hard delete in the base repo; soft-delete/archival left to individual entities.
- `Clock` injection for deterministic timestamps.
- Sample type used to validate the foundation lives **only** in test files.

## 8. Open Questions Deferred to You (not blocking)

- **Optimistic concurrency / document versioning** (a `version` field bumped on update to
  detect lost updates): worth it now, or add when a feature needs it? My recommendation:
  **defer** until a concurrent-edit scenario actually exists.
- These do not change the system plan; raising for visibility only.

---

## 9. Verification Checklist (how I'll prove it works)

1. `make tidy && go build ./...` — clean.
2. `gofmt`/`go vet` — clean (and `golangci-lint` locally on your side).
3. `make test` — unit tests green, `-race`, **no Docker needed**.
4. `make test-int` — integration tests green against real `mongo:7` (Docker needed). I can run
   this here only if Docker is available in my environment; **as in Step 1, the live-Mongo
   integration run is something you may need to execute on your machine.** I will state clearly
   which checks I ran vs. which I could not.
5. I'll paste command output as evidence.

---

## 10. What This Unblocks

Step 3 (Auth & users) builds the first real entity on this foundation: a `User` document +
a `UserRepository` wrapping `Repository[User]`, `users.email` unique index registered, and
sessions with a TTL index — with the integration harness ready to test it all.
