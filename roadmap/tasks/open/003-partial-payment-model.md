# 003 — Partial-payment / overpay edge cases

**Status:** open
**Priority:** low
**Created:** 2026-06-04

## Description

An order's `DueAmount` can go negative if someone overpays (legacy data, refund, etc.). The FE uses `Math.max(0, ...)` for display and the saga correctly transitions `paid` when `paidAmount >= totalAmount`. No bug today, but worth a deliberate model when refunds become a real flow.

Related files:
- `go/nathejk/table/order/querier.go` — DueAmount computation
- `vue/src/helpers/order.js` — `orderDueDkk`

## Acceptance Criteria

- [ ] Design decision documented: how overpay/refund should be represented in the order model
- [ ] If refunds are supported: DueAmount, status transitions, and FE display handle negative due gracefully

## Progress Log

- 2026-06-04 21:54 — Task created.
