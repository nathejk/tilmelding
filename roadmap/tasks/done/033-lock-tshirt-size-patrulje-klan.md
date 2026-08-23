# 033 — lock t-shirt size server-side in the patrulje and klan write handlers

**Status:** done
**Priority:** high
**Created:** 2026-08-23
**Picked up by:** agent session (Zed)
**Started:** 2026-08-23
**Completed:** 2026-08-23

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

- [x] `lockedSize` implemented once and used by every patrulje/klan write path that
      accepts a size
- [x] `PUT /api/patrulje/:id/member/:memberId` with a changed `tshirtSize` persists
      the **stored** size and returns it
- [x] `PUT /api/klan/:id/member/:memberId` likewise
- [x] `POST` on both member endpoints creates a member with no shirt while closed
- [x] A locked save produces **no** order line change and no `lines.changed`
- [x] No zero-sum credit/charge pair can be produced for `tshirt.adult` by any
      patrulje or klan endpoint
- [x] With the closed set empty, size editing behaves exactly as today
- [x] OpenAPI `@Description` on the touched endpoints states that `tshirtSize` is
      ignored for a closed product
- [x] `go build ./...` / `go test ./...` pass in the workspace and with `GOWORK=off`

## Progress Log

- 2026-08-23 — Task created from PRD 003 §8.1. Depends on 030. Disjoint write scope
  from task 034 (crew/personnel) — the two can run in parallel once 030 lands, as
  long as the `lockedSize` helper is landed by whichever goes first.
- 2026-08-23 — Picked up. Landed the shared helpers in `shop.go`: `lockedSize`,
  the `tshirtSKU` constant and `tshirtLocked()`. Task 034 reuses all three.
- 2026-08-23 — Neither read API has a by-id lookup (`SpejderInterface` and
  `SeniorInterface` expose only `GetAll`), so `storedPatruljeSize` /
  `storedKlanSize` scan the roster. Guarded behind `tshirtLocked()` so the extra
  query never happens while the shirt is for sale.
- 2026-08-23 — Decision: a failed roster read returns `""` rather than falling
  back to the requested size. Failing closed means a database blip cannot become
  licence to re-size a shirt that is in the print run.
- 2026-08-23 — The lock is applied *before* the command is published, so the event
  itself carries the locked value — not just the response. A projection replay
  therefore cannot resurrect a size change that was refused.
- 2026-08-23 — ✅ Criteria 1-4 and 7: `shop_lock_test.go` covers a changed size, an
  **emptied** size (which would otherwise cancel a paid shirt), a new member, a
  member with nothing stored, and the failed-read path. The open-shop cases run
  with **no models wired at all**, so the test would nil-panic if the open path
  ever touched the roster — that is the assertion, not just the returned value.
- 2026-08-23 — ✅ Criteria 5-6 follow structurally rather than from a stack test: a
  locked save writes the size that is already stored, so the desired set is
  byte-identical, `SyncNeeded` returns false and nothing is published. And a
  zero-sum credit pair is only ever produced by `ApplyPaidOffset` when a desired
  size differs from a paid one — which the lock makes unreachable. Added
  `TestTshirtSKUMatchesTheDerivedLines` to pin the one assumption that could break
  this: that the SKU the lock names is the SKU the derived lines emit.
- 2026-08-23 — Confirmed both team-level saves (`updatePatruljeHandler`,
  `updateKlanHandler`) carry no member sizes — they take team/contact fields only
  and re-derive sizes from the projection. Recorded that in a comment at each
  `Update` call, as the task asked, so the next reader does not have to re-check.
- 2026-08-23 — ✅ Criteria 8-9: annotations updated on all four member endpoints;
  build, vet and tests green in the workspace and with `GOWORK=off`.
