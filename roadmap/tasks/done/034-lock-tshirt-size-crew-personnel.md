# 034 — lock t-shirt size server-side in the crew and personnel (gøgler) handlers

**Status:** open
**Priority:** high
**Created:** 2026-08-23

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

- [ ] `PUT /api/crew/:id` with a changed `tshirtSize` persists the **stored** size
      and returns it
- [ ] `PUT /api/crew/:id` with an **empty** `tshirtSize` does not erase a stored
      size while the product is closed (the `delete` branch is bypassed)
- [ ] `PUT /api/personnel/:id` with a changed `tshirtSize` persists the stored size
      and returns it
- [ ] A locked save produces no order line change and no `lines.changed`
- [ ] Crew additionals keep every other key untouched by the substitution
- [ ] With the closed set empty, both endpoints behave exactly as today, including
      the `delete` branch
- [ ] OpenAPI `@Description` on both endpoints states that `tshirtSize` is ignored
      for a closed product
- [ ] `go build ./...` / `go test ./...` pass in the workspace and with `GOWORK=off`

## Progress Log

- 2026-08-23 — Task created from PRD 003 §8.1. Depends on 030. Disjoint write scope
  from task 033. Flagged the crew `delete(additionals, tshirtSizeKey)` branch as the
  one way this change could destroy paid-shirt data.
