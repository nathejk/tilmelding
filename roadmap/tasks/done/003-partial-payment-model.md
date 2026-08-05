# 003 — Partial-payment / overpay edge cases

**Status:** done
**Priority:** low
**Created:** 2026-06-04
**Completed:** 2026-08-04

## Description

An order's `DueAmount` can go negative if someone overpays (legacy data, refund, etc.). The FE uses `Math.max(0, ...)` for display and the saga correctly transitions `paid` when `paidAmount >= totalAmount`. No bug today, but worth a deliberate model when refunds become a real flow.

Related files:
- `go/nathejk/table/order/querier.go` — DueAmount computation
- `vue/src/helpers/order.js` — `orderDueDkk`

## Acceptance Criteria

- [x] Design decision documented: how overpay/refund should be represented in the order model
- [ ] If refunds are supported: DueAmount, status transitions, and FE display handle negative due gracefully

## Decision (2026-08-04)

**Defer.** Nathejk has no refund flow, so the order model deliberately does not
represent credit/refund state. Current behaviour is left as-is and is
acceptable:

- `DueAmount = TotalAmount - PaidAmount` on non-terminal orders, which may go
  negative on overpay; the frontend clamps display with `max(0, due)`
  (`vue/src/helpers/order.js::orderDueDkk`).
- A paid order is clamped to `PaidAmount = TotalAmount`, `DueAmount = 0`
  regardless of payment-table drift (querier.go already does this).
- The saga transitions to `paid` when `PaidAmount >= TotalAmount`, so overpay
  still results in a clean paid order.

**Trigger to revisit:** a real refund/partial-refund flow (e.g. MobilePay
refund, or manual organiser refunds).

**Sketch for then:** introduce an explicit refunded/credit concept rather than
letting `DueAmount` carry the sign — e.g. a `RefundedAmount` on the order,
a `StatusRefunded`/`StatusPartiallyRefunded` transition, and FE display of
credit owed. Decide whether refunds reopen an order or are terminal. The
negative-due path is the seam where this would land.

The second acceptance criterion is intentionally left unchecked: it is
conditional on supporting refunds, which this decision defers.

## Progress Log

- 2026-06-04 21:54 — Task created.
- 2026-08-04 — Decision recorded: defer refund modelling (no refund flow
  exists); tolerate negative `DueAmount` on overpay, clamped by the FE. Added a
  durable comment at the negative-due seam in `order/querier.go` pointing here.
  This is a deliberate, revisitable engineering decision, not a code change to
  behaviour — the conservative default that commits the product to nothing.
