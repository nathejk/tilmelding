# 016 — Klan/senior member endpoints (add/edit/delete)

**Status:** doing
**Priority:** high
**Created:** 2026-07-31
**Picked up by:** agent (opus)
**Started:** 2026-07-31

## Description

Klan/senior equivalent of task 015, per PRD 001
(`roadmap/prd/001-seat-based-billing-and-member-identity.md`). Klan has the same
churn: the roster is saved through a single `PUT /api/klan/{id}` and members are
re-minted on every save.

Add explicit, additive member endpoints for klan (senior members):

- `POST /api/klan/{teamId}/member` — issue `memberId`, one create event, enforce
  max (task 018), return created member.
- `PUT /api/klan/{teamId}/member/{memberId}` — one update event, never creates.
- `DELETE /api/klan/{teamId}/member/{memberId}` — one delete event.

Notes:
- Klan's `reservedMemberCount` is legacy naming for "buying X seats" — the seat
  accounting is the same as patrulje (paid participation quantity). See task 014.
- Verify the `senior` projector shares the spejder projector's `INSERT IGNORE` +
  `memberId` PK behavior; adjust if it differs so add→insert, update→update,
  delete→delete hold.

All new endpoints must have OpenAPI annotations.

Related files:
- `go/cmd/api/klan.go` — new handlers
- `go/cmd/api/main.go` — route registration
- `go/nathejk/table/klan/commands.go` — split create vs. update vs. delete
- `go/nathejk/table/senior/` (or wherever the senior projector lives) — confirm
  parity

## Acceptance Criteria

- [ ] `POST /api/klan/{teamId}/member` issues a `memberId`, emits one create
      event, returns the created member with its id.
- [ ] `PUT /api/klan/{teamId}/member/{memberId}` emits one update event, never
      creates.
- [ ] `DELETE /api/klan/{teamId}/member/{memberId}` emits one delete event.
- [ ] Senior projector confirmed to insert/update/delete by `memberId`.
- [ ] Member add/delete re-derives the open order (count-based).
- [ ] OpenAPI annotations added for all three endpoints.
- [ ] `go build` and `go vet` clean.

## Progress Log

- 2026-07-31 10:00 — Task created from PRD 001.
- 2026-07-31 13:05 — Picked up. Plan: mirror task 015 for klan/senior — add AddMember/UpdateMember/DeleteMember to klan commands (senior events), add POST/PUT/DELETE /api/klan/{id}/member[/{memberId}] handlers re-deriving the order, register routes, OpenAPI annotations. Confirming senior projector parity first.
