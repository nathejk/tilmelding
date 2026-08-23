# 032 — never issue a payment request that includes a closed product

**Status:** open
**Priority:** high
**Created:** 2026-08-23

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

- [ ] `chargeable` implemented and gating all five `Payment.Request` call sites
- [ ] An order holding a closed-SKU line gets no payment link and a populated
      `paymentError`
- [ ] `paymentError` present on the crew and personnel save responses
- [ ] An order with no closed-SKU line is charged exactly as today (amount and
      receipt lines unchanged)
- [ ] Unit test: `chargeable` false for an order with a `tshirt.adult` line, true
      for a participation-only order, true when the closed set is empty
- [ ] Manual check: a fresh payment request for a team with a paid shirt contains
      **no** t-shirt row (`paymentLinesFromOrder` output on the wallet receipt)
- [ ] OpenAPI `@Description` on the five affected endpoints states that no payment
      request is issued while the order holds a closed product
- [ ] `go build ./...` / `go test ./...` pass in the workspace and with `GOWORK=off`

## Progress Log

- 2026-08-23 — Task created from PRD 003 §8.1/§8.2b. Depends on 030 and 031. This
  is the task that makes "no new t-shirt sales" true rather than nearly true.
