# 006 — Event replay safety for the saga

**Status:** open
**Priority:** low
**Created:** 2026-06-04

## Description

If JetStream replays old `payment.received` events on restart (depending on consumer ack config), the saga sleeps 2s per event — N×2s startup delay for N replayed events.

The codebase already exposes the `streaminterface.CatchupListener` interface; implement `CaughtUp()` on the saga and skip the sleep during replay (the projection is fully populated by then).

Related files:
- `go/nathejk/table/order/saga.go`
- `go/superfluids/streaminterface/stream.go` — `CatchupListener` interface

## Acceptance Criteria

- [ ] Saga implements `CaughtUp()` from `streaminterface.CatchupListener`
- [ ] During replay/catch-up phase the 2s sleep is skipped
- [ ] After catch-up, normal settle delay resumes for live events

## Progress Log

- 2026-06-04 21:54 — Task created.
