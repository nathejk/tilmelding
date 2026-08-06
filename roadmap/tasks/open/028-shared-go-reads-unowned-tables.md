# 028 — shared-go reads tables it does not project

**Status:** open
**Priority:** low
**Created:** 2026-08-04

> **Mostly resolved.** Two of the three tables are gone, and the ownership
> question turned out to be the wrong question — the answer for both status
> tables was "delete, don't move". What is left is one two-line upstream diff
> and one upstream decision, neither of which affects tilmelding today. Priority
> dropped from medium to low accordingly. The original framing, options and
> decision are preserved at the bottom, marked superseded, because the reasoning
> is worth keeping.

## What is still relevant

### 1. `spejderstatus` — one upstream diff away from gone

The local projector and the root `table` package are deleted. All that survives
is a `CREATE TABLE IF NOT EXISTS spejderstatus (...)` in `cmd/api/main.go`,
labelled as a compatibility shim, because `shared-go/tables/spejder`'s `GetAll`
still does:

```sql
IFNULL(ss.status, 'paid') AS status
...
left join spejderstatus ss on s.memberId = ss.id and s.year = ss.year
```

Nothing has ever written to that table (the projector subscribed to no subject
and had an empty handler), so `ss.status` is always NULL and the `IFNULL` always
yields `'paid'`. The upstream change is therefore behaviour-preserving:

```diff
-  IFNULL(ss.status, 'paid') AS status,
+  'paid' AS status,
 ...
 from spejder s
-left join spejderstatus ss on s.memberId = ss.id and s.year = ss.year
```

Push that, bump `go.mod`, delete the six shim lines. Done.

Separate question it exposes, **not** part of this task: hard-coding `'paid'`
means the patrulje roster reports every scout as paid regardless of reality.
That is what the code already does, but if member status should mean something,
it needs a `status` column on `spejder` and a projector that maintains it — a
feature, not a cleanup.

### 2. `confirm` — read upstream, projected nowhere at all

This one got *worse* than originally described, and is now purely shared-go's
problem. `payment.Query.ConfirmBySecret` reads the table in live code:

```sql
UPDATE signup s JOIN confirm c ON s.teamId = c.teamId SET s.email = c.emailPending WHERE secret = ?
SELECT teamId FROM confirm WHERE secret = ?
```

The projector that built `confirm` was deleted here as orphaned (`f60b5fa`), so
**no repo projects it now** — the table is not merely unowned, it is absent.

Why tilmelding is unaffected, verified rather than assumed:

- `ConfirmBySecret` is not on `data.PaymentInterface` (`GetAll`,
  `GetByReference`, `AmountPaidByTeamID`) and has no caller here.
- The live email-verification path does not touch it: `signup.VerifyEmail` reads
  `signup.EmailPending` through its own querier and publishes
  `NathejkSignupEmailVerified`.

So the fix is upstream and is a choice between two: delete `ConfirmBySecret` as
dead, or reinstate a `confirm` projection in shared-go if some consumer still
needs secret-based email confirmation. Nothing to do in this repo either way.

## Resolved

| Item | Outcome |
|---|---|
| `patruljestatus` | **Deleted.** No reader left in either repo; the projector wrote `startedUts=1` unconditionally, so the column carried no information and the JOINs it fed acted as an invisible filter. |
| `spejderstatus` projector | **Deleted.** Never a projector: an empty `Consumes()` meant `xstream`'s `Subscribe` built no consumer at all, and `HandleMessage`'s body was commented out. Only the `CREATE TABLE` remains, see above. |
| `confirm` projector | Deleted under task 027 as orphaned. The upstream reader outliving it is item 2 above. |
| `patruljemerged` | Diagnosed: nothing projects it, and **both** references were unreachable, so the broken joins could never fail in production. Both removed. |
| Local readers | All gone: `internal/data/{member,team}.go` moved to entity queriers; `personnel.querier.GetAll` deleted (dead code against a non-existent `staff` table). |
| Root `nathejk/table` package | Gone, including the orphaned `errors.go`. Only `nathejk/table/personnel/` remains (blocked on task 001). |

## Acceptance Criteria

- [x] Ownership decision recorded — and then superseded: two of the three tables
      were not worth moving, they were worth deleting
- [x] `patruljestatus` deleted
- [x] Root `nathejk/table` package gone; only `personnel/` remains
- [x] `patruljemerged` diagnosed and its dead queries removed
- [x] No tilmelding code reads a table it does not project
- [ ] `spejderstatus` fully gone — blocked on the upstream `spejder.GetAll` diff
      (item 1)
