# 013 — Paid orders should never have non-zero `dueAmount`

**Status:** done
**Priority:** medium
**Created:** 2026-06-04
**Picked up by:** claude
**Started:** 2026-06-04
**Completed:** 2026-06-04

## Description

The team show endpoints return a `paidOrders` array. Each entry currently has a `dueAmount` property — and in practice some entries surface a **non-zero** `dueAmount`, which is incoherent: a paid order is by definition fully covered.

Symptom in the UI: the "Tidligere betalte ordrer" section sums `po.paidAmount` to display each row's amount, but the cumulative "orders completed" total can come out as **zero** when the underlying paidAmount is also zero — because the order somehow ended up in `status='paid'` while its joined `paidAmount` is below `totalAmount`.

### Likely causes (to investigate)

1. **Race between saga read and projection state.** The saga publishes `NathejkOrderPaid` based on a `paidAmount >= totalAmount` check at saga-read time, then the projector flips `orders.status='paid'`. If a payment subsequently moves out of the `('reserved','received')` set the JOIN-computed `paidAmount` drops, while `status` is already `paid`. Result: `dueAmount > 0` on a paid order.
2. **Manually set status in dev data.** Someone may have run `UPDATE orders SET status='paid'` for testing without aligning payment rows.
3. **Total mutated after paid.** The commander rejects mutations on non-open orders, so this *shouldn't* happen, but worth verifying nothing in the projector or saga path can update `totalAmount` post-paid.
4. **`dueAmount` computation in `ListByOwner`.** `o.DueAmount = o.TotalAmount - o.PaidAmount` is unconditional; for paid orders we should either clamp it to zero or compute it differently.

### Possible fixes

- **Clamp `dueAmount` to 0 for paid orders** in the read path (`querier.go::scanOrder` / `ListByOwner` / `GetByID`):

  ```go
  if o.Status == StatusPaid {
      o.DueAmount = 0
  } else {
      o.DueAmount = o.TotalAmount - o.PaidAmount
  }
  ```

  Cheapest fix; addresses the wire-shape bug regardless of root cause.

- **And/or** fix the underlying inconsistency:
  - Audit the saga / projector for paths that could leave the two columns out of sync.
  - On the FE, sum `po.totalAmount` (or `Math.max(po.paidAmount, po.totalAmount)`) instead of `po.paidAmount` for paid orders, since "what they paid" is by definition the total once paid.

- **Drop `dueAmount` from paid orders** entirely in the JSON output (custom MarshalJSON or omitting the field for paid status). Less flexible — clamping is preferable.

### Investigation steps

1. Run `SELECT orderId, status, totalAmount FROM orders WHERE status='paid';` and `SELECT orderForeignKey, status, SUM(amount) FROM payment WHERE status IN ('reserved','received') GROUP BY orderForeignKey;` to spot orders where the paid sum disagrees with the order total.
2. Confirm whether MobilePay test mode ever advances payments to `received`. If not, no order should ever be in `status='paid'` legitimately — the data is from manual updates.

### Related files

- `go/nathejk/table/order/querier.go` — `scanOrder`, `ListByOwner`, `GetByID`, `orderColumns`
- `go/nathejk/table/order/saga.go` — paid-transition trigger
- `vue/src/views/PatruljeView.vue` (and `KlanView`/`StaffView`/`FriendView`) — paid-orders rendering
- `vue/src/helpers/order.js` — `totalPaidDkk`

## Acceptance Criteria

- [x] Every paid order in the API response has `dueAmount == 0` (or the field is omitted for paid orders)
- [x] FE "Indbetalt" total correctly reflects what was paid across paid orders
- [x] Investigation step 1 above produces a clean report (no orders with `status='paid'` and `paidAmount != totalAmount`), or the data inconsistency is documented and explained
- [x] If a clamp is applied, regression test or comment notes that the underlying status/payment drift is still possible and how it would surface

## Progress Log

- 2026-06-04 22:35 — Task created.
- 2026-06-04 22:50 — Diagnostic data collected. The one paid order in dev is
  `a283de9f-362a-4215-8c3a-4fdadb39db1a` with `totalAmount=135000`. All
  payments in the `payment` table for 2026 are `status='requested'` (the
  full table was 14 rows, 0 reserved / 0 received). So the saga cannot
  have run — it gates on `paidAmount >= totalAmount` and `paidAmount` is
  computed via a JOIN that filters to `status IN ('reserved','received')`,
  which is the empty set here. Conclusion: the order's `status='paid'`
  was set manually (presumably to test the FE rendering of paid orders,
  since MobilePay test mode never advances payments past `requested`).

  Implication: this is **not** a saga / projector bug. It's a read-model
  consistency bug — once an order is `paid`, the `dueAmount` and
  `paidAmount` fields on the wire should be derived from the order's
  totalAmount, not from the live payment-table JOIN, otherwise any
  drift (manual `UPDATE`, refund, payment row deletion, etc.) leaves
  the wire shape incoherent.

  Recommended fix: clamp paid orders' `paidAmount` and `dueAmount` in
  the read path (`querier.go`):

  ```go
  if o.Status == StatusPaid {
      o.PaidAmount = o.TotalAmount
      o.DueAmount = 0
  } else {
      o.DueAmount = o.TotalAmount - o.PaidAmount
  }
  ```

  This addresses the FE "0 kr." symptom (the paid-orders renderer reads
  `paidAmount` for the row amount) AND the `dueAmount > 0 on paid order`
  wire-shape bug, in one change.
- 2026-06-04 22:42 — Picked up. User confirmed the paid order's status was
  manually flipped to test FE rendering. Applying the clamp fix in
  `querier.go::GetByID` / `FindOpenOrder` / `ListByOwner` (the three
  places that compute DueAmount today).
- 2026-06-04 22:46 — Implemented. Centralised the clamp in `scanOrder`
  rather than duplicating across the three callers: when `Status ==
  StatusPaid`, `PaidAmount` is forced to `TotalAmount` and `DueAmount`
  to `0`; otherwise `DueAmount = TotalAmount - PaidAmount` as before.
  The three call sites (`GetByID`, `FindOpenOrder`, `ListByOwner`) lost
  their now-redundant `o.DueAmount = ...` lines. The clamp comment in
  `scanOrder` explicitly documents the four drift scenarios (manual
  UPDATE, refund, MobilePay cancellation, payment row deletion) so a
  future reader doesn't think the read API is silently masking a saga
  bug. ✅ All four criteria met. `go build` and `go vet` clean. Moving
  to done.
