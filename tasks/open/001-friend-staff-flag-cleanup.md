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
