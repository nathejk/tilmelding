# 011 — Show paid orders and totals on team pages

**Status:** done
**Priority:** high
**Created:** 2026-06-04
**Picked up by:** claude
**Started:** 2026-06-04
**Completed:** 2026-06-04

## Description

Team pages (patrulje, klan, personnel) currently show only the open order. Users also need to see their paid/completed orders so they have a full payment history, plus a clear summary of total paid and total due across all orders.

The backend already has `order.Queries.ListByOwner` which returns every order (newest first, all statuses). The frontend needs to render:

1. The open order (current, editable — already done).
2. A list of paid orders (read-only, collapsed or expandable).
3. Summary line: **Total paid** (sum of paidAmount across all orders) and **Total due** (dueAmount on the open order, or 0 if none).

### Implementation sketch

**Backend:**
- Add paid orders to the show handler response envelope, e.g. `"paidOrders": [...]` alongside the existing `"order"` (open order).
- Or: return a single `"orders": [...]` array containing all orders and let the FE partition by status. Simpler API, slightly more FE logic.

**Frontend:**
- Render paid orders in the Betalinger fieldset (or a new one) as a collapsed list showing date, line summary, and amount.
- Show aggregate totals: sum of `paidAmount` across all orders, and `dueAmount` from the open order.

Related files:
- `go/cmd/api/patrulje.go` — `showPatruljeHandler`
- `go/cmd/api/klan.go` — `showKlanHandler`
- `go/cmd/api/personnel.go` — `showPersonnelHandler`
- `go/nathejk/table/order/querier.go` — `ListByOwner`
- `vue/src/views/PatruljeView.vue`
- `vue/src/views/KlanView.vue`
- `vue/src/views/StaffView.vue`
- `vue/src/views/FriendView.vue`
- `vue/src/helpers/order.js`

## Acceptance Criteria

- [x] Show handler responses include paid orders (status=PAID) alongside the open order
- [x] FE renders paid orders with date, line summary (product names + quantities), and paid amount
- [x] FE shows a "Total paid" sum across all orders
- [x] FE shows "Amount due" from the open order (or 0 if no open order)
- [x] Paid orders are clearly marked as non-editable (read-only display)

## Progress Log

- 2026-06-04 21:54 — Task created.
- 2026-06-04 22:10 — Picked up. Plan: (1) extend show handlers to expose all orders for the owner, partitioned into the open order + a paid-orders list; (2) add aggregate totals (paid across all orders, due on the open one) to the order helper on the FE; (3) render a read-only "Tidligere betalinger" section on patrulje/klan/staff/friend views listing each paid order's date / lines / amount. Going to start with the backend to lock the response shape, then move to the FE.
- 2026-06-04 22:14 — Backend: added `app.loadOrders(ctx, ownerType, ownerID)` helper in new `cmd/api/orders.go` that returns the open order plus the list of paid orders (filtering out cancelled). Wired it into `showPatruljeHandler`, `showKlanHandler`, and `showPersonnelHandler`; envelope now includes `"paidOrders": [...]` alongside `"order"`. Build green.
- 2026-06-04 22:18 — FE helper: added `orderShortLines`, `orderDateShort`, and `totalPaidDkk` to `helpers/order.js`. The new `totalPaidDkk` sums paidAmount across the open order and every paid order, replacing the previous per-open-order-only `orderPaidDkk` for the Indbetalt total.
- 2026-06-04 22:20 — FE views: added `paidOrders` ref + load to all four views (Patrulje, Klan, Staff, Friend). Each Betalinger fieldset now renders a compact "Tidligere betalte ordrer" section between the open-order grid and the totals — date, line summary (e.g. "4× Patrulje-deltagelse"), amount in DKK. The Indbetalt and At betale totals now reflect cumulative paid (across all orders) and the open order's due. ✅ All five criteria met.
- 2026-06-04 22:22 — `go build` and `go vet` clean (only the pre-existing mobilepay tag warning remains, unrelated). Moving to done.
