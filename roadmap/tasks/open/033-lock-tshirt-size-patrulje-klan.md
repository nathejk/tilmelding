# 033 — lock t-shirt size server-side in the patrulje and klan write handlers

**Status:** open
**Priority:** high
**Created:** 2026-08-23

## Description

While a product is closed, its size cannot change. This is the half of PRD 003
that protects **paid** shirts: a size edit on a paid shirt is free today (recorded
as a zero-sum credit/charge pair, PRD 002) and would move a shirt after the print
file is cut.

Depends on task 030 (`app.skuClosed`).

Enforcement is server-side. Task 035 removes the picker from the UI, but that is an
affordance, not a guarantee — a stale client or a crafted request must not be able
to change a size.

**Shared helper** (put it next to `sellable` in `go/cmd/api/orders.go`, or in
`patrulje.go` beside `tshirtSizesFor` — either is fine, one home only):

```go
// lockedSize returns the size a write should persist: the stored one while the
// product is closed for sale, the requested one otherwise.
func (app *application) lockedSize(sku, stored, requested string) string
```

**Call sites**

`go/cmd/api/patrulje.go`:
- `addPatruljeMemberHandler` — a new member gets no shirt while closed; the field
  is simply not honoured (stored value is `""`)
- `updatePatruljeMemberHandler` (`:672` area) — `input.Member.TShirtSize`
  substituted with the value from the `spejder` projection for that member
- `updatePatruljeHandler` — check whether the team-level save carries member sizes;
  if it does, apply the same substitution

`go/cmd/api/klan.go`:
- `addKlanMemberHandler` (`:567` area) — as above
- `updateKlanMemberHandler` (`:613` area) — `input.Member.TShirtSize` substituted
  from the klan senior projection
- `updateKlanHandler` (`:396`) — team-level save; it re-derives from the projection
  rather than from the body, so confirm there is no size path here and note that in
  a comment if so

The stored value must be read from the projection, not trusted from the request.
The response must carry the stored value back, so a stale client re-renders the
truth rather than its own rejected input.

## Acceptance Criteria

- [ ] `lockedSize` implemented once and used by every patrulje/klan write path that
      accepts a size
- [ ] `PUT /api/patrulje/:id/member/:memberId` with a changed `tshirtSize` persists
      the **stored** size and returns it
- [ ] `PUT /api/klan/:id/member/:memberId` likewise
- [ ] `POST` on both member endpoints creates a member with no shirt while closed
- [ ] A locked save produces **no** order line change and no `lines.changed`
- [ ] No zero-sum credit/charge pair can be produced for `tshirt.adult` by any
      patrulje or klan endpoint
- [ ] With the closed set empty, size editing behaves exactly as today
- [ ] OpenAPI `@Description` on the touched endpoints states that `tshirtSize` is
      ignored for a closed product
- [ ] `go build ./...` / `go test ./...` pass in the workspace and with `GOWORK=off`

## Progress Log

- 2026-08-23 — Task created from PRD 003 §8.1. Depends on 030. Disjoint write scope
  from task 034 (crew/personnel) — the two can run in parallel once 030 lands, as
  long as the `lockedSize` helper is landed by whichever goes first.
