# 021 — Tests: seat reuse, one-event-per-action, idempotency, caps

**Status:** open
**Priority:** medium
**Created:** 2026-07-31

## Description

Regression coverage for PRD 001
(`roadmap/prd/001-seat-based-billing-and-member-identity.md`). Lock in the
behaviours the fix guarantees so they don't regress.

Cover:

- **Seat reuse:** pay for N seats, delete a member, add another → 0 additional
  participation due. Add an (N+1)th → exactly one seat charged.
- **T-shirt reuse / size change:** changing a t-shirt size, or reassigning a paid
  t-shirt to a new member, adds 0 additional due (subject to stock).
- **One event per action:** add emits exactly one create, edit one update, delete
  one delete.
- **Idempotency:** saving an unchanged roster creates no new rows and emits no
  create/delete events.
- **Caps:** add beyond max rejected; pay below min rejected.

Prefer table-driven Go tests at the order + command layer; add handler-level
tests where practical.

Related files:
- `go/nathejk/table/order/*` (billing/seat accounting)
- `go/nathejk/table/patrulje/*`, `go/nathejk/table/klan/*` (commands)
- `go/cmd/api/*` (handlers, caps)

## Acceptance Criteria

- [ ] Seat-reuse test (delete+add = 0 due; +1 member = one seat charged).
- [ ] T-shirt reuse / free size-change test.
- [ ] One-event-per-action tests for add/edit/delete.
- [ ] Idempotency test (unchanged roster → no new rows/events).
- [ ] Cap tests (max on add, min on pay) for patrulje and klan.
- [ ] `go test ./...` passes.

## Progress Log

- 2026-07-31 10:00 — Task created from PRD 001.
