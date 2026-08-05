# 027 — Delete the orphaned projectors in nathejk/table

**Status:** open
**Priority:** medium
**Created:** 2026-08-04

## Description

Five of the eight projector constructors in the local `nathejk/table` root
package are never wired into `mux.AddConsumer` and are dead. They fall into two
kinds, and the first kind is not merely unused but actively hazardous.

### Superseded duplicates (3) — delete

These project the same tables that the shared-go entities now project, having
been left behind when the entities moved (`ed16c31`):

| Constructor | File(s) | Writes | Superseded by |
|---|---|---|---|
| `NewKlan` | `klan.go`, `klan.sql` | `klan` | `shared-go/tables/klan` |
| `NewPatrulje` | `patrulje.go`, `patrulje.sql` | `patrulje` | `shared-go/tables/patrulje` |
| `NewSignup` | `signup.go` (inline SQL) | `signup` | `shared-go/tables/signup` |

**These must not be rewired.** Two projectors writing the same table from
overlapping subjects would fight; keeping them invites exactly that mistake.

### Fully dead legacy (2) — delete

Orphaned *and* their tables are never read anywhere in tilmelding or shared-go.
Both still subscribe to the pre-cutover monolith subject `"nathejk"`:

| Constructor | File(s) | Writes | Table read by |
|---|---|---|---|
| `NewPincode` | `pincode.go`, `pincode.sql` | `pincode` | nothing |
| `NewRegistrant` | `registrant.go`, `registrant.sql` | `registrant` | nothing |

### Also dead, same package

- `senior.sql`, `spejder.sql` — orphaned SQL, no `go:embed` references them.
  Pre-dates the shared-go move (the entity packages carry their own `table.sql`).
- `cmd/api/main.go:258-259` — commented-out calls to `table.NewPatrulje`,
  `table.NewPatruljeMerged` and `table.NewSpejder`. The latter two **no longer
  exist as functions**, so those comments reference nothing.
- `escapeNull` in `confirm.go` — unexported and uncalled. (A second copy in
  `signup.go` is inside a `/* */` block, which is why the package still
  compiles; it goes away with the file.)
- `Filters` / `Metadata` in `filters.go` — unused. Note they are exported, but
  no other module imports `nathejk.dk`, so "unused" means dead. Confirm before
  removing.

## Deletion is self-contained — verified

Checked for helpers shared with the three *live* projectors (`NewConfirm`,
`NewPatruljeStatus`, `NewSpejderStatus`), which must keep working:

- `substr` is defined **and used only** inside `patrulje.go`; it goes with it.
- `escapeNull`'s only real definition is in `confirm.go` (the `signup.go` copy
  is commented out), so deleting `signup.go` cannot break it.
- No other cross-file references between the orphans and the live files.

## Acceptance Criteria

- [ ] `klan.go`, `klan.sql`, `patrulje.go`, `patrulje.sql`, `signup.go`,
      `pincode.go`, `pincode.sql`, `registrant.go`, `registrant.sql` deleted
- [ ] `senior.sql` and `spejder.sql` deleted
- [ ] The stale commented-out `AddConsumer` entries removed from `main.go`
- [ ] The three live projectors (`confirm`, `patruljestatus`, `spejderstatus`)
      still wired and their tables still created
- [ ] Decision recorded on `filters.go` (`Filters`/`Metadata`) and
      `confirm.go`'s `escapeNull` — delete or keep
- [ ] `go build ./...`, `go vet ./...`, `go tool staticcheck ./...`,
      `go test ./...` pass in the workspace **and** with `GOWORK=off`

## Progress Log

- 2026-08-04 — Created after the shared-go entity switch (`ed16c31`). Counted
  the orphans by enumerating every `func New*` in the root package and grepping
  for references with commented-out code stripped: 3 live, 5 orphaned. Verified
  the live three are load-bearing (their tables are genuinely read) before
  proposing any deletion — see task 028, which covers who should own them.
