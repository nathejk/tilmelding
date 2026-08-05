# 028 — shared-go reads three tables it does not project

**Status:** open
**Priority:** medium
**Created:** 2026-08-04

## Description

After the entity move (`ed16c31`), `shared-go/tables/*` queriers **read** three
tables that are still projected only by tilmelding's local `nathejk/table` root
package. shared-go therefore depends on state it does not own.

| Table | Projected by (local, tilmelding) | Read by (shared-go) |
|---|---|---|
| `patruljestatus` | `table.NewPatruljeStatus` | `klan`, `patrulje`, `senior`, `spejder` queriers |
| `spejderstatus` | `table.NewSpejderStatus` | `spejder` querier |
| `confirm` | `table.NewConfirm` | `payment` queries (`GetTeamIDBySecret` etc.) |

Also read locally by `internal/data` (`member.go`, `team.go`, `signup.go`,
`payment.go`) and `nathejk/table/personnel`.

**Why it matters:** a second service consuming these shared entities would
compile and run, but every `JOIN patruljestatus` / `JOIN spejderstatus` /
`FROM confirm` would return nothing, because no projector in that service
populates them. The failure is silent — empty results, not an error. Today it
only works because tilmelding happens to run the projectors in the same
process.

This is the mirror image of task 027: those five projectors are dead and should
go; these three are load-bearing and in the wrong repo.

## Options

1. **Move the three projectors into shared-go** as their own entities
   (`tables/patruljestatus`, `tables/spejderstatus`, `tables/confirm`), wired
   from each service's `main.go`. Keeps ownership with the module that reads
   them. Note `patruljestatus` is read by four different entities, so it should
   be its own package rather than folded into any one of them.
2. **Fold each into the entity that reads it** — only viable for
   `spejderstatus` (single reader) and `confirm` (payment); wrong for
   `patruljestatus`.
3. **Document the requirement** and leave projection to the consuming service —
   cheapest, but leaves the silent-empty-join trap in place. If chosen, at least
   assert the tables exist at startup so it fails loudly.

Option 1 is preferred.

## Related finding: `patruljemerged` has no projector at all

While mapping the above: `internal/data/team.go:62` references
`patruljemerged` in **live** code:

```sql
SELECT DISTINCT m.teamId FROM patruljemerged m
JOIN patruljestatus s ON m.teamId = s.teamId WHERE s.startedUts > 0 ...
```

Nothing in tilmelding or shared-go creates or writes `patruljemerged` — the
`table.NewPatruljeMerged` constructor that presumably did no longer exists (it
survives only as a stale comment in `main.go:258`, see task 027). The same query
appears in `shared-go/tables/{klan,senior}/query.go` but there it is inside
`/* */` blocks, i.e. dead.

So the live call path in `internal/data/team.go` queries a table that may not
exist. Determine whether:
- the table still exists as legacy/monolith-era data (then it is unprojected
  and will drift), or
- it does not exist (then that query errors at runtime and the code path is
  presumably unreachable — confirm and delete it).

Worth resolving as part of this task, or splitting out once diagnosed.

## Acceptance Criteria

- [ ] Decision recorded on ownership of `patruljestatus`, `spejderstatus`,
      `confirm` (move to shared-go vs document + startup assertion)
- [ ] If moving: projectors live in shared-go, tilmelding wires them from
      `main.go`, and the local copies are removed
- [ ] A service using the shared entities cannot silently get empty joins —
      either it projects the tables, or startup fails loudly if they are absent
- [ ] `patruljemerged` diagnosed: table exists (and needs a projector) or the
      live query in `internal/data/team.go` is dead and removed
- [ ] `go build ./...` / `go test ./...` pass in the workspace and with
      `GOWORK=off`

## Progress Log

- 2026-08-04 — Created after the shared-go entity switch (`ed16c31`), from an
  audit of which local projections are orphaned (task 027). Mapped every reader
  of the three tables with commented-out code distinguished from live, which is
  also how the `patruljemerged` gap surfaced. No code change yet — the ownership
  question needs deciding first.
