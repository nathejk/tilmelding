# 002 — NathejkOrderPaid saga timing

**Status:** done
**Priority:** low
**Created:** 2026-06-04
**Completed:** 2026-08-04

## Description

`nathejk/table/order/saga.go` waits `DefaultSagaSettle` (2s) between receiving a
payment event and reading the projection. This matches the existing pattern in
`mobilepayCallbackHandler` but is a heuristic. Under heavy load or a cold
projection catch-up it could fire too early, miss the payment amount, and leave
the order in `open` until the next event.

Possible improvements:
- Retry loop (read up to N times with backoff until `paidAmount >= totalAmount`
  or N attempts) instead of a single fixed sleep.
- "Check now, schedule re-check" mechanism.
- Reduce reliance on the payment projection: the `NathejkPaymentReceived` body
  already carries `Amount`, so the *just-received* payment need not be read
  back from the projection. Note this only removes one input — the transition
  still needs the order's `totalAmount` and any *other* payments' amounts, so
  it does not eliminate the projection read entirely for multi-payment orders.

Context added since this task was written (2026-06-04):
- `NewSaga` now takes a `settle time.Duration` parameter (pass 0 for the
  default). This is a test seam — a unit test can inject `settle=0` and drive
  `HandleMessage` directly, so the timing behaviour can be pinned down without
  waiting on wall-clock delays. There are currently no saga tests; adding them
  belongs with this work.
- The saga is now typed as `cqrs.Consumer` (`github.com/jrgensen/cqrs`) and is
  a candidate for extraction to `shared-go`; keep any fix free of `cmd/api`
  dependencies.

Related files:
- `go/nathejk/table/order/saga.go`
- `go/cmd/api/payment.go` — `mobilepayCallbackHandler`, the 2s wait this mirrors

## Acceptance Criteria

- [x] Saga reliably transitions orders to `paid` even if the payment projection lags by several seconds
- [x] Timing behaviour is covered by a unit test using the `settle` seam
- [x] No N×2s startup delay when JetStream replays old events (this is task 006 — decide whether to merge)

## Progress Log

- 2026-06-04 21:54 — Task created.
- 2026-08-04 — Refreshed to reflect reality: `NewSaga` gained a `settle`
  parameter (a test seam that did not exist at creation), the saga is now a
  `cqrs.Consumer` headed for `shared-go`, and the "read amount from the body"
  idea was qualified — it removes one projection read but not all of them for
  multi-payment orders. Corrected the related-files list (the mirrored 2s wait
  is in `cmd/api/payment.go`) and added a test acceptance criterion. No code
  change.
- 2026-08-04 — Done. Replaced the single fixed sleep-then-read with a
  read-first bounded retry: `HandleMessage` reads up to `attempts`
  (`DefaultSagaAttempts` = 5) times, transitioning as soon as the order shows
  fully paid, and waiting `settle/attempts` between reads only when live.
  Consequences: an order whose projection is already current transitions
  immediately (no up-front wait, which the old code always paid); a lagging
  projection is tolerated within the budget; a genuinely under-paid order
  exhausts the attempts and stays open (safe). Extracted the read/decide into
  `attemptTransition`, returning a retry flag so only "open but not yet fully
  paid" is retried — unknown/legacy reference, already-paid, cancelled and free
  orders are terminal.
  This supersedes the always-sleep timing that task 006 introduced; updated
  `saga_test.go` accordingly (retry-until-caught-up, give-up-after-max,
  immediate-when-already-paid, replay-never-waits).
  Left the `GetByReference` not-found path as a terminal no-op rather than
  retrying it: we react to a `payment.received`, so the row should already
  exist from `.requested`/`.reserved`; the real lag is in the joined
  `paidAmount`, which the retry covers. Did not merge with 006 — both are now
  done, so the overlap is moot.
