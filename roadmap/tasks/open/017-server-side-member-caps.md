# 017 — Server-side member-count caps (max on add, block pay below min)

**Status:** open
**Priority:** high
**Created:** 2026-07-31

## Description

Enforce team-size bounds server-side, per PRD 001
(`roadmap/prd/001-seat-based-billing-and-member-identity.md`). Today the 3–7
bound is only a frontend `canSave` gate, which allowed the "20 members" state.

Two enforcement points:

1. **Maximum on add.** The member add endpoints (tasks 015/016) reject an add
   that would take the team above its maximum active member count. Return a clear
   4xx the frontend can surface.
2. **Minimum on pay.** Every payment-initiating path rejects when the active
   member count is below the minimum — no payment started, no payment link
   issued. Paths:
   - the payment-link creation branch in `updatePatruljeHandler` /
     `updateKlanHandler`
   - the dedicated `PUT /api/pay/{teamId}` handler

Bounds come from the same source as `buildTeamConfig` (patrulje 3–7; klan per its
config — confirm the klan bounds before wiring).

Related files:
- `go/cmd/api/patrulje.go`, `go/cmd/api/klan.go` — add endpoints + pay/link paths
- `go/cmd/api/*` — the `PUT /api/pay/{teamId}` handler
- `go/cmd/api/patrulje.go` — `buildTeamConfig` (min/max source)

## Acceptance Criteria

- [ ] Adding a member beyond the team maximum is rejected (patrulje + klan) with
      a clear error.
- [ ] Initiating payment / issuing a payment link is rejected when the team is
      below its minimum member count (patrulje + klan), on all payment paths.
- [ ] Bounds sourced consistently with `buildTeamConfig`.
- [ ] `go build` and `go vet` clean.

## Progress Log

- 2026-07-31 10:00 — Task created from PRD 001.