- [ ] `confirm` resolved upstream: `ConfirmBySecret` deleted, or a projection
      added (item 2)
- [x] `go build ./...` / `go test ./...` pass in the workspace and with
      `GOWORK=off`

---

# Superseded (kept for the reasoning)

The sections below are the task as originally written. They are **out of date**:
the premise was that three load-bearing projectors sat in the wrong repo and
should move to shared-go. Two turned out to be writing nothing of value and were
deleted instead; the third's projector was already dead. Nothing here is work to
do.

## Original description

After the entity move (`ed16c31`), `shared-go/tables/*` queriers **read** three
tables that are still projected only by tilmelding's local `nathejk/table` root
package. shared-go therefore depends on state it does not own.

| Table | Projected by (local, tilmelding) | Read by (shared-go) |
|---|---|---|
| `patruljestatus` | `table.NewPatruljeStatus` | `klan`, `patrulje`, `senior`, `spejder` queriers |
| `spejderstatus` | `table.NewSpejderStatus` | `spejder` querier |
| `confirm` | `table.NewConfirm` | `payment` queries (`GetTeamIDBySecret` etc.) |

**Why it matters:** a second service consuming these shared entities would
compile and run, but every `JOIN patruljestatus` / `JOIN spejderstatus` /
`FROM confirm` would return nothing, because no projector in that service
populates them. The failure is silent — empty results, not an error. Today it
only works because tilmelding happens to run the projectors in the same
process.

This is the mirror image of task 027: those five projectors are dead and should
go; these three are load-bearing and in the wrong repo. *(That last clause is
what proved wrong.)*

## Original options

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

Option 1 was preferred and became the decision below. It was abandoned once the
projectors were actually read: there was nothing worth moving. Option 3's "fail
loudly" instinct was right, though — the labelled shim in `main.go` is the
honest version of it.

## Related finding: `patruljemerged` has no projector at all

While mapping the above: `internal/data/team.go:62` referenced `patruljemerged`
in what looked like **live** code (the file has since been deleted entirely):

```sql
SELECT DISTINCT m.teamId FROM patruljemerged m
JOIN patruljestatus s ON m.teamId = s.teamId WHERE s.startedUts > 0 ...
```

Nothing in tilmelding or shared-go creates or writes `patruljemerged`. Resolved
on 2026-08-04: both references were unreachable. See the Resolved table.

## Decision (2026-08-04) — superseded 2026-08-06

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

### Steps, in order — not the plan any more

Recorded for the record; steps 1–4 were never executed and should not be. Only
step 5's outcome survives (the root `table` package did disappear, just by
deletion rather than migration).

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
- 2026-08-06 — Bumped to shared-go `e7b46bb` (`v0.0.0-20260806204955-`). It
  disables `GetInactive`, but **`spejder.GetAll` still LEFT JOINs
  `spejderstatus`**, so the table must still exist and a clean delete is still
  blocked. Went as far as is safe: deleted `spejderstatus.{go,sql}` — the
  "projector" only ever created a table (empty `Consumes()` meant
  `mux.Subscribe` created no consumer at all, so registering it was a genuine
  no-op) — and replaced it with the bare `CREATE TABLE IF NOT EXISTS` in
  `main.go`, commented as a shim for the upstream join. The root `nathejk/table`
  package is now gone entirely; only `nathejk/table/personnel/` is left.

  Aside, noted while verifying: `payment.Query.ConfirmBySecret` in shared-go
  reads the `confirm` table in **live** code, and that projector was deleted in
  `f60b5fa`. It has no caller in tilmelding, so nothing here breaks, but a
  consumer that calls it gets a missing-table error. Upstream's to fix.
- 2026-08-06 — Reviewed how much of this task is still relevant, and rewrote the
  top of the file to say so. Answer: two items, both upstream, neither affecting
  tilmelding. Everything about local projectors, ownership and migration is done
  or moot, so the original description, options, decision and ordered steps moved
  below a "Superseded" banner rather than being deleted — the reasoning explains
  why the answer changed. Priority medium → low, since nothing here blocks work
  in this repo.

  Also promoted `confirm` from an aside to a numbered item, having verified
  properly this time that tilmelding cannot reach it: `ConfirmBySecret` is absent
  from `data.PaymentInterface` and uncalled, and the live email-verification path
  (`signup.VerifyEmail`) reads `signup.EmailPending`, not `confirm`. Dropped two
  stale acceptance criteria ("if moving: projectors live in shared-go", "a
  service using the shared entities cannot silently get empty joins") — the first
  is not happening, and the second is satisfied for tilmelding and is a shared-go
  criterion, not one this repo can tick.
