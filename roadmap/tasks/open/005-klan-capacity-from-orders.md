# 005 — Source klan.RequestedMemberCount from order projection

**Status:** open
**Priority:** low
**Created:** 2026-06-04

## Description

`nathejk/table/klan/query.go::RequestedMemberCount` currently sums `klan.reservedMemberCount` (legacy projection). It would be cleaner to source it from `SUM(order_line.quantity) WHERE productSku='participation.klan'` so the single order projection drives both the klan capacity gate and the order commander's `checkStock`. In steady state both sources track each other; the migration is low-priority.

Related files:
- `go/nathejk/table/klan/query.go` — `RequestedMemberCount`
- `go/nathejk/table/klan/commands.go` — `capacity()`
- `go/nathejk/table/order/querier.go` — `ReservedQuantity`

## Acceptance Criteria

- [ ] `RequestedMemberCount` reads from order_line (not klan.reservedMemberCount)
- [ ] Klan capacity gate and order checkStock use the same data source
- [ ] No behaviour change for existing flows (reservations, waitlisting)

## Progress Log

- 2026-06-04 21:54 — Task created.
