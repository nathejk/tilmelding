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

Also read locally by `nathejk/table/personnel`. (`internal/data` used to read
`patruljestatus` too, from `member.go` and `team.go`; both files are gone now
that the entity queriers own those reads, so the local reader count is one
lower — the shared-go side of the problem is unchanged.)

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

While mapping the above: `internal/data/team.go:62` referenced
`patruljemerged` in what looked like **live** code (the file has since been
deleted entirely):

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

## Audit (2026-08-06): can the two status projectors just be deleted?

Asked directly, since the whole point of moving them upstream is that someone
reads them. Answer: **yes for both, eventually, but neither today** — and for
two different reasons.

### `spejderstatus` — the projector is provably a no-op

`nathejk/table/spejderstatus.go` cannot write a row:

- `Consumes()` returns an empty slice — it subscribes to nothing.
- `HandleMessage` is a bare `return nil`; its entire body is inside `/* */`.

So the table is created and stays empty forever, and every reader of it is
reading nothing. Deleting it therefore changes no data — but the `CREATE TABLE`
is the only reason the reader's SQL is valid:

```sql
-- shared-go/tables/spejder/querier.go GetAll (live, and in the pinned version)
IFNULL(ss.status, 'paid') AS status
...
left join spejderstatus ss on s.memberId = ss.id and s.year = ss.year
```

On an existing database the table is already there, so nothing breaks; on a
**fresh** one, dropping the projector makes the patrulje roster query fail with
"table doesn't exist". That is exactly the silent-environment-dependence this
task is about, so it must not be traded for a louder version of itself.

Unblocking is small and behaviour-free upstream: because the table is
guaranteed empty, `IFNULL(ss.status,'paid')` is *always* `'paid'`. Replace it
with the literal and drop the join. Same file also has `GetInactive`, which
inner-joins the empty table and so can only ever return zero rows — it is dead
and should go with it (as should the dead `TeamModel.GetSpejder` noted in step
2 below).

### `patruljestatus` — the projector is live but carries no information

It does fire (`Consumes` `NATHEJK:*.*.*.signedup`, and `Match` lines up once
`subject.FromStr` has turned the first `:` into a `.`), and it writes:

```sql
INSERT INTO patruljestatus SET teamId=%q, year=%q, startedUts=1
  ON DUPLICATE KEY UPDATE startedUts=VALUES(startedUts)
```

`startedUts` is the literal `1` on every row, so `WHERE startedUts > 0` is true
for every row and the column says nothing. The only real content of the table is
"this team published a signedup event" — which is why `JOIN patruljestatus`
behaved as an invisible filter, hiding teams without a row.

Readers, as of today:

| Reader | State |
|---|---|
| shared-go working tree | **none** — removed in shared-go `24cf73c` "stop joining read models on patruljestatus" |
| shared-go **pinned** (`v0.0.0-20260805205843-d0d6fdf64ba1`) | still joins it in `klan.GetByID`, `klan.GetAll`, `patrulje.GetByID`, `spejder.GetAll`, and the senior queries |
| tilmelding | **none**, as of this commit |

That pinned row is the blocker: `24cf73c` and `9028f9f` are local to the
shared-go checkout and **not pushed** (`origin/main` is at `d0d6fdf`), and
`GOWORK=off` — the CI and production resolution path — builds the pinned
version. Delete the projector now and production stops finding any patrulje or
klan at all.

The local reader has been removed: `personnel.querier.GetAll` joined
`patruljestatus`, but it could never have run — it selected `t.staffId` FROM a
table named `staff` while the entity projects `personnel` (primary key
`userId`, no `teamId` column). Nothing called it; the handlers use `GetByID`
only. Removed together with `personnel/filter.go` and
`data.PersonnelInterface.GetAll`.

### Order of operations

1. ~~Push shared-go (`24cf73c`, `9028f9f`) and bump `go.mod` in tilmelding.~~
   **Done 2026-08-06** — pinned at `v0.0.0-20260806122607-9028f9ff641c`, which
   mentions `patruljestatus` only in comments. Nothing anywhere reads it.
2. ~~Delete `nathejk/table/patruljestatus.{go,sql}` and its `main.go` wiring.~~
   **Done 2026-08-06.**
3. Upstream, drop the `spejderstatus` join from `spejder.GetAll` (literal
   `'paid'`) and delete `GetInactive`; push; bump. **← next, and the only thing
   still blocking.**
