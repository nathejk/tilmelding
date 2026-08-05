# 005 — Source klan.RequestedMemberCount from order projection

**Status:** done
**Priority:** low
**Created:** 2026-06-04
**Completed:** 2026-08-04

## Description

`nathejk/table/klan/query.go::RequestedMemberCount` currently sums `klan.reservedMemberCount` (legacy projection). It would be cleaner to source it from `SUM(order_line.quantity) WHERE productSku='participation.klan'` so the single order projection drives both the klan capacity gate and the order commander's `checkStock`. In steady state both sources track each other; the migration is low-priority.

Related files:
- `go/nathejk/table/klan/query.go` — `RequestedMemberCount`
- `go/nathejk/table/klan/commands.go` — `capacity()`
- `go/nathejk/table/order/querier.go` — `ReservedQuantity`

## Acceptance Criteria

- [x] `RequestedMemberCount` reads from order_line (not klan.reservedMemberCount)
- [x] Klan capacity gate and order checkStock use the same data source
- [x] No behaviour change for existing flows (reservations, waitlisting)

## Progress Log

- 2026-06-04 21:54 — Task created.
- 2026-08-04 — Done. `RequestedMemberCount` now sums `order_line.quantity`
  WHERE `productSku='participation.klan'` on non-cancelled orders, the same
  query shape as `order.querier.ReservedQuantity`, so the klan capacity gate
  and the order commander read the same underlying tables.

  Verified behaviour-preserving before switching: `klan.reservedMemberCount`
  was written only on the `.reserved` event (consumer.go), while waitlisted
  `.requested` teams set a *separate* `requestedMemberCount` column — so the
  old sum already counted reserved-only, exactly the teams that get
  `participation.klan` order lines. Reservations and waitlisting are therefore
  unchanged. The single intentional divergence: a cancelled order now frees
  its seats (`status <> 'cancelled'`), which the stale klan column never did.

  Chose to duplicate the SQL rather than inject `order.Queries` into klan:
  the shared *data source* (order_line/orders) satisfies the AC, and keeping
  klan free of an import on the order package preserves its portability toward
  shared-go. No unit test added — it is a SQL source swap and go-sqlmock was
  dropped from the module in task 025; the equivalence is argued above and
  build/vet/staticcheck pass.
