# 006 — Event replay safety for the saga

**Status:** done
**Priority:** low
**Created:** 2026-06-04
**Completed:** 2026-08-04

## Description

If JetStream replays old `payment.received` events on restart, the saga sleeps
`DefaultSagaSettle` (2s) per event in `HandleMessage` — an N×2s startup delay
for N replayed events. The sleep exists to let the payment projection catch up
before the saga reads it; during replay the projection is already fully
populated, so the sleep is pure waste.

The original approach — implement `CaughtUp()` on the saga and skip the sleep
until it fires — is correct, and as of `jrgensen/stream` **v0.1.2** it actually
works. Two things had made it look impossible, both now resolved:

- The task first named `go/superfluids/streaminterface/stream.go`. That package
  is gone; `CatchupListener` / `CaughtUp()` now live in
  `github.com/jrgensen/stream`.
- Calling `CaughtUp()` was **broken in stream v0.1.1**: the jetstream
  `Subscribe` path did no catch-up tracking at all, so a handler implementing
  `CatchupListener` was never notified. v0.1.2 fixes this — jetstream now
  tracks each ordered consumer it creates and calls `CaughtUp()` once the
  backlog that existed at subscribe time has drained (detected via
  `NumPending == 0`).

How it reaches the saga: `xstream.Mux` subscribes each consumer as its own
handler (`stream.Subscribe(consumer.Consumes(), consumer)`), and the jetstream
layer type-asserts that handler to `stream.CatchupListener` at runtime. So it
is enough for the concrete `*saga` to grow a `CaughtUp()` method — the static
`cqrs.Consumer` type it is returned as is irrelevant, and `cqrs` need not
re-export anything.

Sketch:
- Add an `atomic.Bool` "live" field to `saga`.
- `CaughtUp()` sets it.
- `HandleMessage` skips `time.Sleep(s.settle)` while it is false (replay) and
  keeps the settle delay once true (live events, where the projection may
  genuinely lag).

This keeps the fix entirely inside the saga — no `cmd/api` wiring — which
matters because the saga is typed as `cqrs.Consumer` and is a candidate for
extraction to `shared-go` (tasks 022/023/025); it must not depend on `cmd/api`.

Note: the bespoke `catchupDetector` in `cmd/api/catchup.go` is a *different*
catch-up signal (it drives one-shot dead-letter reporting for the whole
process) and does not fan out to consumers. It is not the mechanism for this
task and should be left alone.

Related files:
- `go/nathejk/table/order/saga.go` — the `time.Sleep(s.settle)` in `HandleMessage`
- `github.com/jrgensen/stream` — `CatchupListener` / `CaughtUp()`, and the
  jetstream `Subscribe` catch-up tracking fixed in v0.1.2 (external)

See also task 002: its second acceptance criterion ("no N×2s startup delay on
replay") is exactly this task. Consider merging them.

## Acceptance Criteria

- [ ] The concrete `saga` implements `CaughtUp()` (from `stream.CatchupListener`)
- [ ] During replay/catch-up the saga's per-event settle sleep is skipped
- [ ] After catch-up, the normal settle delay resumes for live events
- [ ] The fix lives entirely in `nathejk/table/order`; the saga gains no import
      of `cmd/api`
- [ ] Covered by a unit test (the `settle` seam and a direct `CaughtUp()` call
      make this possible without a live broker)

## Progress Log

- 2026-06-04 21:54 — Task created.
- 2026-08-04 — Rewritten to reflect reality. Original description was stale:
  `superfluids/streaminterface` no longer exists (now `jrgensen/stream`), and
  the saga is now typed as `cqrs.Consumer`. Crucially, verified against the
  bumped dependency that calling `CaughtUp()` was **broken in stream v0.1.1**
  and is **fixed in v0.1.2** (jetstream `Subscribe` now honours
  `CatchupListener`). Since `xstream.Mux` subscribes each consumer as its own
  handler, implementing `CaughtUp()` on the concrete `*saga` is now sufficient
  and self-contained — the original approach, viable again. No code change
  here; the dependency bump that unblocks it is a separate commit.
- 2026-08-04 — Done. Added a `live atomic.Bool` set by `CaughtUp()`;
  `HandleMessage` skips the settle wait while `live` is false (replay).
  Wiring is automatic — the mux subscribes the saga as its own handler and
  jetstream v0.1.2 type-asserts `CatchupListener`, so `main.go` is unchanged.
  Kept the order package free of a direct `stream` import (the 022 invariant):
  the method presence is locked with a local
  `var _ interface{ CaughtUp() } = (*saga)(nil)` rather than a
  `stream.CatchupListener` assertion. Added a `sleep` test seam and
  `saga_test.go` (5 cases): asserts the consumer implements `CaughtUp()`,
  that the settle wait is skipped during replay and taken once live, and that
  the fully-paid/open transition and its guards behave.
  Two caveats recorded honestly: (1) this speeds startup but does not make
  replay race-free against the independently-replaying payment/order
  projectors — that hardening is task 002's bounded-retry loop; (2) correctness
  depends on stream ≥ v0.1.2 calling `CaughtUp()`; if it never fires the saga
  stays in no-settle mode, which is why the v0.1.2 bump preceded this.
