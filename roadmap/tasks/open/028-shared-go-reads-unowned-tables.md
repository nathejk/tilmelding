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

- [x] Decision recorded on ownership of `patruljestatus`, `spejderstatus`,
      `confirm` (move to shared-go vs document + startup assertion)
- [ ] If moving: projectors live in shared-go, tilmelding wires them from
      `main.go`, and the local copies are removed
- [ ] A service using the shared entities cannot silently get empty joins —
      either it projects the tables, or startup fails loudly if they are absent
- [x] `patruljemerged` diagnosed: table exists (and needs a projector) or the
      live query in `internal/data/team.go` is dead and removed
- [x] `go build ./...` / `go test ./...` pass in the workspace and with
      `GOWORK=off`

## Decision (2026-08-04)

**Option 1 — move the three projectors into shared-go**, each as its own entity
package (`tables/confirm`, `tables/patruljestatus`, `tables/spejderstatus`).
`patruljestatus` must be its own package rather than folded into a reader,
because four separate entities (`klan`, `patrulje`, `senior`, `spejder`) join
it. Option 3 (document only) was rejected: it leaves the silent-empty-join trap
in place, and the whole point of shipping these entities in a shared module is
that a consumer gets working state without reading tilmelding's `main.go`.

**Not implemented here.** The change belongs in the shared-go module, which is
out of scope for this repo. It also cannot land as one atomic tilmelding commit:
switching `main.go` to shared projectors would break `GOWORK=off` (the CI and
production resolution path) until shared-go is committed, pushed, and its
version bumped in `go.mod`. Sequencing therefore matters.

### Steps, in order

1. **In shared-go**, add three entity packages, ported verbatim from
   tilmelding's `nathejk/table/`:
   - `tables/confirm/` — from `confirm.go` (schema is inline in
     `CreateTableSql()`, not an embedded `.sql`; keep or convert to
     `table.sql` to match the other entities).
   - `tables/patruljestatus/` — from `patruljestatus.go` + `patruljestatus.sql`.
   - `tables/spejderstatus/` — from `spejderstatus.go` + `spejderstatus.sql`.
   Follow the existing convention: `table.go` with `New(w cqrs.Writer, ...)`,
   `consumer.go`, `table.sql`. They need only `cqrs`, `shared-go/messages` and
   `shared-go/types` — no new dependencies.
2. **In shared-go**, delete the dead `TeamModel.GetSpejder` in
   `tables/spejder/querier.go`. It joins the unprojected `patruljemerged` table
   and is uncalled — the copies in tilmelding were removed under this task.
3. Commit and push shared-go.
4. **In tilmelding**, bump the shared-go version in `go.mod`, wire the three
   from `main.go` (they are already grouped and commented there, see task 027),
   and delete `nathejk/table/{confirm.go,patruljestatus.go,patruljestatus.sql,
   spejderstatus.go,spejderstatus.sql}`.
5. Check whether `nathejk/table/errors.go` still has any consumer once those
   go. If not, the root `table` package disappears entirely and only
   `nathejk/table/personnel/` remains (blocked on task 001).
6. Verify with `GOWORK=off` as well as the workspace.

## Progress Log

- 2026-08-04 — Created after the shared-go entity switch (`ed16c31`), from an
  audit of which local projections are orphaned (task 027). Mapped every reader
  of the three tables with commented-out code distinguished from live, which is
  also how the `patruljemerged` gap surfaced. No code change yet — the ownership
  question needs deciding first.
- 2026-08-04 — `patruljemerged` half **done**. Diagnosed: nothing projects the
  table, and **both** references were unreachable, which corrects this task's
  original "live code" framing. `GetDiscontinuedTeamIDs` was declared in the
  `Models.Teams` interface but never called; `TeamModel.GetSpejder` (singular)
  was in no interface and never called — the live callers use
  `Members.GetSpejdere` (plural), a different method. So the broken joins could
  not fail in production. Removed both with a comment at each site. Noted that
  `shared-go/tables/spejder/querier.go` holds a copy of the same dead method.
- 2026-08-04 — Ownership half: decision recorded above (Option 1). Began the
  shared-go implementation and stopped — modifying the shared-go module was
  declined, and it is not this repo's to change. Left the task open with the
  ordered steps rather than a partial migration: adding the packages to
  shared-go while tilmelding still projects the same tables would mean two
  projectors for one table across two repos, which is worse than the current
  documented gap. Nothing here is blocking tilmelding; the gap only bites a
  *second* service adopting these entities.
