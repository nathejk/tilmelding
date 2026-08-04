# 022 — Introduce cqrs seam, remove pkg/tablerow

**Status:** done
**Priority:** high
**Created:** 2026-08-04
**Picked up by:** zed-agent
**Started:** 2026-08-04
**Completed:** 2026-08-04

## Description

The entities under `go/nathejk/table/` are candidates for extraction into the
shared `shared-go` repository. Today they cannot move, because they import
`nathejk.dk/pkg/tablerow` — a module-internal package. They also import
`github.com/jrgensen/stream` and `github.com/jrgensen/stream/subject`
directly, and take a concrete `*sql.DB` for the query side, which welds them
to one specific infrastructure choice and makes them awkward to mock.

Introduce a single infrastructure seam, `cqrs`, holding the three roles that
`cmd/api/main.go` already wires by hand:

```go
tableSignup := signup.New(publisher, writer, reader, ...)
```

- **Publisher** — command side; appends domain events to the event stream.
- **Writer** — projection side; applies statements that build the read model.
- **Reader** — query side; reads the projected read model.

The package lives at `go/internal/cqrs` for now. Once the design settles it
moves out to `github.com/jrgensen/cqrs` and only the import path changes, so
the seam must not leak anything repo-specific.

Constraints:

- `Reader` must remain assignable from `*sql.DB` and must satisfy
  `goqu.SQLDatabase` — `section` and `crewmember` pass the reader to
  `goqu.New("mysql", r)`.
- `Publisher` / `Consumer` / `Message` / `Subject` must stay interchangeable
  with their `jrgensen/stream` counterparts, so the existing jetstream
  transport and `streamtest` fakes keep working without adapters.
- No behaviour change. This is a pure dependency-direction refactor.

## Acceptance Criteria

- [x] `internal/cqrs` defines `Publisher`, `Writer`, `Reader` and `Consumer`,
      plus the `Message` / `Subject` seam and a subject constructor.
- [x] `internal/cqrs` carries the `EnsureColumn` / `EnsureIndex` migration
      helpers, expressed in terms of `Reader` and `Writer`.
- [x] `internal/cqrs/cqrstest` provides in-memory `Writer` and `Publisher`
      fakes so entities can be tested without a database or a broker.
- [x] No package under `nathejk/table/` imports `pkg/tablerow`,
      `jrgensen/stream` or `jrgensen/stream/subject`.
- [x] Entity constructors take `cqrs.Reader` instead of `*sql.DB`.
- [x] `pkg/tablerow` is deleted.
- [x] `go build ./...` and `go test ./...` pass.

## Progress Log

- 2026-08-04 09:10 — Task created.
- 2026-08-04 09:12 — Picked up. Surveyed the blast radius: 30 files import
  `pkg/tablerow`, but only three of its symbols are actually used (`Consumer`,
  `EnsureColumn`, `EnsureIndex`) — `SQLTableRow`, `SQLPrimaryKeys`,
  `SQLTableCreator` and `EntityChangedPublisher` are dead and will not be
  carried over. The `stream` surface used under `nathejk/table/` is equally
  narrow: `subject.FromStr` (108), `stream.Subject` (38), `stream.Message`
  (27), `stream.Publisher` (20), `stream.Consumer` (2). Plan: build the seam
  with type aliases to `stream` so satisfaction stays implicit in both
  directions, then mechanically rewrite the entity packages.
- 2026-08-04 09:40 — `internal/cqrs` written. Chose type *aliases* for
  `Message` / `MutableMessage` / `MessageFunc` / `Subject` and *fresh
  declarations* with identical method sets for `Publisher` / `Consumer`.
  Wrapper types were the alternative and were rejected: an adapter layer
  between cqrs and the jetstream transport would have to be maintained and
  would silently rot. With aliases, `metatagger`, `xstream.Mux` and
  `streamtest` satisfy the seam as-is, and four assertions in `cqrs.go` fail to
  compile if either side's method set drifts.
- 2026-08-04 09:55 — `Reader` is the full `*sql.DB` method set minus
  connection-lifecycle calls. Not a narrower interface: `section` and
  `crewmember` hand the reader to `goqu.New`, which demands
  `goqu.SQLDatabase`. Asserted in `cqrs_test.go` rather than in `cqrs.go`, so
  the seam itself stays free of a goqu dependency when it moves out.
- 2026-08-04 10:05 — ✅ Criteria 1–3 complete: seam, migration helpers and
  `cqrstest` fakes in place; `go vet` clean.
- 2026-08-04 10:30 — Entity packages rewritten. Only three genuine fixups were
  needed beyond the scripted symbol/import rewrite: `database/sql` became
  unused in ten `table.go` files once the constructor stopped naming
  `*sql.DB`, and three files needed the `cqrs` import added by hand. The
  remaining `database/sql` imports in the `query.go`/`querier.go` files are
  legitimate — `sql.ErrNoRows`, `sql.NullInt64` — and stay.
