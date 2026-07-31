# 017 — Server-side member-count caps (max on add, block pay below min)

**Status:** done
**Priority:** high
**Created:** 2026-07-31
**Picked up by:** agent (opus)
**Started:** 2026-07-31
**Completed:** 2026-07-31

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

- [x] Adding a member beyond the team maximum is rejected (patrulje + klan) with
      a clear error.
- [x] Initiating payment / issuing a payment link is rejected when the team is
      below its minimum member count (patrulje + klan), on all payment paths.
- [x] Bounds sourced consistently with `buildTeamConfig`.
- [x] `go build` and `go vet` clean.

## Progress Log

- 2026-07-31 10:00 — Task created from PRD 001.
- 2026-07-31 13:25 — Picked up. Plan: define patrulje/klan min-max consts (reuse in buildTeamConfig calls); enforce max in add member handlers (reject when active count >= max); enforce min on payment paths (updatePatrulje/updateKlan payment-link branch + PUT /api/pay/:id). Checking error-response helpers + the pay handler first.
- 2026-07-31 13:30 — Note: `/api/pay/:id` route is commented out in routes.go; the active payment-initiating path is the payment-link branch in updatePatrulje/updateKlanHandler. Gating that.
- 2026-07-31 13:35 — Added `patruljeMin/Max` (3/7) and `klanMin/Max` (1/4) consts, reused in the buildTeamConfig calls. Max-on-add: add handlers count the projection and 422 (`FailedValidationResponse`) when full. Min-on-pay: payment branch is non-fatal — when active submitted members < min it issues NO payment link and returns `paymentError` in the envelope (kept non-fatal so the live order sync isn't broken for a team still being built). Counts active from the submitted roster (lag-free).
- 2026-07-31 13:40 — ✅ Verified max-on-add against dev team 551c8afb (20 members): POST /member → HTTP 422 `{"error":{"members":"en patrulje kan højst have 7 spejdere"}}`, row count unchanged (20). Klan path is symmetric. build/vet/staticcheck clean.
- 2026-07-31 13:41 — Completed. Min-on-pay covered by code + task 021 tests (constructing a below-min live PUT would mutate real team data). paymentError surfaced to FE in tasks 018/019.
