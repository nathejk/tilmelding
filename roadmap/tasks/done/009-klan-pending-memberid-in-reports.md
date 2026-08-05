# 009 — Klan reservation pending-N MemberID in reports

**Status:** done
**Priority:** low
**Created:** 2026-06-04
**Completed:** 2026-08-04

## Description

The reservation flow uses `pending-N` synthetic MemberIDs on klan participation lines while seats are reserved before the senior identities are filled in. This works but exposes the placeholder convention in the projection (`order_line.memberId LIKE 'pending-%'`). Once `updateKlanHandler` runs, the snapshot replaces them with real IDs.

If reports start to surface the placeholders, consider:
- Filtering them out in any "members per order" report.
- Or re-design so the order isn't created until member identities exist (changes the UX: no payment link at reservation time).

Related files:
- `go/cmd/api/klan.go` — `pendingMemberID()`, `reservationLineID()`
- `go/nathejk/table/order/` — order_line.memberId

## Acceptance Criteria

- [x] Decision documented: filter in reports OR defer order creation
- [ ] If filtering: report queries exclude `memberId LIKE 'pending-%'`
- [ ] If deferring: UX flow updated, payment link only generated after members known

## Decision (2026-08-04)

**Keep the placeholder approach; filter in reports if/when needed. Do not defer
order creation.**

Deferring order creation until senior identities exist would remove the
reservation-time payment link — a team could not pay to hold its seats until
every member was entered, which is a worse UX and the reason the `pending-N`
convention exists.

No report surfaces the placeholders today: the `pending-N` IDs live on
`order_line.memberId` and appear in the order detail JSON during the
reservation window, but the frontend aggregates klan participation lines by
SKU rather than listing per-member, and `updateKlanHandler` replaces them with
real senior IDs as soon as members are entered.

Guidance recorded for the future: any "members per order" report built off
`order_line.memberId` must exclude `memberId LIKE 'pending-%'`. The recognisable
prefix exists precisely to make that filter trivial. The two conditional
criteria below are left unchecked because neither branch is triggered yet.

## Progress Log

- 2026-06-04 21:54 — Task created.
- 2026-08-04 — Decision recorded: keep placeholders, filter-if-needed, do not
  defer order creation. Verified no current report surfaces the IDs (klan lines
  are aggregated by SKU for display). Added a durable comment on
  `pendingMemberID` in `cmd/api/klan.go` stating the decision and the
  `NOT LIKE 'pending-%'` filter to apply if a per-member report is ever added.
  No behaviour change.