- 2026-08-04 10:40 — `EntityChangedPublisher`, `SQLTableRow`,
  `SQLPrimaryKeys` and `SQLTableCreator` were dead on arrival and are not
  carried into cqrs; only `Consumer` (now `Writer`), `EnsureColumn` and
  `EnsureIndex` had callers. `pkg/deadletter` and `pkg/sqlpersister` now
  declare `var _ cqrs.Writer`, which makes the projection-sink contract
  explicit at both ends of the decorator.
- 2026-08-04 10:50 — ✅ Criteria 4–6 complete: `pkg/tablerow` deleted, nothing
  references it, `go build ./...` and `go vet ./...` clean.
- 2026-08-04 11:00 — Moved the `klan` and `patrulje` commander tests from
  `streamtest` to `cqrstest`, which removed the last `jrgensen/stream` import
  from the entity packages. Side effect: `go mod tidy` dropped
  `json-iterator/go`, `modern-go/concurrent` and `modern-go/reflect2` —
  `cqrstest` encodes bodies with stdlib `encoding/json`. This effectively
  reverts the dependency added in 339f458.
- 2026-08-04 11:05 — ✅ Criterion 7 complete: `go build ./...`,
  `go vet ./...`, `staticcheck` and `go test ./...` all pass; `gofmt -l` clean.
  Added tests for the seam itself (subject parsing, both migration helpers
  including their idempotent and error paths) and for the fakes.
- 2026-08-04 11:10 — All criteria met. `go list -deps ./nathejk/table/...`
  shows the entities' only remaining infrastructure dependency is
  `internal/cqrs`. Three unrelated module-internal dependencies still block the
  `shared-go` move — `internal/validator`, `internal/mailer` + `internal/sms`,
  and `internal/payment/mobilepay` — split out as task 023 rather than grown
  into this one. Moving to done.
- 2026-08-04 11:40 — Follow-up on request: the cqrs interface implementations
  are now sub-packages of the module rather than siblings elsewhere in the
  tree. `pkg/sqlpersister` → `internal/cqrs/sqlpersister`, `pkg/deadletter` →
  `internal/cqrs/deadletter`; `go/pkg/` is now empty and removed. Only
  `cmd/api/main.go` imported either, so the move was two import lines. The
  module now carries the seam plus every adapter for it, which is what makes it
  self-contained enough to extract: nothing outside `internal/cqrs` names a
  broker or a driver except `main`, which must.
- 2026-08-04 12:20 — Asked whether multi-dialect support in the two adapters
  would be a small change. Answer differed per package, and the interesting
  finding was neither of them:

  - `sqlpersister`: nothing to do. It forwards an opaque string to `Exec` and
    has no dialect knowledge to begin with.
  - `deadletter`: small, and now done. Exactly two statements varied — the DDL
    and the capture INSERT's placeholders. Added a `Dialect` struct holding
    those two strings, with `MySQL` (default), `Postgres` and `SQLite` values
    and a `WithDialect` option. `Reset`/`Count` are ANSI and shared. Chose
    plain data over an interface: with two strings there is nothing to
    dispatch on, and it lets a caller support an unanticipated database
    without changing the package (covered by `TestCustomDialect`).
  - The actual obstacle is the projections: 50 uses of
    `ON DUPLICATE KEY UPDATE` / `INSERT ... SET` / `INSERT IGNORE`, plus 17
    files building statements with `%q`, which emits `"value"` — a string in
    MySQL but an *identifier* in PostgreSQL and SQLite, so every such
    statement would fail. Not attempted; not small.

  Recorded the asymmetry in the package docs so nobody has to rediscover it:
  `sqlpersister` states that its portability says nothing about the statements
  passing through it, and `migrate.go` now states plainly that `EnsureColumn`
  and `EnsureIndex` are MySQL/MariaDB-only (SQLite has no INFORMATION_SCHEMA)
  with a note to move them to a `mysql` sub-package rather than generalise
  them in place, should that ever be needed.

## Outcome

`internal/cqrs` is the single seam the entities see: `Publisher` (command
side), `Writer` (projection side), `Reader` (query side), plus `Consumer` and
the `Message`/`Subject` types. The adapters live beneath it:

```
internal/cqrs/
  cqrs.go         Publisher, Writer, Reader, Consumer, Message/Subject seam
  migrate.go      EnsureColumn, EnsureIndex
  sqlpersister/   Writer — executes statements against a database
  deadletter/     Writer decorator — diverts failing statements to a table
  cqrstest/       in-memory Writer and Publisher fakes
```

The production `Publisher` and `Reader` need no adapter: a metatagger over
JetStream and an `*sql.DB` satisfy them as they stand. 61 files changed,
`pkg/tablerow` gone, `go/pkg/` gone, no behaviour change.

When the package is promoted to `github.com/jrgensen/cqrs`, the mechanical part
is a find/replace of `nathejk.dk/internal/cqrs`. Two things to decide at that
point:

- The `stream` aliases in `cqrs.go` are the one place naming a concrete
  messaging library. As an external module, cqrs would depend on
  `jrgensen/stream` — acceptable, and the reason the aliases were kept to a
  single file.
- The `goqu.SQLDatabase` assertion lives in `cqrs_test.go`. Keep it there so
  the module doesn't take a goqu dependency, but do carry the test across —
  it is the only thing preventing someone from narrowing `Reader` and breaking
  the goqu call sites.
