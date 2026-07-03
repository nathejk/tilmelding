# 002 — NathejkOrderPaid saga timing

**Status:** open
**Priority:** low
**Created:** 2026-06-04

## Description

`nathejk/table/order/saga.go` waits `DefaultSagaSettle` (2s) between receiving a payment event and reading the projection. This matches the existing pattern in `mobilepayCallbackHandler` but is a heuristic. Under heavy load or a cold projection catch-up it could fire too early, miss the payment amount, and leave the order in `open` until the next event.

Possible improvements:
- Retry loop (read up to N times with backoff until paid >= total or N attempts).
- "Check now, schedule re-check" mechanism.
- Read amount directly from the message body rather than relying on the projection JOIN.

Related files:
- `go/nathejk/table/order/saga.go`

## Acceptance Criteria

- [ ] Saga reliably transitions orders to `paid` even if the payment projection lags by several seconds
- [ ] No N×2s startup delay when JetStream replays old events (see also task 006)

## Progress Log

- 2026-06-04 21:54 — Task created.
