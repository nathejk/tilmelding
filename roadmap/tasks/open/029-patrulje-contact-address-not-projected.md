# 029 — patrulje contact address and postal code are written but never read back

**Status:** open
**Priority:** medium
**Created:** 2026-08-06

## Description

The patrulje page collects a contact **address** and **postal code**, and they
are lost the moment the page reloads.

The write path carries them all the way to the stream:

- `vue/src/views/PatruljeView.vue` binds `contact.address` and `contact.postal`
  and PUTs them under `contact`.
- `shared-go/tables/patrulje.Contact` has `Address` and `PostalCode`.
- `shared-go/messages.NathejkPatruljeUpdated` (`messages/team.go:43-44`) has
  `contactAddress` / `contactPostalCode`.

The read path has nowhere to put them:

- `shared-go/tables/patrulje/table.sql` has only `contactName`, `contactPhone`,
  `contactEmail`, `contactRole` — **no** address or postal-code columns.
- so the consumer cannot project them and `patrulje.GetByID` cannot select them.

Net effect: the user types an address, saves, and the field comes back blank —
`contact.address` and `contact.postal` are `""` on every GET. This is
long-standing, not a regression: the query this repo used before
(`internal/data.TeamModel.GetContact`) did not select them either, because the
columns do not exist.

The two keys are still on the wire (`patruljeContactResponse` in
`go/cmd/api/patrulje.go`) because the form binds to them; they are documented
there as always empty, pointing here.

## Decide first

Is the contact address wanted at all? Two honest options:

1. **Complete the round trip** — add `contactAddress` / `contactPostalCode` to
   the patrulje table and its consumer in shared-go, select them in
   `GetByID`, and map them in `newPatruljeContactResponse`. Note the existing
   events already carry the values, so a replay would populate the new columns
   for historical teams.
2. **Drop the fields** — remove the two inputs from `PatruljeView.vue`, the two
   fields from `patruljeContactResponse`, and (upstream) from `patrulje.Contact`
   and the message. Cheapest, and defensible if nobody uses a contact address.

Option 1 requires a shared-go change, so it is subject to the same sequencing
as task 028 (push shared-go, bump `go.mod`, then wire tilmelding).

## Acceptance Criteria

- [ ] Decision recorded: complete the round trip or drop the fields
- [ ] If completing: the address entered on the patrulje page survives a reload
- [ ] If dropping: no input collects a value the backend cannot return
- [ ] No wire key remains that is documented as permanently empty
- [ ] `go build ./...` / `go test ./...` pass in the workspace and with
      `GOWORK=off`

## Progress Log

- 2026-08-06 — Found while replacing `internal/data.TeamModel.GetContact` with
  `patrulje.GetByID` (which does select the four contact columns). Traced the
  address through the write path to the event and confirmed there is no column
  and no projection for it. Not fixed with the migration: the contract change
  is the same either way and the fix needs a product decision plus an upstream
  schema change.
