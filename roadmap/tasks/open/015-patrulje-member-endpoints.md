# 015 — Patrulje member endpoints (add/edit/delete)

**Status:** open
**Priority:** high
**Created:** 2026-07-31

## Description

Implements the explicit member lifecycle from PRD 001
(`roadmap/prd/001-seat-based-billing-and-member-identity.md`).

Today the roster is saved through a single `PUT /api/patrulje/{id}` that
republishes an update per member and mints a fresh UUID for any member arriving
without a `memberId` (`patrulje.Commands.Update`,
`go/nathejk/table/patrulje/commands.go:105`). Because the response omits members,
the client never learns the assigned id and re-sends without one — so every save
recreates members.

Add explicit, additive member endpoints for patrulje (these coexist with the
existing PUT until task 020 removes member lifecycle from it):

- `POST /api/patrulje/{teamId}/member` — server issues the `memberId`, publishes
  exactly one member **created** event, enforces the team maximum (task 018 adds
  the cap; wire the hook here), and returns the created member including its
  `memberId`. This is where the id is issued and persisted together — no separate
  id-only endpoint.
- `PUT /api/patrulje/{teamId}/member/{memberId}` — publishes exactly one member
  **update** event; never creates a new identity.
- `DELETE /api/patrulje/{teamId}/member/{memberId}` — publishes exactly one
  member **deleted** event.

Adding/removing a member re-derives the open order (count-based, task 014),
tolerating projection lag the way `setDerivedLinesAfterCreate` already does.

All new endpoints must have OpenAPI annotations (repo rule).

Related files:
- `go/cmd/api/patrulje.go` — new handlers
- `go/cmd/api/main.go` — route registration
- `go/nathejk/table/patrulje/commands.go` — split create vs. update vs. delete
- `go/nathejk/table/spejder/consumer.go` — confirm create→insert, update→update,
  delete→delete

## Acceptance Criteria

- [ ] `POST /api/patrulje/{teamId}/member` issues a `memberId`, emits one create
      event, returns the created member with its id.
- [ ] `PUT /api/patrulje/{teamId}/member/{memberId}` emits one update event and
      never creates a new identity.
- [ ] `DELETE /api/patrulje/{teamId}/member/{memberId}` emits one delete event.
- [ ] Member add/delete re-derives the open order (count-based).
- [ ] OpenAPI annotations added for all three endpoints.
- [ ] `go build` and `go vet` clean.

## Progress Log

- 2026-07-31 10:00 — Task created from PRD 001.
