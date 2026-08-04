# 025 — Switch to the external cqrs module

**Status:** done
**Priority:** high
**Created:** 2026-08-04
**Picked up by:** zed-agent
**Started:** 2026-08-04
**Completed:** 2026-08-04

## Description

`internal/cqrs`, built in task 022, has been published as
`github.com/jrgensen/cqrs`. Switch every import to the external module and
delete the local copy.

This completes the sequence started in 022 and 023. With `cqrs` external, the
packages under `nathejk/table/` have no `nathejk.dk` dependency at all and can
move to `shared-go` whenever we choose — Go's prohibition on importing another
module's `internal` tree no longer stands in the way.

## Acceptance Criteria

- [x] No Go file imports `nathejk.dk/internal/cqrs` or any sub-package.
- [x] `github.com/jrgensen/cqrs` is a direct requirement in `go.mod`.
- [x] `go/internal/cqrs/` is deleted.
- [x] `go list -deps ./nathejk/table/...` reports no `nathejk.dk` package
      outside `nathejk/table/` itself.
- [x] Builds and tests pass both in the workspace and with `GOWORK=off`
      (the CI/production path).
- [x] The `go-bff-layout` skill documents the new layout.

## Progress Log

- 2026-08-04 14:30 — Picked up. Confirmed `github.com/jrgensen/cqrs v0.1.0`
  resolves from the proxy, and that its package layout matches the local one
  (root + `cqrstest`, `deadletter`, `sqlpersister`), so the switch is a prefix
  rewrite rather than a restructuring.
- 2026-08-04 14:35 — Diffed the published source against the local copy before
  trusting it rather than assuming they matched. `cqrs.go`, `migrate.go` and
  `deadletter/dialect.go` are byte-identical; `cqrstest` and
  `deadletter/deadletter.go` differ only in their own import path. **But
  `sqlpersister` has been reworked upstream**: `New` now returns an exported
  `*Writer` instead of an unexported `*client` and takes variadic options, with
  a new `WithLogWriter` for redirecting the echo of failing statements. Our one
  call site — `sqlpersister.New(db.DB())` in `main.go` — is unaffected, since
  the extra parameter is variadic and the result is consumed as a
  `cqrs.Writer`. Worth knowing that the upstream API is now ahead of what this
  repo was written against.
- 2026-08-04 14:40 — Rewrote the import prefix in 53 files, deleted
  `internal/cqrs/`, ran `gofmt -w` to re-sort the import groups (the path moved
  from the `nathejk.dk/…` block to the `github.com/…` block) and `go mod tidy`.
- 2026-08-04 14:45 — `go mod tidy` dropped `github.com/DATA-DOG/go-sqlmock`
  entirely. Verified this is correct rather than a mistake: the only sqlmock
  users in this repo were `internal/cqrs/cqrs_test.go` and
  `deadletter/deadletter_test.go`, both of which left with the local copy and
  now live upstream. Anything here that later needs to mock `database/sql` will
  have to `go get` it back.
- 2026-08-04 14:50 — Verified with `GOWORK=off` as well as in the workspace.
  `cqrs` is deliberately *not* added to `go.work`: unlike `shared-go` there is
  no sibling checkout to develop against, so it resolves from the proxy in
  every environment. Confirmed nothing in `docker/` or the CI workflow
  referenced the old paths.
- 2026-08-04 15:00 — Updated the `go-bff-layout` skill, which had gone stale:
  it still documented a `pkg/` directory containing `sqlpersister` and
  `tablerow` (both gone since 022) and told contributors to publish events via
  `jrgensen/stream` directly. Added a "The cqrs seam" section covering the
  three interfaces and their production implementations, the `interfaces.go`
  convention from 023, and two new "don'ts" — no `internal/` imports under
  `nathejk/table/`, and don't recreate `pkg/`. Checked each claim in the doc
  against the tree rather than writing it from memory.
- 2026-08-04 15:05 — All criteria met. `go list -deps ./nathejk/table/...`
  now reports zero `nathejk.dk` dependencies outside `nathejk/table/` itself;
  `jrgensen/stream` appears only transitively via `cqrs`, which is the intended
  arrangement. `gofmt`, `go build`, `go vet`, `staticcheck` and `go test ./...`
  clean in both workspace modes. Moving to done.

## Outcome

53 files re-pointed, `internal/cqrs/` deleted, one line added to `go.mod`.

The three-task sequence is finished:

| Task | Removed |
|---|---|
| 022 | `pkg/tablerow`, direct `jrgensen/stream` imports, `*sql.DB` in constructors |
| 023 | `internal/validator`, `internal/mailer`, `internal/sms`, `internal/payment/mobilepay` |
| 025 | `nathejk.dk/internal/cqrs` |

`nathejk/table/...` is now self-contained. Its external dependencies are
`jrgensen/cqrs`, `nathejk/shared-go`, `goqu`, `uuid` — nothing repo-local.

Remaining before an actual move to `shared-go`, none of which is a blocker:

- The projections are MySQL-specific (`ON DUPLICATE KEY UPDATE`,
  `INSERT ... SET`, `%q` string quoting). Fine for a shared module used only by
  MySQL consumers; a decision if that stops being true.
- `nathejk/table/interfaces.go` and the parent package's `Filters`/`Metadata`
  are shared by the sub-packages, so the parent has to move with them.
- Only `klan`, `patrulje`, `order` and `payment` have tests. The rest would
  move untested.