4. Delete `nathejk/table/spejderstatus.{go,sql}` and its `main.go` wiring.
5. ~~`confirm` is already gone (`f60b5fa`), so after step 4 the root `table`
   package holds only `errors.go` — check whether that still has a consumer.~~
   **Done 2026-08-06** — it had none (nothing referenced
   `table.ErrRecordNotFound` / `ErrEditConflict` / `ErrVerificationFailed`), so
   `errors.go` is deleted already. After step 4 the root `table` package
   disappears entirely and only `nathejk/table/personnel/` remains.

Steps 2 and 4 are then pure deletions with no behaviour change, which is the
point of doing them in this order.

### Exact upstream change still needed (step 3)

In `shared-go/tables/spejder/querier.go`, `GetAll`:

```diff
-  IFNULL(ss.status, 'paid') AS status,
+  'paid' AS status,
 ...
 from spejder s
-left join spejderstatus ss on s.memberId = ss.id and s.year = ss.year
```

That is behaviour-preserving *because* the table is empty by construction, so
`ss.status` is always NULL and the IFNULL always yields `'paid'`. Whether
hard-coding `'paid'` is the right answer is a separate question — it is what the
code already does today, and the member status this once modelled has no
projector anywhere. Delete `GetInactive` in the same pass: it inner-joins the
empty table, so it can only return zero rows.

**Note on data:** neither table needs migrating. `spejderstatus` is empty by
construction. `patruljestatus` holds only `(teamId, year, startedUts=1)`, all of
which is derivable from the `signedup` events, and no query reads a column from
it once the joins are gone. Both can simply be dropped from the schema.

## Acceptance Criteria

- [x] Decision recorded on ownership of `patruljestatus`, `spejderstatus`,
      `confirm` (move to shared-go vs document + startup assertion) — and then
      superseded for two of the three: they are not worth moving, they are worth
      deleting. See the audit above.
- [x] `patruljestatus` deleted: no reader remains in either repo, and the
      projector wrote a constant
- [ ] `spejderstatus` deleted — blocked on shared-go dropping the LEFT JOIN in
      `spejder.GetAll` (step 3 above)
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
- 2026-08-06 — Housekeeping only: `internal/data/{member,team}.go` were deleted
  when the klan and patrulje read paths moved to the entity queriers, so this
  task's list of local readers shrank. The shared-go ownership gap and the
  ordered steps above are untouched and still the work to do.
- 2026-08-06 — Audited whether `patruljestatus` and `spejderstatus` can simply be
  **deleted** rather than moved; see the audit section. Both can, and the
  original "move them to shared-go" decision is superseded for these two:
  `spejderstatus` has a projector that provably writes nothing, and
  `patruljestatus` writes a constant. Neither is worth carrying into a shared
  module.

  Did the one piece that is this repo's to do: removed `personnel.querier.GetAll`
  — the last local reader of `patruljestatus`, and dead code that referenced a
  non-existent `staff` table — plus `personnel/filter.go` and
  `PersonnelInterface.GetAll`. The deletions themselves are blocked on shared-go
  `24cf73c`/`9028f9f` being pushed and pinned, because `GOWORK=off` still builds
  a shared-go that joins `patruljestatus`. Ordered steps recorded above.
- 2026-08-06 — shared-go pushed and pinned at `9028f9ff641c`; verified the
  pinned module mentions `patruljestatus` only in comments. **Deleted
  `patruljestatus.{go,sql}`** and its `main.go` wiring — a pure deletion, no
  behaviour change, both build paths green. Also deleted the now-orphaned
  `nathejk/table/errors.go` (step 5): nothing referenced its three aliases once
  the entity projectors had left. Updated the `go-bff-layout` skill, which still
  listed seven root projectors that no longer exist.

  `spejderstatus` stays for now: the pinned module (and the shared-go working
  tree) still LEFT JOIN it in `spejder.GetAll`, so dropping the `CREATE TABLE`
  would break the patrulje roster on a fresh database. The exact upstream diff
  is written out above.

  Aside, noted while verifying: `payment.Query.ConfirmBySecret` in shared-go
  reads the `confirm` table in **live** code, and that projector was deleted in
  `f60b5fa`. It has no caller in tilmelding, so nothing here breaks, but a
  consumer that calls it gets a missing-table error. Upstream's to fix.
