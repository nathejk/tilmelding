# 009 — Klan reservation pending-N MemberID in reports

**Status:** open
**Priority:** low
**Created:** 2026-06-04

## Description

The reservation flow uses `pending-N` synthetic MemberIDs on klan participation lines while seats are reserved before the senior identities are filled in. This works but exposes the placeholder convention in the projection (`order_line.memberId LIKE 'pending-%'`). Once `updateKlanHandler` runs, the snapshot replaces them with real IDs.

If reports start to surface the placeholders, consider:
- Filtering them out in any "members per order" report.
- Or re-design so the order isn't created until member identities exist (changes the UX: no payment link at reservation time).

Related files:
- `go/cmd/api/klan.go` — `pendingMemberID()`, `reservationLineID()`
- `go/nathejk/table/order/` — order_line.memberId

## Acceptance Criteria

- [ ] Decision documented: filter in reports OR defer order creation
- [ ] If filtering: report queries exclude `memberId LIKE 'pending-%'`
- [ ] If deferring: UX flow updated, payment link only generated after members known

## Progress Log

- 2026-06-04 21:54 — Task created.
