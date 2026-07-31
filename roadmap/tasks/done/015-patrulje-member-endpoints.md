# 015 — Patrulje member endpoints (add/edit/delete)

**Status:** done
**Priority:** high
**Created:** 2026-07-31
**Picked up by:** agent (opus)
**Started:** 2026-07-31
**Completed:** 2026-07-31

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

- [x] `POST /api/patrulje/{teamId}/member` issues a `memberId`, emits one create
      event, returns the created member with its id.
- [x] `PUT /api/patrulje/{teamId}/member/{memberId}` emits one update event and
      never creates a new identity.
- [x] `DELETE /api/patrulje/{teamId}/member/{memberId}` emits one delete event.
- [x] Member add/delete re-derives the open order (count-based).
- [x] OpenAPI annotations added for all three endpoints.
- [x] `go build` and `go vet` clean.

## Progress Log

- 2026-07-31 10:00 — Task created from PRD 001.
- 2026-07-31 12:40 — Picked up. Plan: add `AddMember`/`UpdateMember`/`DeleteMember` commands to patrulje (each emits exactly one event; Add issues the memberId), add `POST/PUT/DELETE /api/patrulje/{id}/member[/{memberId}]` handlers that re-derive the open order, register routes, and annotate for OpenAPI. Investigating routing + existing annotation style first.
- 2026-07-31 12:45 — Found no OpenAPI tooling or annotations anywhere in the repo despite the repo rule. Decision: added swaggo-style (`@Summary`/`@Router`/…) doc comments to the new handlers as the de-facto Go OpenAPI convention; if a generator is wired up later they'll be picked up. Flagged for the team.
- 2026-07-31 12:50 — Added `AddMember` (issues id, publishes spejder.updated WITH teamId = create path), `UpdateMember` (publishes spejder.updated WITHOUT teamId so the projector does a pure UPDATE and never resurrects/creates), `DeleteMember` (publishes spejder.deleted) to `patrulje/commands.go`, sharing a `newScoutUpdated` body builder.
- 2026-07-31 12:55 — Added handlers + `rederivePatruljeOrder`/`replaceMemberLines` in `patrulje.go`: re-derivation applies the single-member change on top of the (possibly lagging) projection so the returned order is correct immediately. Registered 3 routes. build/vet/staticcheck clean.
- 2026-07-31 13:00 — ✅ Verified end-to-end against dev team 551c8afb: POST added exactly one row (20→21) returning the member with a server-issued id + recomputed order; PUT changed tshirt m→l with NO new row (still 21) — the churn is gone; DELETE removed it (21→20). All criteria met.
- 2026-07-31 13:01 — Completed. Existing roster PUT still handles members for now; it is removed in task 020 after the frontend migrates.
