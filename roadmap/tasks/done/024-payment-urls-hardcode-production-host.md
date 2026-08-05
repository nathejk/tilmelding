# 024 — Payment URLs hard-code the production host

**Status:** done
**Priority:** medium
**Created:** 2026-08-04
**Completed:** 2026-08-04
**Picked up by:**
**Started:**
**Completed:**

## Description

Six places build user-facing URLs by concatenating a hard-coded
`https://tilmelding.nathejk.dk` rather than using the already-configured
`cfg.baseurl`. The consequence is that **the payment flow cannot be exercised
outside production**.

`cfg.baseurl` already exists for exactly this purpose — it is a flag with a
`BASEURL` environment fallback (`cmd/api/main.go`) and is already threaded into
the mailer as a template variable. Production is unaffected, because there
`cfg.baseurl` happens to equal the hard-coded string. Everywhere else is broken.

### The consequential one: the MobilePay callback

`nathejk/table/payment/commands.go`:

```go
CallbackURL: "https://tilmelding.nathejk.dk/callback/mobilepay/" + reference,
```

This is the URL MobilePay returns the payer to after they approve a payment,
and that route (`/callback/mobilepay/:ref` → `mobilepayCallbackHandler`) is what
triggers `Payment.Capture`. So on any non-production deployment:

- the payer is redirected to the production site after paying;
- the *staging* capture never runs;
- production receives a callback for a reference it did not create. Depending
  on whether the two environments share MobilePay credentials, it either
  errors or captures a payment it knows nothing about.

### The cosmetic ones: post-payment return URLs

Five handlers build the `returnUrl` that is stored on the payment row and
published on `NathejkPaymentRequested`:

| File | Line | URL |
|---|---|---|
| `cmd/api/klan.go` | 159 | `/klan/{teamID}` |
| `cmd/api/klan.go` | 242 | `/klan/{teamID}` |
| `cmd/api/patrulje.go` | 244 | `/patrulje/{teamID}` |
| `cmd/api/personnel.go` | 120 | `/badut/{userID}` |
| `cmd/api/crew.go` | 170 | `/crew/{userID}` |

These send the user to the wrong host after paying. Same root cause, trivial
fix: `app.config.baseurl` is already in scope in all five.

## Approach

The five handler sites are a straight substitution. The callback deserves a
decision, because there are two ways to do it and one is better:

**Option A — thread the base URL into the commander.** Add a
`payments.WithBaseURL(...)` option and have `Request` keep building the URL.
Smallest diff.

**Option B — move the callback URL out of the domain entirely.** Drop
`CallbackURL` from `payments.PaymentRequest` and let
`cmd/api/mobilepayprovider.go` build it from the reference. The adapter is
constructed in `main`, where `cfg.baseurl` is in scope, so it becomes
`newMobilepayProvider(paymentClient, cfg.baseurl)`.

Option B is preferred. The path `/callback/mobilepay/` names a specific
provider, so it is provider-specific routing that ended up in a domain package
by accident. The domain's job is to ask for a payment; how the payer finds
their way back is a property of the provider integration. This also shrinks the
port introduced in task 023 by one field.

Note that a correct `CallbackURL` must be reachable by MobilePay from the
public internet, so a local dev environment needs a tunnel or a public
hostname regardless — fixing this makes staging work, not localhost.

## Acceptance Criteria

- [x] No Go file outside tests builds a URL from a hard-coded
      `https://tilmelding.nathejk.dk`.
- [x] The MobilePay callback URL derives from `cfg.baseurl`.
- [x] The five `returnUrl` sites derive from `cfg.baseurl`.
- [x] Setting `BASEURL=https://staging.example.com` produces callback and
      return URLs on that host (verify by unit test or by inspecting the
      `NathejkPaymentRequested` event, not by a live payment).
- [x] `nathejk/table/payment/commands_test.go` no longer asserts a production
      hostname it does not control.
- [x] `go build ./...` and `go test ./...` pass.

## Progress Log

- 2026-08-04 14:05 — Task created. Found while inverting the payment provider
  dependency in task 023: the `CallbackURL` line was carried over unchanged to
  keep that refactor behaviour-preserving, with a `TODO` left at the site.
  Grepping for the host turned up five more instances in the handlers, so this
  is a pattern rather than a one-off. Production is unaffected — `cfg.baseurl`
  defaults to the same value — which is presumably why it has survived.
- 2026-08-04 — Done, Option B. Dropped `CallbackURL` from
  `payments.PaymentRequest`: `/callback/mobilepay/` names a provider, so
  building it belongs in the adapter, not the domain. `mobilepayProvider` now
  takes a `baseURL` (trailing slash trimmed) and derives the callback as
  `baseURL + "/callback/mobilepay/" + reference`; `main.go` passes
  `cfg.baseurl`. This also shrinks the port by one field.
  The five handler `returnUrl` sites (klan ×2, patrulje, personnel, crew) now
  use `app.config.baseurl` instead of the literal host.
  Added `cmd/api/mobilepayprovider_test.go` with a fake `mobilepay.Client`
  proving the callback tracks `BASEURL` (production, staging, trailing-slash)
  — the AC's staging check, without a live payment. Removed the now-invalid
  `CallbackURL` assertion from the payment commander test. Confirmed no
  hard-coded host remains in non-test Go outside the `BASEURL` flag default.
  Note (unchanged, out of scope): a working callback must be reachable by
  MobilePay from the public internet, so local dev still needs a tunnel — this
  fixes staging, not localhost.
