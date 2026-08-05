# 002 — NathejkOrderPaid saga timing

**Status:** open
**Priority:** low
**Created:** 2026-06-04

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

- [ ] Saga reliably transitions orders to `paid` even if the payment projection lags by several seconds
- [ ] Timing behaviour is covered by a unit test using the `settle` seam
- [ ] No N×2s startup delay when JetStream replays old events (this is task 006 — decide whether to merge)

## Progress Log

- 2026-06-04 21:54 — Task created.
- 2026-08-04 — Refreshed to reflect reality: `NewSaga` gained a `settle`
  parameter (a test seam that did not exist at creation), the saga is now a
  `cqrs.Consumer` headed for `shared-go`, and the "read amount from the body"
  idea was qualified — it removes one projection read but not all of them for
  multi-payment orders. Corrected the related-files list (the mirrored 2s wait
  is in `cmd/api/payment.go`) and added a test acceptance criterion. No code
  change.
