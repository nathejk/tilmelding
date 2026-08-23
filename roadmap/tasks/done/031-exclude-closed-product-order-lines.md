# 031 — exclude closed-product lines from derived order lines

**Status:** open
**Priority:** high
**Created:** 2026-08-23

## Description

The behavioural core of PRD 003. While a product is closed, its **unpaid** units
stop being billed: the line falls off the open order and the amount due drops.
Paid units are untouched.

Depends on task 030 (`app.skuClosed`).

**Where to apply it — the chokepoint, not the call sites**

A desired set is *built* in eleven places (four show handlers,
`updateKlanHandler:429`, `updatePatruljeHandler:466`, `rederiveKlanOrder:509`,
`rederivePatruljeOrder:565`, `crew.go:78` and `:158`, `personnel.go:43` and `:97`)
but only *consumed* in two, both in `go/cmd/api/orders.go`:

- `app.syncNeeded` (L66) — decides whether a GET publishes anything
- `app.setDerivedLinesAfterCreate` (L174) — performs the write

Filter in those two and the sets cannot diverge. That matters: if the sync check
and the write disagree, `SyncNeeded` returns true forever and the order
republishes its lines on **every page load**. Filtering at eleven call sites would
make that a convention; filtering here makes it structural.

```go
// sellable drops lines for products that are closed for sale, unless the order
// has money in flight.
func (app *application) sellable(o *order.Order, lines []order.DesiredLine) []order.DesiredLine
```

`setDerivedLinesAfterCreate` currently takes an `orderID` and needs the order
itself for the guard, so its signature becomes `(ctx, o *order.Order, desired)`.
Every call site already has the order in scope — mechanical change.

**The in-flight guard**

Skip the filter entirely when `o.PaidAmount > 0`. That order has money reserved or
received against it and must be left exactly as its payer saw it. `paidAmount` is
computed on the order projection as the sum of payments in
`('reserved','received')` for that order (`orderColumns` in shared-go
`tables/order/querier.go:102-104`), so this needs no payment query and no
shared-go change.

`requested` deliberately does **not** count. Every save with `due > 0` issues a
payment request, so treating a request as in-flight would exempt nearly every
order and cancel nothing. It is also indistinguishable from an abandoned link:
nothing writes `timedout` or `rejected` — the payment projector
(`shared-go/tables/payment/consumer.go`) only subscribes to `.requested`,
`.reserved`, `.received`, so an abandoned request stays `requested` forever. See
PRD 003 §8.2b.

**Why paid shirts are safe** — verified, do not re-litigate: `ApplyPaidOffset`
(shared-go `tables/order/commander.go:684`) matches desired units against paid
units per `(SKU, size)`, and leftover *paid* units produce nothing: "a reduction is
not a size change, and this mechanism does not refund." Feeding it a set with no
`tshirt.adult` lines therefore emits **no credit lines** for paid shirts. Paid
orders are immutable anyway (`ErrNotOpen`), and `loadOrders` only re-derives the
open one.

**Leave alone:** `requestSeatHandler` (`klan.go:348`) calls `SetDerivedLines`
directly with participation-only placeholder lines. It cannot carry a merchandise
SKU; add a comment saying that is why it bypasses the filter.

## Acceptance Criteria

- [ ] `sellable` implemented and applied in both `syncNeeded` and
      `setDerivedLinesAfterCreate`; no producer function changed
- [ ] `setDerivedLinesAfterCreate` takes the order rather than an order id
- [ ] Unit tests: closed SKU dropped; other SKUs untouched; `PaidAmount > 0` skips
      the filter; empty closed set is a no-op returning the input
- [ ] A member with an **unpaid** shirt: line gone from the open order, due amount
      down by 175 kr
- [ ] A member with a **paid** shirt: paid order unchanged, and **no negative
      `tshirt.adult` line** appears anywhere
- [ ] A GET on a team with no unpaid shirt publishes **no** `lines.changed` —
      verify by watching the stream across two page loads
- [ ] A GET is stable: two consecutive loads produce identical order lines
- [ ] `go build ./...` / `go test ./...` pass in the workspace and with `GOWORK=off`

## Progress Log

- 2026-08-23 — Task created from PRD 003 §8.1/§8.2. Depends on 030. Pairs with 032
  — without that gate, an order exempted by the in-flight guard can still mint a
  new payment link containing the closed product.
