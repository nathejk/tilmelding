# 034 — lock t-shirt size server-side in the crew and personnel (gøgler) handlers

**Status:** done
**Priority:** high
**Created:** 2026-08-23
**Picked up by:** agent session (Zed)
**Started:** 2026-08-23
**Completed:** 2026-08-23

## Description

Task 033 for the two single-person signup types. Same rule — while a product is
closed its size cannot change — but the storage is different in both cases, and
crew has a trap.

Depends on task 030 (`app.skuClosed`) and shares the `lockedSize` helper with task
033 (whichever lands first introduces it).

**Crew — `go/cmd/api/crew.go`**

`updateCrewHandler` (L111-160). Crew has **no dedicated t-shirt column**: the size
lives inside the `crewmember` additionals JSON blob under `tshirtSizeKey`
(`crew.go:41`). The handler currently folds the request value in:

```go
if input.Member.TshirtSize != "" {
    additionals[tshirtSizeKey] = input.Member.TshirtSize
} else {
    delete(additionals, tshirtSizeKey)
}
```

**The `delete` branch is the trap.** While closed, it must not run. A save that
happens to carry an empty `tshirtSize` — a stale client, or a form that never
loaded the field because task 035 removed the picker — would otherwise erase the
record of a shirt somebody has already paid for. Substitute the stored value first,
then fold; while closed the stored value is what gets written back, empty or not.

The stored value comes from the existing `crewmember` row, parsed the same way
`crewMemberToView` (L263) does.

**Personnel / gøgler — `go/cmd/api/personnel.go`**

`updatePersonnelHandler` — `input.Person.TshirtSize` feeds
`derivedLinesForPersonnel(person, input.Person)` (L97, function at L169).
Substitute from the stored `personnel.Staff` before building the desired set, so
both the persisted person and the derived lines see the locked value.

Note the background-sync pattern: `CrewView.vue` and `BadutView.vue` fire a
`syncOrder()` PUT on every t-shirt selection (the `watch` on `tshirtSize`). That
path becomes unreachable once task 035 removes the picker, but it must be harmless
if it does fire — which is exactly what this task guarantees.

## Acceptance Criteria

- [x] `PUT /api/crew/:id` with a changed `tshirtSize` persists the **stored** size
      and returns it
- [x] `PUT /api/crew/:id` with an **empty** `tshirtSize` does not erase a stored
      size while the product is closed (the `delete` branch is bypassed)
- [x] `PUT /api/personnel/:id` with a changed `tshirtSize` persists the stored size
      and returns it
- [x] A locked save produces no order line change and no `lines.changed`
- [x] Crew additionals keep every other key untouched by the substitution
- [x] With the closed set empty, both endpoints behave exactly as today, including
      the `delete` branch
- [x] OpenAPI `@Description` on both endpoints states that `tshirtSize` is ignored
      for a closed product
- [x] `go build ./...` / `go test ./...` pass in the workspace and with `GOWORK=off`

## Progress Log

- 2026-08-23 — Task created from PRD 003 §8.1. Depends on 030. Disjoint write scope
  from task 033. Flagged the crew `delete(additionals, tshirtSizeKey)` branch as the
  one way this change could destroy paid-shirt data.
- 2026-08-23 — Picked up. `lockedSize`, `tshirtSKU` and `tshirtLocked()` already
  landed with 033, so this is the two substitutions plus the trap.
- 2026-08-23 — Crew: substitution inserted **before** the additionals fold, reading
  the stored size back through `crewMemberToView` on the current `crewmember` row.
  The `delete` branch is left exactly as it was — it does not need a special case,
  because once the size has been substituted the branch can only fire when the
  stored size is genuinely empty. That is a smaller change than gating the branch
  and has the same effect. Fails closed on an unreadable row.
- 2026-08-23 — Personnel: substitution reads `person.TshirtSize`, the record the
  handler had already fetched, so no extra query. Placed before both
  `Personnel.Update` and `derivedLinesForPersonnel` so the persisted value and the
  derived lines cannot disagree.
- 2026-08-23 — ✅ Criteria 1-3, 5-6: `shop_lock_crew_test.go` and
  `shop_lock_personnel_test.go`. The crew tests exercise the fold directly — the
  empty-save case (the trap), a requested change, starting a shirt from nothing,
  unrelated additionals keys surviving, and that the `delete` branch still works
  when the sale is open. `crewMemberToView` is pinned too, since the lock compares
  against whatever it returns.
- 2026-08-23 — ✅ Criterion 4 follows structurally, as in 033: a locked save writes
  the stored size, so the desired set is unchanged and `SyncNeeded` returns false.
  The personnel tests assert the stronger version — that the derived t-shirt line
  carries the **stored** size after a change was requested, for both
  `participation.gogler` and `participation.crew` owners.
- 2026-08-23 — ✅ Criteria 7-8: annotations were written in task 032 (both handlers
  had none at all, so the blocks added there already state the lock); build, vet,
  gofmt and tests green in the workspace and with `GOWORK=off`.
- 2026-08-23 — All four signup types are now locked server-side and no closed
  product can be sold. Only the frontend (035) remains, so until it lands the
  pickers are still visible — they simply have no effect.
