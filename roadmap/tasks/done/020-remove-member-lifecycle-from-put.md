# 020 — Remove member lifecycle from roster PUT handlers

**Status:** done
**Priority:** high
**Created:** 2026-07-31
**Picked up by:** agent (opus)
**Started:** 2026-07-31
**Completed:** 2026-07-31

## Description

Final backend step of PRD 001
(`roadmap/prd/001-seat-based-billing-and-member-identity.md`). Once the frontend
is migrated to the member endpoints (tasks 018/019), remove the member
create/delete path — and the ad-hoc UUID minting — from the roster `PUT`
handlers so member identities can no longer be created as a side effect of a
save. This structurally guarantees no churn.

- `updatePatruljeHandler` / `updateKlanHandler` handle team + contact fields and
  order recomputation only; they no longer accept or act on member lifecycle.
- Remove the `if m.MemberID == "" { uuid.New() }` minting from
  `patrulje.Commands.Update` (and the klan equivalent), or drop the member loop
  from `Update` entirely in favour of the dedicated member commands.

Must land after 018/019 so no caller relies on the old member-in-PUT behavior.

Related files:
- `go/cmd/api/patrulje.go` — `updatePatruljeHandler`
- `go/cmd/api/klan.go` — `updateKlanHandler`
- `go/nathejk/table/patrulje/commands.go` — `Update`
- `go/nathejk/table/klan/commands.go` — update command

## Acceptance Criteria

- [x] Roster `PUT` handlers no longer create or delete members.
- [x] Ad-hoc UUID minting removed from the patrulje and klan update commands.
- [x] Team/contact update and count-based order re-derivation still work via PUT.
- [x] A save of an unchanged roster produces no new member rows / no create
      events (verified against dev).
- [x] `go build` and `go vet` clean.

## Progress Log

- 2026-07-31 10:00 — Task created from PRD 001.
- 2026-07-31 14:30 — Picked up. Plan: patrulje — make Update team/contact-only (drop members param, loop, uuid minting); handler re-derives order from projection; FE putState drops members. Klan — make the roster PUT team/status-only (drop member loop + placeholder rows + minting from UpdateMembers); FE putState drops members. Checking callers first.
- 2026-07-31 14:45 — patrulje: `Update` is now team/contact only. `updatePatruljeHandler` drops the Members field, re-derives the order from GetSpejdere (self-heal, same as show handler), and gates payment on the projection member count. FE PatruljeView putState no longer sends members.
- 2026-07-31 14:55 — klan: `UpdateMembers` → `UpdateTeam` (team event + status transitions only; removed the per-senior event loop, the ad-hoc UUID minting, and the memberCount placeholder-row logic — seniors are now added explicitly via the member endpoints). `updateKlanHandler` re-derives from GetSeniore and gates payment on the projection count. FE KlanView putState no longer sends members.
- 2026-07-31 15:00 — build/vet/staticcheck clean; vue build clean. ✅ Verified: a team-only PUT to dev team fcc870d1 kept the roster at 4 rows (no churn) and re-derived the order to 70000 (700 kr, t-shirts only); paymentError empty (4 ≥ min 3).
- 2026-07-31 15:01 — Completed. The roster PUT can no longer create or delete member identities — the churn is structurally impossible.
