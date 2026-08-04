# 023 — Remaining internal deps blocking the table → shared-go move

**Status:** done
**Priority:** medium
**Created:** 2026-08-04
**Picked up by:** zed-agent
**Started:** 2026-08-04
**Completed:** 2026-08-04

## Description

Task 022 introduced `internal/cqrs` and removed `pkg/tablerow`, so the
entities under `go/nathejk/table/` no longer depend on the repo for their
infrastructure. Three module-internal dependencies remain, and each must be
resolved before the packages can move to `shared-go`:

**1. `internal/validator` — 6 files, smallest effort**

`nathejk/table/filters.go` plus the `filter.go` in `senior`, `spejder`,
`klan`, `patrulje` and `personnel` implement:

```go
func (f *Filter) Validate(v validator.Validator)
```

and call `validator.PermittedValue(...)`. Two options: move `validator` to
`shared-go` (it is a generic, dependency-free helper), or invert the direction
so `Validate` returns errors and the HTTP layer maps them. The former is less
disruptive; the latter removes a concept from the domain packages entirely.

**2. `internal/mailer` and `internal/sms` — 1 file, trivial**

`nathejk/table/signup/repository.go` holds `sms.Sender` and `mailer.Mailer`.
Both are single-method-ish interfaces already, so the fix is plain dependency
inversion: declare the ports in the `signup` package and let `cmd/api` satisfy
them with the concrete clients. No new shared package needed.

**3. `internal/payment/mobilepay` — 1 file, largest effort**

`nathejk/table/payment/commands.go` uses `mobilepay.Client` along with a
sizeable chunk of its type vocabulary — `Amount`, `Payment`, `PaymentMethod`,
`Customer`, `PaymentReference`, `UserFlowWeb`, `PaymentStateAuthorized`. This
is not a thin seam: the MobilePay data model is baked into the command
signatures. Either move `mobilepay` to `shared-go` as-is, or define a
payment-provider port in the `payment` package with Nathejk's own vocabulary
and adapt MobilePay to it in `cmd/api`. The second is the better design and
the more invasive change; decide deliberately rather than by default.

Once these are gone, `go list -deps ./nathejk/table/...` should show no
`nathejk.dk/...` package other than `nathejk/table/...` itself and the (by
then external) cqrs module.

## Acceptance Criteria

- [x] No package under `nathejk/table/` imports `internal/validator`.
- [x] No package under `nathejk/table/` imports `internal/mailer` or
      `internal/sms`.
- [x] No package under `nathejk/table/` imports `internal/payment/mobilepay`.
- [x] `go list -deps ./nathejk/table/...` lists only `nathejk/table/...` and
      the cqrs module under `nathejk.dk`/the cqrs import path.
- [x] `go build ./...` and `go test ./...` pass.

## Approach

Dependency inversion in all three cases, via an `interfaces.go` file sitting
alongside the entity files of each package that needs one. The entity declares
the interface it requires; `cmd/api` supplies something that satisfies it.

## Progress Log

- 2026-08-04 11:05 — Task created as the follow-up to 022, which removed the
  `pkg/tablerow` blocker. Blockers above were found with
  `go list -deps ./nathejk/table/...` after 022 landed; effort estimates come
  from reading the call sites, not from attempting the change.
- 2026-08-04 12:45 — Picked up. Approach fixed by request: local
  `interfaces.go` per package rather than moving packages to `shared-go`.
  Checked the import graph first — parent `table` imports `table/payment`, so
  `table/payment` must not import the parent; its port has to be
  self-contained. `senior`/`spejder`/`klan`/`patrulje`/`personnel` already
  import the parent as `tables`, so the validator port can live there once
  rather than being copied five times.
- 2026-08-04 12:50 — Noted while reading call sites: nothing calls the
  `Validate` methods on the `nathejk/table` filter types. The only
  `.Validate(v)` callers in the repo are `data.Product` and `data.Filters` in
  `cmd/api/home.go`. Leaving the methods in place regardless — deleting unused
  API is a separate decision from inverting a dependency.
