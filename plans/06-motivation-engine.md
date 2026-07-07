# Step 6 — Motivation Engine

> **Phase:** A (Backend) · **Status:** Draft for approval
> **Parent:** [`00-system-plan.md`](./00-system-plan.md) §8.6, §5, §9
> **Goal of this step:** Turn a topic's live progress into an encouraging **motivational
> message**, surfaced on the topic view. This is a small, pure enrichment on top of Step 5:
> a threshold → message mapping, computed (never stored), attached to each topic response.
>
> **Out of scope:** The dashboard's overall message + New-Year countdown (Step 7). This step
> only concerns the per-topic message.

---

## 1. Why this step exists

The product's emotional payoff is that opening a Topic tells you how you're doing in a
human, motivating way — not just a number. Step 5 made progress live; this step maps that
progress ratio to a message. It stays a **pure domain function** (ratio → message), so it is
trivially testable, deterministic, and reused unchanged by the dashboard later.

---

## 2. Outcome / Definition of Done

- A pure `domain` function maps a `Progress` to a motivational message across well-defined
  bands (including the "no goals yet" case).
- `GET /topics/{id}` and `GET /topics` include a `motivation` string on each topic, matching
  that topic's current progress.
- The message updates as goals are completed (e.g. reaching 100% shows the celebratory copy).
- `go build`, `go vet`, `gofmt`, `golangci-lint`, `make test` pass; `make test-int` passes
  against real `mongo:7`, asserting the message reflects progress end-to-end.
- `docs/dev_run.md` updated (response note + changelog).

---

## 3. Design

### 3.1 Bands (thresholds)
Following the master plan's suggested bands, plus an explicit **no-goals** state (a topic with
zero goals is meaningfully different from one with goals none of which are done):

| State | Condition | Meaning |
| --- | --- | --- |
| `NoGoals` | `total == 0` | nothing to do yet |
| `Started` | `ratio == 0` and `total > 0` | goals exist, none done |
| `Quarter` | `0 < ratio ≤ 0.25` | 1–25% |
| `Half` | `0.25 < ratio ≤ 0.50` | 26–50% |
| `Most` | `0.50 < ratio < 1.0` | 51–99% |
| `Complete` | `ratio == 1.0` | 100% |

Boundaries are exact and closed as shown, so every ratio maps to exactly one band.

### 3.2 Default copy (encouraging tone — your call, see §7)
| State | Message |
| --- | --- |
| `NoGoals` | "Set your first goal — every big year starts with one step." |
| `Started` | "The hardest part is starting, and you're already here. Let's go." |
| `Quarter` | "You're on the board. Keep the momentum going." |
| `Half` | "Real progress — you're building something. Keep at it." |
| `Most` | "More than halfway. The finish line is in sight." |
| `Complete` | "Every goal complete — this one's in the books. 🎉" |

The **exact wording and tone are a product decision that's yours** (system plan §9). I'll ask
you to pick a tone (see §7) and you can override any individual line.

### 3.3 Shape
- `domain/motivation.go`: a `MotivationBand` type + `Motivate(p Progress) string` returning
  the message for the band. Copy lives in one table here, so changing it later is a one-file
  edit. One message per band (deterministic); a "message set with rotation" is an easy future
  extension and not needed now.
- `TopicView` (service) gains a `Motivation string`, set from `Motivate(progress)` wherever a
  view is built (Create/Get/List). No new queries — it's a pure function of progress already
  in hand.
- `TopicResponse` (transport) gains a `motivation` JSON field.

---

## 4. Files

```
internal/
  domain/
    motivation.go            # bands + Motivate(Progress) string
    motivation_test.go       # every band + exact boundaries

  service/
    topic_service.go         # CHANGED: TopicView.Motivation, set on Create/Get/List
    topic_service_test.go    # CHANGED: assert motivation matches progress

  transport/http/
    topic_dto.go             # CHANGED: TopicResponse.Motivation
    topic_flow_integration_test.go  # CHANGED: assert motivation present
    goal_flow_integration_test.go   # CHANGED: assert message changes as goals complete

docs/dev_run.md              # CHANGED: note the motivation field; changelog
```

No new endpoints, collections, indexes, or dependencies.

---

## 5. Testing Strategy

| Layer | What | Runs in |
| --- | --- | --- |
| domain | each band incl. exact boundaries (0, 0.25, 0.5, 0.9999, 1.0) and no-goals | `make test` |
| service | `TopicView.Motivation` matches `Motivate(progress)` on create/get/list | `make test` |
| end-to-end | topic detail carries a message; completing all goals flips it to the `Complete` copy | `make test-int` |

## 6. Decisions Within This Step (my discretion — flag to change)

- **Motivation on both detail and list** (cheap pure function; consistent for the SPA).
- **One message per band** (deterministic, easy to test); message-sets/rotation deferred.
- Boundary convention as in §3.1 (upper-closed bands).

## 7. Product Copy — DECIDED: Encouraging & warm

**Locked by you:** the "encouraging & warm" voice — the exact six lines in §3.2 are the copy
to ship:

| State | Message |
| --- | --- |
| `NoGoals` | Set your first goal — every big year starts with one step. |
| `Started` | The hardest part is starting, and you're already here. Let's go. |
| `Quarter` | You're on the board. Keep the momentum going. |
| `Half` | Real progress — you're building something. Keep at it. |
| `Most` | More than halfway. The finish line is in sight. |
| `Complete` | Every goal complete — this one's in the books. 🎉 |

The copy lives in a single table in `domain/motivation.go`, so tweaking any line later is a
one-file change.

## 8. Verification Checklist

1. `make tidy && go build ./...` — clean.
2. `gofmt` / `go vet` (both tags) / `golangci-lint` — clean.
3. `make test` — unit suite green, `-race`.
4. `make test-int` — motivation reflected end-to-end against real `mongo:7`.
5. Live smoke: create a topic (see the no-goals message), add goals and complete them, watch
   the message move through the bands to the celebratory 100% copy. Update `docs/dev_run.md`
   and re-verify.

## 9. What This Unblocks

Step 7 (Dashboard): reuses the same `Progress`/motivation primitives to produce an overall
status message alongside aggregate progress and the New-Year countdown.
