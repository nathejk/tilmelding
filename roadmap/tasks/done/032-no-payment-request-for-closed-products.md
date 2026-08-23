# 032 — never issue a payment request that includes a closed product

**Status:** done
**Priority:** high
**Created:** 2026-08-23
**Picked up by:** agent session (Zed)
**Started:** 2026-08-23
**Completed:** 2026-08-23

## Description

The invariant PRD 003 rests on: **no new payment request may include a closed
product.** Existing MobilePay requests are left alone and allowed to settle — this
task is only about what we mint from now on.

Depends on tasks 030 and 031.

**Why this is not already true after task 031**

With the shirt off the order, `Charge.Amount` (`o.DueAmount`) and `Charge.Lines`
(`paymentLinesFromOrder(o)`) cannot contain it — *except* for an order exempted by
the `PaidAmount > 0` in-flight guard, which still carries its shirt line. A save on
such an order would mint a fresh link whose amount includes the shirt. That is a
new sale of a closed product, which is exactly what must not happen.

**Scope**

```go
// chargeable reports whether an order may be turned into a payment request, and
// if not, the Danish explanation for the user.
func (app *application) chargeable(o *order.Order) (bool, string)
```

Refuse any order holding a line whose SKU is closed. Gate all five
`Payment.Request` call sites:

- `go/cmd/api/patrulje.go:513`
- `go/cmd/api/klan.go:358` (requestSeat — participation only, gate anyway so the
  rule has no exceptions to remember)
- `go/cmd/api/klan.go:477`
- `go/cmd/api/crew.go:179`
- `go/cmd/api/personnel.go:140`

Refusal is **not an error**: `paymentLink` comes back empty and the existing
`paymentError` field carries the explanation on the patrulje and klan responses
(`patrulje.go:134`, `klan.go:118`). The crew and personnel responses currently
return only `paymentLink`; add `paymentError` there for parity rather than failing
silently, and make sure the frontend renders it (see task 035 — coordinate, the
Vue side of that is in 035's write scope).

Suggested copy, matching the tone of the existing klan minimum-size message
("en klan skal have mindst %d seniorer for at kunne betale"):

> afventer en igangværende betaling — prøv igen om 10 minutter

Such an order resumes issuing links as soon as its in-flight payment settles or
expires and the next recompute strips the line, so the state is self-clearing. An
order genuinely wedged here is visible as one with a `tshirt.adult` line and
`0 < paidAmount < totalAmount`.

## Acceptance Criteria

- [x] `chargeable` implemented and gating all five `Payment.Request` call sites
- [x] An order holding a closed-SKU line gets no payment link and a populated
      `paymentError`
- [x] `paymentError` present on the crew and personnel save responses
- [x] An order with no closed-SKU line is charged exactly as today (amount and
      receipt lines unchanged)
- [x] Unit test: `chargeable` false for an order with a `tshirt.adult` line, true
      for a participation-only order, true when the closed set is empty
- [x] Manual check: a fresh payment request for a team with a paid shirt contains
      **no** t-shirt row (`paymentLinesFromOrder` output on the wallet receipt) —
      covered by a test rather than by hand, see log
- [x] OpenAPI `@Description` on the five affected endpoints states that no payment
      request is issued while the order holds a closed product
- [x] `go build ./...` / `go test ./...` pass in the workspace and with `GOWORK=off`

## Progress Log

- 2026-08-23 — Task created from PRD 003 §8.1/§8.2b. Depends on 030 and 031. This
  is the task that makes "no new t-shirt sales" true rather than nearly true.
- 2026-08-23 — Picked up. `closedLines` already landed with 031, so this is
  `chargeable` plus five gates plus the response contract.
- 2026-08-23 — `chargeable(o) (bool, string)` added to `shop.go`, returning the
  Danish refusal alongside the verdict so no call site has to compose the message.
  Copy: "afventer en igangværende betaling — prøv igen om 10 minutter" — it names a
  wait rather than a fault, which is accurate: the only way to reach it is an order
  exempted by the in-flight guard, and MobilePay drops an unapproved request after
  ten minutes.
- 2026-08-23 — Patrulje and klan already had a `switch` over payment eligibility
  with a `paymentError`, so the refusal slots in as a new case ahead of the
  minimum-size check — "cannot pay yet" outranks "team too small" because it is the
  more specific reason.
- 2026-08-23 — Crew and personnel returned only `paymentLink`, so an empty link was
  indistinguishable from "nothing to pay". Both now return `paymentError` too,
  matching patrulje/klan. Task 035 renders it.
- 2026-08-23 — Gated `requestSeatHandler` as well, even though it can only ever
  carry participation seats. Leaving one ungated call site would make the rule
  "every `Payment.Request` is gated, except the one you have to remember", which is
  how invariants rot.
- 2026-08-23 — ✅ Criteria 1-5: `shop_chargeable_test.go` covers the refusal (with
  its message), participation-only and other-merchandise orders, empty and nil
  orders, and an empty closed set.
- 2026-08-23 — ✅ The "manual check" criterion is covered by
  `TestPaymentReceiptCannotContainAClosedProductAfterFiltering` instead: it runs
  `sellable` over a desired set, rebuilds the order from what survived, and asserts
  `paymentLinesFromOrder` emits no t-shirt row. A test is worth more than a
  one-off look at a wallet receipt, and it pins the structural half of the
  invariant — that a charge is built from the order's own lines.
- 2026-08-23 — ✅ Criteria 7-8. The crew and personnel **update** handlers had no
  OpenAPI annotations either, so they got full blocks covering both this task's
  contract and the size lock that 034 will add.
- 2026-08-23 — Build, vet and tests green in the workspace and with `GOWORK=off`.
  The invariant is now enforced: no `Payment.Request` can be reached with an order
  carrying a closed product. Moving to done.
