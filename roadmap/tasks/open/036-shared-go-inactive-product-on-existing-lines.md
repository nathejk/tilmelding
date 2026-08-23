# 036 — shared-go: allow an inactive product on a line that is not a new sale

**Status:** open
**Priority:** low
**Created:** 2026-08-23

## Description

Phase 2 of PRD 003, part one. Prerequisite for task 037.

`product.active` is the catalogue's own way of saying "this is not for sale", and
PRD 003 wanted to use it. It could not (see PRD 003 §8.5), so closing is currently
driven by a `CLOSED_PRODUCT_SKUS` env var in tilmelding instead. This task removes
the obstacle so the catalogue can become the source of truth.

**The obstacle**

`order.buildLines` (shared-go `tables/order/commander.go`, the `!p.Active` check
around L424) rejects **any** desired line whose product is inactive:

```go
if !p.Active {
    return nil, fmt.Errorf("%s: %w", d.ProductSKU, ErrProductInactive)
}
```

Every tilmelding request re-derives the full desired set — participation *and*
merchandise — from the projection, so the moment `tshirt.adult` goes inactive,
every owner who still has a shirt line on their open order presents a set the
commander refuses. Saves fail outright and show handlers log the failure and
render a stale order.

The comment on `product.GetBySKU` already claims the system behaves otherwise:

> Inactive products are still returned — callers that need to enforce "buyable"
> should check Product.Active themselves; the order commander does this so that
> **retiring a product doesn't break already-issued orders** that still reference
> it.

That last clause is the intent this task implements. Today the check is absolute,
so retiring a product breaks exactly those orders.

**Proposed rule**

Inactive means "no new units", not "this line may not be re-derived". In
`buildLines`, tolerate an inactive product when the line cannot be a new sale:

1. **A line with the same `LineID` already exists on `o.Lines`.** Derived line ids
   are deterministic (`derived:{sku}:{memberId}[:{size}]`, see `defaultLineID`), so
   this is a map lookup: the same unit, being re-derived, not a new one.
2. **The quantity is negative.** A credit hands a paid unit back (the reclaim half
   of a zero-sum size change); it is the opposite of a sale.

Reject in every other case, so `ErrProductInactive` keeps meaning "you cannot start
a new one". Scope the tolerance to `messages.LineOriginDerived`: `AddManualLine`
must keep refusing outright, because a manual add is always new by definition.

**Explicitly out of scope**

- **`checkStock` / `stock = 0`.** It has the same shape of problem, but `stock = 0`
  is a different statement ("no inventory") from `active = 0` ("not sold any
  more"), and participation SKUs already have bespoke overflow semantics there
  (`KindParticipation` passes whenever any stock remains). Leave it; task 004
  (per-size t-shirt inventory) is the place where stock semantics get revisited.
- **Re-pricing.** A re-derived line takes the catalogue's current `UnitPrice`, as
  it does today for active products. A closed product's price should not be moving
  anyway; do not add special handling.

**Note on why tilmelding still needs its own filter afterwards.** This change only
stops inactive products from *erroring*. It does not cancel unpaid units — that is
`app.sellable` in tilmelding, and it stays. Task 037 replaces the env var, not the
filter.

## Sequencing

Same as tasks 028 and 029: this lands in shared-go first (push, tag), then
tilmelding bumps `go.mod`, then task 037 wires it. Local dev picks the change up
immediately via `go.work` and the `../shared-go:/shared-go` mount, so the two can
be developed together but must be committed in that order.

## Acceptance Criteria

- [ ] `buildLines` tolerates an inactive product for a derived line whose `LineID`
      is already on the order
- [ ] `buildLines` tolerates an inactive product for a negative-quantity derived
      line
- [ ] `buildLines` still returns `ErrProductInactive` for a genuinely new derived
      line
- [ ] `AddManualLine` still returns `ErrProductInactive` unconditionally
- [ ] Unit tests in shared-go for all four cases above
- [ ] `SyncNeeded` and `SetDerivedLines` agree on an order holding an inactive
      product — i.e. a repeated show does not republish (this is the failure mode
      that would otherwise be invisible until it hit production as an event storm)
- [ ] Existing shared-go order tests pass unchanged
- [ ] tilmelding's `go.mod` bumped and its suite green in the workspace and with
      `GOWORK=off`

## Progress Log

- 2026-08-23 — Task created as phase 2 of PRD 003 (§8.5 documents why phase 1 could
  not use `active`). Low priority: phase 1 shipped and the shop is closed, so this
  is a mechanism cleanup rather than a fix. If the semantics turn out to need
  broader agreement than a task can carry, escalate to a PRD in shared-go's own
  `roadmap/prd/` — it already has one (its PRD 001 is this repo's PRD 002).
