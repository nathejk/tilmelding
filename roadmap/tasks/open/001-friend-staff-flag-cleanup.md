# 001 — Friend staff flag cleanup

**Status:** open
**Priority:** medium
**Created:** 2026-06-04

## Description

The legacy code overloads `personnel.Type` (a `types.TeamType`) with the literal value `"friend"` to mean "zero-price staff". The order cutover preserves this via `isFriendStaff()` in `cmd/api/personnel.go`. Long-term the right fix is to promote it to a proper field on `Staff`.

Related files:
- `go/cmd/api/personnel.go` — `isFriendStaff()`, `participationSKUForPerson()`
- `go/nathejk/table/personnel/` — `Staff` struct
- `shared-go/messages` — `NathejkPersonnelUpdated`

## Acceptance Criteria

- [ ] `IsFriend bool` (or similar) added to `personnel.Staff` type
- [ ] `NathejkPersonnelUpdated` message carries the new field
- [ ] Projector updated to write/read the field
- [ ] `participationSKUForPerson` switches on the new field, not string-sniffing `Type`/`Status`
- [ ] Legacy data path (`Type == "friend"`) removed or deprecated with a migration note

## Progress Log

- 2026-06-04 21:54 — Task created.
- 2026-08-04 — Assessed while clearing the open board; **left open, blocked on
  an external module.** A hard acceptance criterion is that
  `NathejkPersonnelUpdated` carries the new `IsFriend` field, and that message
  type lives in `github.com/nathejk/shared-go`, a separate module. It is not
  checked out in this environment (`../../shared-go` is absent; it resolves
  read-only from the module cache), so the field cannot be added here.

  Doing only the tilmelding-side half (add `Staff.IsFriend`, update the
  projector, switch `participationSKUForPerson`) would be incoherent: the field
  would never be populated, because nothing would set it without the message
  carrying it. So this needs a coordinated change:
  1. In shared-go: add `IsFriend bool` to `NathejkPersonnelUpdated`; tag,
     release, bump the version here.
  2. In tilmelding: add `IsFriend` to `personnel.Staff` + `personnel.sql`,
     write/read it in the projector, and switch `isFriendStaff()` /
     `participationSKUForPerson()` off string-sniffing `Type == "friend"` onto
     the field, keeping a read-side fallback for legacy rows during migration.

  Pick up once shared-go is available as a sibling checkout (dev workspace) or
  the field has shipped in a shared-go release.