- 2026-08-04 13:00 — ✅ Criterion 1: `nathejk/table/interfaces.go` declares
  `Validator` and `PermittedValue`. Declared `Validator` with only `Check`
  rather than mirroring `internal/validator.Validator`'s four methods — that is
  all the six `Validate` methods use, and the caller keeps `Valid`/`Errors` to
  itself. Put it in the parent package rather than copying it into five
  sub-packages, which the import graph permits: the five that need it already
  import the parent as `tables`, and the parent does not import any of them.
- 2026-08-04 13:05 — ✅ Criterion 2: `signup/interfaces.go` declares
  `SmsSender` and `Mailer`. Pure inversion, no adapter: the concrete clients
  already have these exact signatures, so `main.go` did not change at all.
- 2026-08-04 13:30 — ✅ Criterion 3, the substantial one.
  `payment/interfaces.go` declares a `Provider` port in Nathejk's vocabulary —
  `Amount`, `PaymentRequest`, `PaymentCreated`, `Authorization` — and
  `cmd/api/mobilepayprovider.go` adapts the MobilePay client to it. Chose a
  purpose-shaped port over mirroring `mobilepay.Client`: the commands used only
  a small fraction of MobilePay's payment model, and mirroring it would have
  moved the coupling rather than removed it. Everything MobilePay-shaped (the
  WALLET method, WEB_REDIRECT flow, the `aggregate` amount fields) is now in
  the adapter. The adapter lives in `cmd/api` deliberately: neither package it
  joins should know about the other.
- 2026-08-04 13:35 — Verified the `Capture` rewrite is behaviour-preserving.
  The original took `mpp.Amount` and overwrote `.Value` with
  `authorized - captured`; the port version constructs
  `Amount{Currency: auth.Currency, Value: auth.AuthorizedAmount -
  auth.CapturedAmount}`, and the adapter maps `auth.Currency` from
  `mpp.Amount.Currency`. Same value, same guard
  (`!Authorized || Value <= 0`).
- 2026-08-04 13:50 — Added `payment/commands_test.go`: 8 cases over a fake
  `Provider`, covering the partial-capture remainder arithmetic, the three
  no-op paths (unauthorised, fully captured, over-captured), event ordering
  around the capture, and that a failed authorisation publishes nothing. These
  paths were previously reachable only against MobilePay's live HTTP API —
  writing them is the concrete payoff of the port.
- 2026-08-04 13:55 — ✅ Criteria 4–5. `go list -deps ./nathejk/table/...` now
  reports `nathejk.dk/internal/cqrs` as the entities' only non-`table`
  dependency. `gofmt`, `go build`, `go vet`, `staticcheck` and `go test ./...`
  all clean. Confirmed none of the three internal packages became orphaned:
  `validator` is still used by `cmd/api/app` and `internal/data`, `mailer` and
  `sms` by `main.go`, `mobilepay` by the new adapter and `routes.go`. Moving
  to done.

## Outcome

Three `interfaces.go` files, one adapter, and the entity packages are free of
the repo:

| Port | Declared in | Satisfied by |
|---|---|---|
| `Validator`, `PermittedValue` | `nathejk/table/interfaces.go` | `internal/validator` (structurally) |
| `SmsSender`, `Mailer` | `nathejk/table/signup/interfaces.go` | `internal/sms`, `internal/mailer` (structurally) |
| `Provider` + value types | `nathejk/table/payment/interfaces.go` | `cmd/api/mobilepayprovider.go` (adapter) |

Two of the three needed no adapter and no wiring change. Only payment did,
because a provider's client speaks its own types.

`nathejk/table/...` is now extractable to `shared-go`: its only remaining
non-`table` dependency is `internal/cqrs`, which is itself queued for
extraction (see 022). The two moves should happen together, or `shared-go`
will briefly need to depend on a `nathejk.dk/internal` path, which is
impossible — Go forbids importing another module's `internal` tree.

Noted but not fixed: `payment.Request` builds its callback URL from a
hard-coded `https://tilmelding.nathejk.dk` host, so a non-production
deployment returns the payer to production. It pre-dates this task and fixing
it means threading the base URL into the commander (`cfg.baseurl` is already
available in `main`). Left as a `TODO` at the site.
