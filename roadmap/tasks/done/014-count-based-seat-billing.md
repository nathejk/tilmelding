# 014 — Count-based billing for participation and t-shirts (seats)

**Status:** done
**Priority:** high
**Created:** 2026-07-31
**Picked up by:** agent (opus)
**Started:** 2026-07-31
**Completed:** 2026-07-31

## Description

Implements the core billing change from PRD 001
(`roadmap/prd/001-seat-based-billing-and-member-identity.md`).

A team pays for a number of **units** (participation seats and t-shirts), not for
specific people. The current order layer bills participation per active member
and dedupes already-paid items by `(productSku, memberId)` in `PaidLineKeys` /
`SetDerivedLines`. That double-charges a team when the member occupying a paid
seat changes.

Change participation and t-shirt billing to be **count-based**:

- `seatsPaid` = total `participation.*` quantity across the team's **paid**
  orders for the year.
- `tshirtsPaid` = total `tshirt.adult` quantity across the team's **paid**
  orders for the year.
- Open-order participation due = `max(0, activeMemberCount − seatsPaid)`.
- Open-order t-shirt due = `max(0, activeTshirtCount − tshirtsPaid)`.
- A paid t-shirt is reusable across members and its size may change free of
  charge, subject to available stock for the chosen size (existing `checkStock`
  merchandise rule still applies to the chosen size).

Only **paid** orders count toward paid units; cancelled and open orders do not.
Existing paid orders remain immutable — paid units are derived from them, not by
rewriting them.

Related files:
- `go/nathejk/table/order/querier.go` — `PaidLineKeys` (to be replaced/augmented
  with a paid-quantity-per-SKU query)
- `go/nathejk/table/order/commander.go` — `SetDerivedLines` (dedup logic)
- `go/cmd/api/orders.go` — `derivedLinesNeedSync`, `loadOrders`
- `go/cmd/api/patrulje.go` / `go/cmd/api/klan.go` — derived-line helpers

## Acceptance Criteria

- [x] Order layer exposes paid **quantity per SKU** for an owner/year (replacing
      or complementing `PaidLineKeys`).
- [x] `SetDerivedLines` bills participation and t-shirts by count: due =
      max(0, activeCount − paidCount) per SKU, instead of memberId dedup.
- [x] Deleting a member and adding another within the paid seat count yields 0
      additional participation due.
- [x] Changing a t-shirt size (or reassigning a paid t-shirt to a new member)
      yields 0 additional t-shirt due, subject to stock for the chosen size.
- [x] Only paid orders contribute to paid counts; cancelled/open do not.
- [x] `go build` and `go vet` clean.

## Progress Log

- 2026-07-31 10:00 — Task created from PRD 001.
- 2026-07-31 10:10 — Picked up. Plan: add `PaidQuantityBySKU` to order Queries/querier; replace the `(sku, memberId)` `PaidLineKeys` filter in `SetDerivedLines` with a count-based drop (drop up to paidQty[sku] desired lines per SKU); remove now-unused `PaidLineKeys`. Verifies fcc870d1: 4 paid seats → 0 participation due, 4 t-shirts (0 paid) → 700 kr due.
- 2026-07-31 12:20 — Replaced `PaidLineKeys` with `PaidQuantityBySKU` (SUM(quantity) per SKU across paid orders) in `order/querier.go`. Extracted a pure exported `order.ApplyPaidOffset(desired, paidQty)` used by `SetDerivedLines`.
- 2026-07-31 12:25 — Found the show-path self-heal wouldn't fire: `derivedLinesNeedSync` compared current lines against the *unfiltered* desired, which matched (4+4), so `SetDerivedLines` was never called on already-populated orders. Fixed by making it a method `app.derivedLinesNeedSync` that fetches paidQty and applies `ApplyPaidOffset` before comparing, so it compares against the set that will actually be persisted. Updated the 4 show handlers (patrulje/klan/personnel/crew).
- 2026-07-31 12:30 — Updated stale `PaidLineKeys` comments in `migrate_orders.go`. build + vet + staticcheck clean; api hot-reloaded.
- 2026-07-31 12:35 — ✅ Verified against live dev data: GET /api/patrulje/fcc870d1… re-derived its open order to totalAmount=70000 (4× tshirt only, 0 participation). The 4 paid seats are recognized despite the roster's member IDs having all changed. All criteria met.
- 2026-07-31 12:36 — Completed. Participation and t-shirts now billed by paid-unit count; swapped members / size changes no longer re-charge. Comprehensive tests deferred to task 021.
