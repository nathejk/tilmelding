# PRD 002 — Size-aware paid offset and zero-sum credit lines

**Status:** doing
**Author:** agent session (Zed), on report from Knud
**Created:** 2026-08-20
**Last updated:** 2026-08-21
**Approved:** 2026-08-20
**Shipped:**
**Target users:** organizer (fulfillment / warehouse), participant, gøgler, crew

<!--
Status must match the folder this file is in: draft/, doing/ or done/.
Leave Approved blank until the PRD moves to doing/, and Shipped blank until it
moves to done/. See roadmap/prd/README.md for the lifecycle.
-->

> **This document is dual-homed, and it has round-tripped.** It was authored here
> as **tilmelding PRD 002**, copied to `shared-go` as **shared-go PRD 001** (the
> first PRD in that repo) so the substantive `tables/order` work could be done
> there, and copied back with the implementation record in §0 and §8.1a. The
> numbers differ per repo and both are permanent: a PRD's number never changes
> once assigned (`roadmap/prd/README.md`), so this is **tilmelding PRD 002 =
> shared-go PRD 001**. In this repo, cite it as "PRD 002".
>
> **Beware three different documents called "PRD 001":**
>
> | Reference | Document |
> |---|---|
> | tilmelding PRD 001 | Seat-based billing and member identity (`roadmap/prd/done/001-…`) — the one this PRD amends |
> | shared-go PRD 001 | *This document*, under its shared-go number |
> | tilmelding PRD 002 | *This document*, under its number here |
>
> tilmelding PRD 001 remains correct on money; this PRD fixes what it left unsaid
> about fulfillment. Do not re-litigate its seat model here.
>
> **Canonical copy: this one.** The shared-go work is complete, so every
> outstanding item (§8.2) belongs to this repo — keep this file current and treat
> shared-go's copy as a frozen implementation record. Two live copies is what
> caused the drift being corrected here; one live copy is the fix. Sections
> 8.1–8.3 split the work by repo.

---

## 0. Implementation status

**shared-go: complete** — commit `016cb07` on `main`, fetchable as
`v0.0.0-20260819231649-016cb07ccb98`. §8.1 is built, with the deviations recorded
in §8.1a.

**tilmelding: complete, verified locally, not yet committed or deployed** — §8.2
landed as specified and turned out smaller than first scoped, because the sync
comparison moved into shared-go. `derivedLinesNeedSync` is gone, replaced by a
thin `app.syncNeeded` adapter over `order.Commands.SyncNeeded`; the receipt nets
out zero-sum pairs via `netOutCredits`; the widened `quantity` contract is
documented on `orderLineResponse` and on the two show endpoints' `@Description`.

**Plus one change §8.2 did not anticipate:** the four views hid the open-order
block whenever nothing was due, so the credit pair was correct in the API and
still invisible on the page — the original bug report, one layer up. §7 records
the corrected gating.

Verified against the reported record on the dev stack — the open order now reads:

```
tshirt.adult  qty +1  17500  {"size":"3xl"}
tshirt.adult  qty −1 −17500  {"size":"xxl"}
total 0   due 0   paymentLink ""
```

which is exactly the `(-1 XXL, 1 XXXL)` the original report expected. Also
confirmed: five consecutive GETs leave `changedAt` unmoved (no republish, so the
oscillation risk is closed in practice, not just by construction); changing the
size back collapses the order to zero lines; a 43-line patrulje order with no
size changes is untouched and still due 925000; and across the whole dev database
there is exactly one credit line and no order with a negative total.

This PRD stays `doing` until the change is committed and deployed — §9's metrics
are verified on dev, not in production.

**Outstanding: none in shared-go.** `Commands.Settle` landed in
`v0.0.0-20260820141337-434b63e177b7` exactly as §8.1b specified, including
`ErrEmptyOrder` / `ErrOrderNotFree` and idempotence for an already-closed order.

**Settling in tilmelding: built and verified locally** (§8.2b). Full cycle proven
on the reported record:

```
browse to 3xl (settle omitted)  → cart stays OPEN: +1 3xl, −1 l, total 0
save 3xl      (settle=true)    → cart null; order moves to paid history at 0,-
save again    (settle=true)    → 200, no error; a fresh empty open order
```

And the arithmetic the whole design rests on: after settling an exchange, the
*next* size change reclaims the newly settled size, not the original one —
`−1 l, +1 m` rather than anything touching the long-paid `xxl`. Net across every
non-cancelled order for that owner is `1 × participation.gogler` + `1 × tshirt
3xl`, which is what they should receive. One open order per owner throughout.

Two defects were found and fixed while verifying — see §8.2b.

The consumer audit that §10 flagged as able to invalidate the design is done and
did not — see §11 "Decided".

---

## 1. Summary

When a participant changes their t-shirt size after paying, the change is
correctly free — but it also becomes invisible: the order aggregate still says
the old size and nothing anywhere in the order data says the new one. Make the
paid-unit offset size-aware, and represent a free size change as a zero-sum pair
of derived lines (`−1 old size`, `+1 new size`) so the order shows what actually
has to be shipped while still charging nothing.

## 2. Problem & Motivation

**What problem does this solve?** The order aggregate is the only place
fulfillment can read "which shirts does this team get". After a post-payment
size change it holds the wrong answer, with no indication that anything changed.

Concretely, gøgler `f789d44f-3dfb-4e8f-8143-13ae521367fa` on the local dev
dataset changed XXL → XXXL. Current state:

```
personnel.tshirtSize = 3xl

order d462517e… (paid, totalAmount 27500)
  participation.gogler  qty 1  10000
  tshirt.adult          qty 1  17500  {"size":"xxl"}

order d5cb8c4c… (open, totalAmount 0)
  (no lines at all)
```

`3xl` appears nowhere in the order data. Picking from orders ships an XXL.

**Root cause.** `order.PaidQuantityBySKU` groups paid quantity by `productSku`
only, deliberately ignoring attributes — a paid unit is fungible. So
`ApplyPaidOffset` cancels the desired `tshirt.adult {3xl}` line against the paid
`tshirt.adult {xxl}` unit and the open order ends up empty. The pricing outcome
is exactly what tilmelding PRD 001 specifies ("changing sizes on the existing paid t-shirts
stays free"); the flaw is that it implemented *free* as *absent*.

`derivedLinesNeedSync` applies the same offset before comparing, so it correctly
reports "in sync" — the GET is a pure read and the system is stably wrong. This
is not a transient projection lag; re-visiting the page will never repair it.

**Why now?** Sizes are being edited on live signups today, and every edit after
payment silently mis-picks one shirt. The failure is silent on both sides: the
participant sees a page that looks settled, and fulfillment sees a plausible
order. There is no log line, no reconciliation error, nothing to notice.

**Evidence.** User-reported on `/badut/f789d44f-3dfb-4e8f-8143-13ae521367fa`,
verified against the dev database (query output above). tilmelding PRD 001 §"T-shirt
accounting" and its line "Size is a fulfillment attribute" name the concept but
never nominate a read model, which is the gap this PRD closes.

## 3. Goals

- The order aggregate alone is sufficient to fulfill a team's t-shirts correctly,
  including after any number of post-payment size changes.
- A size change on an already-paid t-shirt continues to cost the payer exactly 0.
- The change is legible: someone reading the order can see that one XXL is
  reclaimed and one XXXL is owed, without diffing against another projection.
- Existing orders self-heal on their next GET, with no migration or manual
  intervention.

## 4. Non-Goals

- **Per-size stock accounting.** `product.Stock` is a single per-SKU number
  today; `Sizes` is only a list of offered slugs. tilmelding PRD 001's "subject to
  available stock for the chosen size" is therefore not implementable, and is
  explicitly out of scope here. **Per-size inventory is a planned feature**, not
  a mistake there — so this PRD must not paint it into a corner: the lines
  it emits carry the size on every unit, positive and negative, which is exactly
  the granularity a future per-size stock check will need. When that feature
  lands, `checkStock` gains a variant dimension and that size-stock rule
  becomes enforceable without revisiting the representation designed here.
- **Changing what anything costs.** No pricing, seat-count, or due-amount
  behaviour changes. This is a representation fix.
- **Mutating paid orders.** A paid order is the payer's receipt and stays
  byte-identical. All adjustment lives on the open order.
- **A general credit-note / refund feature.** Negative lines introduced here are
  strictly internal, derived, and always net to zero against a positive line of
  the same SKU. Refunding money is a separate problem.
- **Reassigning a paid t-shirt between members.** Already free and already
  correct; member identity is not part of the fulfillment question.
- **Backoffice UI for the size mix.** Out of scope beyond the participant-facing
  order display already rendered in `tilmelding`.

## 5. User Stories & Scenarios

- As a **gøgler**, I want to change my t-shirt size after paying and see that my
  order now says XXXL, so that I trust I will receive the right shirt.
- As an **organizer packing shirts**, I want the order to tell me one XXL is
  reclaimed and one XXXL is owed, so that I ship what people actually asked for.

**Happy path (single member, the reported case).** Gøgler has a paid order with
`tshirt.adult {xxl}`. They pick XXXL. `PATCH` → `SetDerivedLines` computes the
desired set (`participation.gogler`, `tshirt.adult {3xl}`), offsets it against
paid units keyed by `(sku, size)`, finds one unmatched paid `xxl` and one
uncovered desired `3xl` for the same SKU, and pairs them:

```
open order (totalAmount 0)
  tshirt.adult  qty −1  lineTotal −17500  {"size":"xxl"}
  tshirt.adult  qty +1  lineTotal +17500  {"size":"3xl"}
```

`dueAmount` stays 0, so no payment link is minted and the page shows nothing
owed — but the lines now state the swap.

**Edge cases.**

1. **Idempotence.** A second GET must produce the identical set and
   `derivedLinesNeedSync` must return false. The pairing has to be
   deterministic, not map-iteration-dependent, or every GET republishes a
   `lines.changed` event.
2. **Repeated changes.** XXL → XXXL → L must leave exactly `−1 xxl`, `+1 l` —
   never a chain of stale credits. The desired set is recomputed from scratch
   each time, so this follows if the algorithm is a pure function of
   (desired, paid).
3. **Change back.** XXL → XXXL → XXL must collapse to no lines at all.
4. **Team, partial change.** Klan paid 4 × `tshirt.adult` (2 × l, 2 × xl). One
   xl member switches to m. Expect `−1 xl`, `+1 m`, and the three unchanged
   shirts absent. Due 0.
5. **Change plus growth.** Klan paid 4 shirts, now wants 5 *and* one size
   changed. Expect the zero-sum pair plus one full-price 5th shirt; due = one
   shirt. The credit must not be applied to the genuinely new unit.
6. **Sizeless SKUs.** `participation.*` lines carry no `size` attribute. They
   must keep behaving exactly as today (offset on `(sku, "")`), and must never
   generate credit lines.
7. **Unpaid order.** No paid units → no offset, no credits, current behaviour.
8. **Shrinking below paid count.** Team drops from 4 paid shirts to 3. Today's
   behaviour is "no lines, no refund"; that is preserved — a *reduction* is not
   a size change and must not emit a lone `−1` credit.

## 6. Requirements

### Functional

Ticked items are implemented in shared-go (`016cb07`); unticked ones depend on
the `tilmelding` work in §8.2.

- [x] Paid units are counted per `(productSku, size)` variant, not per
      `productSku`. Lines with no `size` attribute count as the `""` variant.
- [x] Desired lines are offset against paid units of the **same variant** first,
      and those are dropped from the open order as they are today.
- [x] For each SKU, after same-variant matching, an uncovered desired unit is
      paired with a still-unmatched paid unit of a *different* size, producing
      two derived lines: the desired line at `+quantity`, and a credit line for
      the reclaimed size at `−quantity`. Both use the SKU's catalogue unit
      price, so the pair sums to 0.
- [x] Uncovered desired units beyond the SKU's total paid count are charged at
      full price with **no** credit line.
- [x] Unmatched paid units left over after all desired units are covered produce
      **no** lines (no refunds — edge case 8).
- [x] Pairing is deterministic: for a given (desired, paid) input the output is
      byte-identical across runs and across processes. Paid sizes are consumed in
      catalogue `Sizes` order, with slugs absent from the catalogue last by slug.
- [x] Both the positive and the negative line carry the `size` attribute, so a
      future per-size stock check has the granularity it needs (§4).
- [x] A credit line carries a non-empty `MemberID` — that of the desired line it
      offsets — so it passes `buildLines` validation.
- [x] `buildLines` accepts `Quantity < 0`, computing `LineTotal = unitPrice ×
      quantity` (negative). `Quantity == 0` continues to mean "line absent".
- [x] `defaultLineID` for derived lines includes the size variant, because a
      credit line and a charge line for the same SKU and member must not
      collide on one `lineId`. *(Sized products only — see §8.1a.)*
- [x] `checkStock` evaluates positive quantities only. A credit line must not
      create stock headroom for the paired new size.
- [x] The sync check keys on quantity as well as `(sku, memberId, size)`, so a
      `−1 xxl` is never mistaken for a `+1 xxl`. *(Now `Commands.SyncNeeded` in
      shared-go rather than `derivedLinesNeedSync` in the BFF — see §8.1a.)*
- [x] The payment receipt omits zero-sum pairs rather than sending negative line
      items to the payment provider, and still reconciles against the charged
      amount. *(`netOutCredits` in `paymentlines.go`; cancellation is by unit
      count per SKU, so both halves of a pair leave together and the sum is
      preserved. An unpaired credit — impossible if the offset holds — falls back
      to an unfiltered receipt rather than a silently wrong total.)*
- [x] An order left in the broken state by the current code self-heals on its
      next GET, via the existing sync-check → `SetDerivedLines` path. No
      migration. *(Confirmed on the reported record: a single GET produced the
      pair.)*
- [x] `dueAmount` for a pure size change is 0, and no payment link is minted.
      *(Asserted in `TestSetDerivedLinesRecordsAFreeSizeChange`; the "no link"
      half is the BFF's existing due-amount guard.)*

#### Added during implementation

- [x] Sizes are normalised (trimmed, lowercased) at every comparison point, so a
      casing difference between producers cannot masquerade as a size change.
- [x] `Queries.ShippableByVariant` gives fulfillment the net variant mix, so the
      size-authority decision (§11) has one implementation rather than one per
      consumer.

#### Amendment 2026-08-20: settling zero-value orders

A free size change leaves behind a non-empty open order that totals 0. Nothing
ever closes it: `StatusPaid` is reached only by the payment saga, which requires
a payment, and a zero-value order will never receive one — it even guards
`TotalAmount <= 0` explicitly. So the exchange sits in an editable cart forever
rather than being recorded as a completed change, and the page shows a permanent
open order the user cannot action.

- [x] An open order with at least one line and `TotalAmount == 0` can be settled:
      it transitions to `StatusPaid` with `PaidAmount == 0`, becoming immutable
      and moving into the owner's paid history.
- [x] Settling is driven by the user's explicit save, not by the read path. A GET
      must never transition an order. *(`settle` request field; only `save()`
      sends it, never the background size sync.)*
- [x] An **empty** open order is never settled. Zero lines means "nothing
      happened" — there is no transaction to freeze, and a fresh cart must stay
      reusable. (Changing a size back to the paid one collapses the order to zero
      lines; that must remain an open, empty order.)
- [x] Settling is idempotent and refuses an order that owes money: the
      zero-value check lives with the transition, not only in the caller.
- [x] After settling, the next edit starts a new open order via the existing
      `EnsureOpenOrder` path, and the offset against paid units stays correct.
      *(Verified: the next change reclaims the newly settled size.)*
- [x] The save response reflects the post-settle state, so the page does not keep
      showing a settled order as the editable one. *(`ordersForResponse`, plus the
      frontend `order.value = data.order || null` fix — see §8.2b.)*

### Non-Functional

- **Purity.** `ApplyPaidOffset` stays a pure, exported, unit-testable function.
  The show-path sync check and the write path must compute the same target from
  the same input, or the system oscillates. *Satisfied, and strengthened: the two
  paths now share `offsetAgainstPaid` rather than merely agreeing — §8.1a. Purity
  is what forced `sizeOrder` to become an argument.*
- **Backwards compatibility.** Changing `defaultLineID` changes the primary key
  of existing derived `order_line` rows. This is safe *only* because
  `NathejkOrderLinesChanged` has snapshot semantics — the projector deletes and
  replaces an order's lines wholesale (`tables/order/consumer.go`). *Confirmed by
  the §11 audit: `handleLinesChanged` issues `DELETE FROM order_line WHERE
  orderId=…` before re-inserting, so no cleanup step is needed.*
- **Event volume.** No new `lines.changed` events on unchanged reads. A GET that
  finds the order in sync must publish nothing.
- **Replay safety.** Recomputation from a replayed stream must converge to the
  same lines.

## 7. UX / UI Notes

Mostly invisible, deliberately — the participant should see their new size
reflected and nothing new to pay.

`aggregateOrderLines` in `vue/src/helpers/order.js` already groups by
`productSku|size` and sums `quantity` / `lineTotal`, so it will render the pair
as two rows without modification:

| | | |
|---|---|---|
| T-shirt (XXL) | −1 | −175,00 |
| T-shirt (XXXL) | 1 | 175,00 |

**Decided: render the `+`/`−` rows as they happen.** Clean and honest — the rows
state what changed and the total is correct. No exchange-labelling, no hiding of
zero-sum pairs, no special-casing of the amounts.

**Correction (2026-08-20): this does require a frontend change, contrary to an
earlier version of this section.** `aggregateOrderLines` does produce the table
above unmodified — but the block containing it was never rendered, because all
four views gated it on money owed:

```js
const showOpenOrder = computed(() => payableAmount.value > 0)
```

A free size change owes nothing, so the open-order block stayed hidden and the
page showed only the paid order's *old* size in "Tidligere betalte ordrer". The
backend was correct and the user still saw the stale size — the original bug
report, one layer up. An earlier draft of this section actually predicted the
hiding and then treated it as a convenience ("in the common case the participant
sees no new rows at all"), which was wrong: the whole point of the pair is to be
seen.

The gate is therefore split in all four views:

```js
// status badge, itemised lines, "I alt"
const showOpenOrder = computed(() => payableAmount.value > 0 || expenses.value.length > 0)
// "At betale", no-refund notice
const showPayable = computed(() => payableAmount.value > 0)
```

"An order exists" and "money is owed" are different questions, and the block
answers the first. `At betale: 0,-` and the no-refund warning stay tied to the
second, since neither says anything when nothing is due.

One consequence worth stating so it is not later mistaken for a bug: a negative
row (`−1`, `-175,-`) is now visible to the participant. It is not a refund and
must not be worded as one; it reads as "this shirt is no longer part of the
order", which is true. `oreToDkk` is `Math.round(ore / 100)`, so the sign is
preserved rather than dropped.

Affected views, all of which use the shared helper: `BadutView.vue`,
`CrewView.vue`, `KlanView.vue`, `PatruljeView.vue`.

## 8. Technical Considerations

### 8.1 shared-go — the substantive change

`tables/order/querier.go`

- Add a variant-keyed paid-quantity query alongside `PaidQuantityBySKU`
  (suggested: `PaidQuantityByVariant`, returning `map[VariantKey]int` where
  `VariantKey{SKU, Size}`). The existing `SELECT … GROUP BY l.productSku`
  becomes a group over `productSku` plus the `size` attribute; `attributes` is a
  `text` column holding JSON, so the size must be extracted in Go or with
  `JSON_EXTRACT` — prefer Go-side extraction for parity with `lineSize`, since
  the column is not a native `json` type and may hold `NULL` or `''`.
- Whether `PaidQuantityBySKU` is kept (as a derived sum over variants) or
  replaced is the implementer's call; if replaced, note it in `Queries` and
  update `sagaFakeQueries` in `saga_test.go`.

`tables/order/commander.go`

- `ApplyPaidOffset` gains the variant-aware algorithm and the credit-line
  emission. Suggested shape, per SKU:
  1. Bucket desired units and paid units by size.
  2. Cancel same-size matches (drop the desired line, decrement paid).
  3. Walk the remaining uncovered desired units in input order; while unmatched
     paid units of other sizes remain, pop one and emit the pair. **Pop in
     catalogue `Sizes` order** (`product.Product.Sizes`), not ascending slug
     order: the catalogue already defines the meaningful sequence, so the credited
     size is the one a human reading the lines would expect, whereas
     lexicographic order would credit `3xl` before `xl` purely on string
     comparison. A paid size absent from the catalogue list (a slug retired since
     it was bought) sorts last, by slug, so the ordering stays total and
     deterministic. Do not iterate a map directly.
  4. Emit any still-uncovered desired units unpaired, at full price.
  5. Discard any still-unmatched paid units (no refunds).
- The function's doc comment currently states the size-blind rationale verbatim;
  it must be rewritten, since that reasoning is what this PRD supersedes. The
  *pricing* rationale survives — the *representation* claim does not.
- `buildLines`: relax `if d.Quantity <= 0 { continue }` to `== 0`. Keep the
  `ErrMissingMemberID`, catalogue, `Active`, and `IsEligibleFor` validations
  unchanged for credit lines.
- `defaultLineID`: `derived:{sku}:{memberId}` → include the size variant.
- `checkStock`: sum only positive quantities into `newQtyBySKU`. Note
  `existingQtyBySKU` (this order's previous lines) must be treated consistently,
  or an order that already holds a credit line will compute a wrong
  `reservedElsewhere`. `ReservedQuantity` sums `l.quantity` across non-cancelled
  orders and will now see negative rows: **exclude them** (positive-only), so a
  credit cannot silently inflate available stock. The reclaimed XXL is genuinely
  returning to inventory, but stock is per-SKU today so the return cannot be
  attributed to a size — and since the pair nets to zero units for the SKU,
  positive-only and net accounting agree on the SKU total anyway. Returning a
  *specific size* to inventory is the per-size inventory feature's job (§4);
  leave it to that work rather than half-doing it here.
- `SetDerivedLines`' inline comment about billing by count needs the same
  representation correction as `ApplyPaidOffset`'s.

`tables/payment/commands.go`

- `linesReconcile` sums `Line.Amount` and compares to `Charge.Amount.Value`.
  Zero-sum pairs do not change the sum, so reconciliation holds *if* the pair is
  either both-present or both-absent in the receipt. It breaks if only one half
  is filtered out. This is the constraint the receipt builder in `tilmelding`
  must respect.
- No change expected here; verify with a test rather than assuming.

Tests to add in `tables/order/commander_test.go`, extending
`TestApplyPaidOffset`: one case per scenario in §5, plus a determinism case
asserting stable output across repeated calls with shuffled input ordering.

### 8.1a What was actually built, and where it deviates

Implemented in `016cb07`. Three additions the original §8.1 did not call for, and
one simplification of §8.2.

**Size normalisation — new.** `tables/order/variant.go` owns the size vocabulary:
`VariantKey{SKU, Size}`, the `size` attribute constant, and `lineSize`, which
trims and lowercases. Variant matching compares these strings, so a producer
writing `"XXL"` where another writes `"xxl"` would look like two sizes and the
offset would *invent a size change nobody made* — the same class of silent
wrongness this PRD exists to remove. Non-string attribute values are stringified
rather than rejected, since attributes arrive from JSON where a size could
plausibly appear as a number.

**`ApplyPaidOffset` takes the catalogue order as an argument — new.** §6 requires
the function to stay pure; §8.1 requires it to consume paid sizes in catalogue
`Sizes` order. Those are in tension, since catalogue order is data the function
does not hold. Resolved by passing it in:

```go
func ApplyPaidOffset(
    desired []DesiredLine,
    paid map[VariantKey]int,
    sizeOrder map[string][]string,
) []DesiredLine
```

**`Commands.SyncNeeded` — new, and it replaces most of §8.2.** Both callers must
pass the *same* `sizeOrder` or they compute different targets and the order
republishes its lines on every GET — §8.3's headline risk, made sharper by the
extra argument. Rather than document the hazard and test for it, the comparison
moved here: `offsetAgainstPaid` is shared by `SetDerivedLines` and `SyncNeeded`,
so the read path and the write path cannot disagree by construction. `tilmelding`
calls one function instead of reimplementing the keying:

```go
need, err := app.commands.Order.SyncNeeded(ctx, openOrder, desired)
```

`SyncNeeded` keys on `(sku, memberId, size, quantity)` per §6. `ApplyPaidOffset`
and `PaidQuantityByVariant` stay exported, but the BFF no longer needs either.

**`Queries.ShippableByVariant` — new, and it is what the §11 authority decision
means in practice.** Net quantity per variant across the owner's non-cancelled
orders: what fulfillment packs. Both properties are load-bearing and each is the
opposite of a neighbouring query — **net**, unlike `ReservedQuantity`, because a
size change is a pair and positive-only would pack both shirts; **including open
orders**, unlike `PaidQuantityByVariant`, because a free size change lives on an
open order with nothing due, so paid-only would pack the old size. Since the
correct filter differs per question, a consumer writing its own `SUM` over
`order_line` will get one of them wrong; this exists so none has to.

It does include a genuinely unpaid (N+1)-th shirt sitting on an open order. It
answers "what does the order say they get", not "what has been paid for"; a
packing list that cares should cross-reference `DueAmount`.

**Deviations from §6 worth noting:**

- Same-variant cancellation stays **whole-line**, as today. Going unit-level
  would change what a partly-covered multi-unit line costs, which §4 forbids.
  Real derived lines are one unit per member, so this is a documented caveat
  rather than a limitation anyone reaches. `reclaim` likewise takes one size per
  line: splitting a 2-unit line across two reclaimed sizes would produce credits
  whose pairing with the charge is no longer readable.
- `defaultLineID` appends the size **only for sized products**, so
  `participation.*` lineIds are byte-identical to before. §6 asked for the size
  unconditionally; narrowing it keeps the blast radius smaller.
- `PaidQuantityBySKU` is **kept**, not replaced, so a version bump does not force
  every consumer to move in the same commit. `sagaFakeQueries` gained the two new
  methods.

**Tests** (`commander_test.go`, `commander_sync_test.go`): all §5 scenarios
including change-plus-growth and change-back; catalogue-order and retired-size
reclaim; determinism across 200 runs with reshuffled map insertion order; the
credit not aliasing the charge line's attribute map; and four integration tests
through `SetDerivedLines` — the reported case producing `+1 3xl / −1 xxl` at total
0, `SyncNeeded` false immediately after a write, a credit not satisfying a
charge, and a credit not creating stock headroom.

### 8.1b shared-go — settling zero-value orders (not yet built)

Needed for the amendment in §6. `tables/order`:

- **New `Commands.Settle(ctx, orderID) (*Order, error)`.** Transitions an open,
  non-empty, zero-value order to `StatusPaid` by publishing the existing
  `NathejkOrderPaid` with `PaidAmount: 0`. No new event type and no projector
  change: `handlePaid` already updates `WHERE status='open'`, which makes a
  replayed settle a no-op.

  Guards, in the command rather than in callers:

  - not `StatusOpen` → return nil error and the order unchanged (idempotent; a
    second save must not fail).
  - zero lines → refuse (`ErrEmptyOrder`, or return unchanged — implementer's
    call, but it must not settle).
  - `TotalAmount != 0` → refuse with an error. A caller must not be able to
    freeze an order that owes money; that is the payment saga's job and only
    after money arrives.

- **Why a command and not a side effect of `SetDerivedLines`.** Auto-settling
  inside `SetDerivedLines` would fire on the BFF's self-healing *read* path,
  making a GET mutate state — and §6 requires the opposite. Keeping it a separate
  command leaves the trigger with the user's action while the invariant stays in
  the entity.

- **Why this does not corrupt the offset.** `PaidQuantityByVariant` sums raw
  `l.quantity`, negatives included, so once a settled exchange counts as paid the
  arithmetic still holds: the returned size nets to zero and the new size becomes
  the paid one.

  ```
  paid order 1:  tshirt.adult xxl  +1
  settled order: tshirt.adult 3xl  +1 ,  tshirt.adult xxl  −1
  → PaidQuantityByVariant = {(tshirt.adult,xxl): 0, (tshirt.adult,3xl): 1}
  ```

  A later change to `l` then reclaims the `3xl` — the correct size to hand back.
  This property is load-bearing; a positive-only paid count would hand the owner
  a free second shirt. Worth a test that says so.

- **Update the saga's package comment.** It states that the saga is "the only
  path by which orders reach StatusPaid". That stops being true, and the comment
  is the first thing a reader trusts. The saga's own `TotalAmount <= 0` guard
  should stay: a zero-value order must not be settled by a stray payment.

- **`ShippableByVariant` is unaffected** — it already spans every non-cancelled
  order, so an exchange counts the same before and after settling.

**Audit note for the shared-go side:** `hq` `patruljenumber/saga.go` sums
quantities over **paid** orders, and settled zero-value orders now join that set.
It filters to participation SKUs, and a participation line can only survive the
offset when it is unpaid — which makes the order cost money and therefore
unsettleable — so a settled order can only carry zero-sum t-shirt pairs and the
saga is unaffected. Confirm this rather than assume it, and note that the rule is
*value*-based: a genuinely free (0 kr) product would also be settled.

### 8.2 tilmelding — BFF

`go/cmd/api/orders.go`

- Replace `derivedLinesNeedSync` with a call to `order.Commands.SyncNeeded`
  (§8.1a). The original plan — add `quantity` to the local comparison key and
  switch it to the variant-keyed query — is superseded: keeping a second
  implementation of the comparison is what created the divergence risk. Delete
  the local helper rather than fix it, along with its doc comment claiming
  quantity drift is harmless "because the read-path helpers always emit
  quantity=1", which credits make false.

`go/cmd/api/paymentlines.go`

- `paymentLinesFromOrder` groups by `productSku|size` and will emit negative
  `UnitCount` / `Amount` rows. MobilePay's line-item API is unlikely to accept a
  negative quantity, and a receipt showing a credit the payer never receives is
  misleading regardless. Filter zero-sum pairs out of the receipt, preserving
  `linesReconcile` (§8.1) — pairs sum to 0, so dropping both halves is safe;
  dropping one is not.
- `receiptLabel` / `lineSize` need no change.

`go/cmd/api/personnel.go`, `klan.go`, `patrulje.go`, `crew.go`

- No change expected: `derivedLinesFor*` already emit the correct
  size-attributed desired set. The fix is entirely in how that set is offset.
  Verify rather than assume, particularly for `derivedLinesForKlanSeniore` with
  its synthetic `pending-N` member ids.

**API endpoints:** no new or changed endpoints. Response *shapes* are unchanged
(`order.lines` already carries `quantity` and `attributes`), so existing OpenAPI
annotations remain accurate — but the `quantity` field's contract widens from
"positive" to "non-zero, may be negative". Update the annotation descriptions
on the show/update endpoints that return an order envelope
(`showPersonnelHandler`, `updatePersonnelHandler`, and the klan/patrulje/crew
equivalents) to say so, and confirm the generated spec does not declare
`quantity` with a `minimum: 1`-style constraint.

**Data / storage:** no schema change. `order_line.quantity` and `lineTotal` are
signed `int(11)` and already accommodate negatives. `order_line`'s primary key
is `(orderId, lineId)`, so the wider `lineId` must still fit `varchar(128)` —
`derived:tshirt.adult:{uuid}:3xl` is ~55 chars, comfortable.

### 8.2b tilmelding — calling Settle (not yet built)

Depends on §8.1b. Three parts, and the first is the one that needs a decision.

**1. "Gem ændringer" and the silent auto-sync are the same request today.**
`BadutView.vue` watches the t-shirt selection and fires `syncOrder()` → the same
`PUT /api/personnel/:id` the save button calls through `save()`. `CrewView`,
`KlanView` and `PatruljeView` follow the same pattern. So a handler that settles
unconditionally would settle on **every intermediate size selection**, not on
save: a user browsing S → M → L would leave two settled zero-value exchange
orders behind and a third open, each freezing a size they only looked at. The
arithmetic stays correct — each new order reclaims the previously settled size —
but the paid history fills with exchanges that never happened, and §6 requires the
terminal transition to follow the user's explicit save.

The PUT therefore has to distinguish the two. Options:

- **A request field on the save action** — e.g. `{"person": …, "settle": true}`,
  sent by `save()` and omitted by `syncOrder()`. Smallest change; keeps one
  endpoint; the flag is honest about being a user intent rather than a state.
- **A query parameter** (`?settle=1`) — same thing, but a mutation intent in the
  URL rather than the body.
- **A separate endpoint** (`POST /api/personnel/:id/settle`) — cleanest REST
  reading and independently testable, at the cost of a second round trip from the
  save handler and a new route in all four flows.

Recommendation: **A**. It keeps the auto-sync non-terminal, which is the property
that matters, without adding a route to four views. Whichever is chosen needs an
OpenAPI annotation update, per the repo rule.

**2. Settle after the lines are written.** In each update handler, after
`SetDerivedLines` and before the payment-link block:

```go
if o.TotalAmount == 0 && len(o.Lines) > 0 && input.Settle {
    if s, err := app.commands.Order.Settle(r.Context(), o.OrderID); err == nil {
        o = s
    } else {
        log.Printf("Settle %s: %v", o.OrderID, err)
    }
}
```

A failure is logged and ignored: the order stays open and settles on the next
save, which is strictly better than failing the user's save over bookkeeping. The
existing `if o.DueAmount > 0` payment-link block is unaffected — a settled order
owes nothing by construction.

Handlers: `updatePersonnelHandler` (badut/crew), `updateKlanHandler`,
`updatePatruljeHandler`, and the crew update path.

**3. Return the post-settle envelope.** `updatePersonnelHandler` currently
replies `{"person", "order", "paymentLink"}`. Once `order` can be *settled*, that
is no longer the editable open order, and the page would show a frozen order as
its cart until the next full load. Reply with the refreshed pair instead —
`order` = the current open order (nil, or the fresh empty one) and `paidOrders` =
the history including the just-settled order.

No frontend change is needed for this: `putState()` in all four views already does
`if (data.paidOrders) paidOrders.value = [...data.paidOrders]`, it simply never
receives the key today. This also fixes a latent staleness bug — paid history
currently never refreshes after a save.

After settling, the exchange is rendered by `orderShortLines` under "Tidligere
betalte ordrer" as `1× T-shirt (S), -1× T-shirt (XXL)` at `0,-`, and
`totalPaidDkk` is unchanged because the settled order's `paidAmount` is 0.

### 8.2c What settling actually took (built)

Option **A** from §8.2b was taken: a `settle` boolean on the existing PUT, sent
by `save()` and omitted by the background `syncOrder()`. Chosen as specified;
nothing in the investigation argued against it.

`app.settleIfFree` (`orders.go`) gates on the in-memory order and calls
`order.Settle`. `app.ordersForResponse` assembles `{order, paidOrders}` from the
order just written plus the owner's history, prepending a just-settled order to
the history if the projection has not picked it up yet. All four update handlers
gained the `Settle bool` input field and both calls; the member add/update/delete
endpoints deliberately did not.

**Two defects found while verifying, both now fixed:**

1. **`Settle` raced the projector.** It re-reads the order through the read model,
   which had not yet applied the lines published microseconds earlier, so it
   returned `ErrEmptyOrder` and the order silently stayed open. `settleIfFree`
   now retries on `ErrEmptyOrder` and `ErrOrderNotFree` on the same budget as
   `setDerivedLinesAfterCreate` (10 × 50ms). Both errors are safe to retry here
   *because* the conditions were already checked against the in-memory order, so
   a disagreement can only be lag. Anything else fails fast.

2. **`updatePersonnelHandler` called `SetDerivedLines` directly**, without the
   `setDerivedLinesAfterCreate` retry the other three handlers use — so a save
   against a brand-new order returned **400** to the user (observed:
   `SetDerivedLines "record not found"`). Latent before, but settling makes a
   brand-new order the normal case on the next edit, since a settled order is no
   longer open. Now routed through the wrapper.

**Frontend:** `putState` takes `{ settle }` in all four views, and the response
handling changed from `if (data.order) order.value = data.order` to
`order.value = data.order || null`. The truthiness guard would have kept a
settled order in the cart forever, because `order: null` is now a real answer
rather than "no data". No other view change was needed.

**Response shape:** `updateKlanResponse` and `updatePatruljeResponse` gained
`paidOrders`, matching their show responses (`[]`, not `null` — the frontend does
`data.paidOrders ? [...] : []`). The two wire-shape tests were updated to mirror
the handlers. This also fixes a latent staleness bug: a save never refreshed the
paid history.

### 8.3 Dependencies, sequencing & risks

- **Cross-repo ordering.** shared-go has landed (`016cb07`); `tilmelding` bumps
  its dependency (from `v0.0.0-20260813144842-d598b7d310ea` to
  `v0.0.0-20260819231649-016cb07ccb98`) and makes the §8.2 changes in the same
  commit. `SyncNeeded` and `SetDerivedLines` now share one implementation, so the
  "they must agree" hazard is gone — and a bump without the §8.2 change cannot
  even be committed by accident, since `derivedLinesNeedSync` fails to compile
  against the new `ApplyPaidOffset` signature. The compiler now enforces what was
  previously a sequencing note.
- **Other shared-go consumers.** Audited — see §11. No consumer is broken by the
  change; `ReservedQuantity` was the only genuine instance of a sum that
  negatives would have corrupted, and it is now positive-only.
- **Risk: oscillation.** A mismatch between the sync check and the write path
  causes a `lines.changed` event on every page load. **Closed** by moving the
  comparison into shared-go (§8.1a): both paths call `offsetAgainstPaid`, and a
  test asserts `SyncNeeded` is false for lines `SetDerivedLines` just wrote.
- **Risk: non-determinism.** Map iteration in the pairing step would produce
  different-but-equivalent line sets per call, defeating the sync check with the
  same oscillation symptom. **Closed**: sizes are consumed from a sorted slice,
  and a test re-runs the offset 200 times with reshuffled map insertion order.
- **Risk: negative quantities leaking.** Sums over `order_line` elsewhere
  (reserved stock, reporting) silently change meaning. **Closed for known code**
  by the §11 audit; `ReservedQuantity` was the one instance. It remains the thing
  to check when new tooling reads `order_line` — which is what
  `ShippableByVariant` exists to prevent.

## 9. Success Metrics

- The reported record renders `−1 XXL` / `+1 XXXL` with due 0, and
  `personnel.tshirtSize` agrees with the net t-shirt lines across the owner's
  orders. Verifiable directly on
  `/badut/f789d44f-3dfb-4e8f-8143-13ae521367fa`.
- Zero owners whose roster sizes disagree with `ShippableByVariant` for their
  year. Worth writing as a standing reconciliation query: it is exactly the
  invariant that was silently violated, and the only way this class of bug
  becomes noticeable.

  Note the direction, per the authority decision in §11: the **order** is
  correct and a disagreeing **roster** is the stale side, so the query reports
  rosters to repair. An earlier draft of this metric had it the other way round.
- No increase in `lines.changed` event volume attributable to GET requests
  (guards against the oscillation risk).
- Total charged for a size-change-only edit is unchanged at 0 kr.

## 10. Rollout / Task Breakdown

Sequenced; each step is releasable except where noted.

1. shared-go: variant-keyed paid quantity + size-aware `ApplyPaidOffset` +
   credit lines, with tests. No behaviour change until a consumer bumps. **Done,
   `016cb07`.**
2. shared-go: negative-quantity support in `buildLines`, `defaultLineID`,
   `checkStock` and `ReservedQuantity` (positive-only, per §11). **Done, same
   commit.**
3. tilmelding: bump `shared-go` to `v0.0.0-20260819231649-016cb07ccb98`, switch
   to `SyncNeeded`, filter zero-sum pairs from the payment receipt, refresh
   OpenAPI annotation wording. All in one commit — a bump *without* the
   `SyncNeeded` switch does not compile, and would report drift on every read if
   it did.
4. Verify self-healing on the reported record and on a klan with a partial size
   change.

**No version tag is needed.** An earlier step 3 called for tagging shared-go;
that repo has never used semver tags — every consumer, including this one's last
bump (`1f3de5e`), pins a pseudo-version, and `@latest` already resolves to
`016cb07`. Tagging remains optional cleanup, not a gate.

No feature flag: the change is a pure recomputation on the read path, and
existing orders converge on their next GET.

Proposed tasks for `roadmap/tasks/open/` (create on approval). shared-go items
completed in `016cb07`:

- [x] Task: shared-go — audit consumers of `order_line.quantity`,
      `PaidQuantityBySKU` and the derived `lineId` format *(did not invalidate
      the design — findings in §11)*
- [x] Task: shared-go — variant-keyed paid quantity query
- [x] Task: shared-go — size-aware `ApplyPaidOffset` with zero-sum credit lines
- [x] Task: shared-go — allow negative derived line quantities (`buildLines`,
      `defaultLineID`, `checkStock`, `ReservedQuantity` positive-only)
- [x] Task: shared-go — tests for the §5 scenarios incl. determinism and
      no-op-on-resync
- [x] Task: shared-go — `SyncNeeded` and `ShippableByVariant` *(not in the
      original breakdown; see §8.1a)*
- [x] Task: tilmelding — bump shared-go to `v0.0.0-20260819231649-016cb07ccb98`
      and replace `derivedLinesNeedSync` with `order.Commands.SyncNeeded`
      *(smaller than originally scoped)*
- [x] Task: tilmelding — filter zero-sum pairs from the MobilePay receipt
- [x] Task: tilmelding — update OpenAPI annotations for the widened `quantity`
      contract *(documented on `orderLineResponse` and the klan/patrulje show
      endpoints; there is no generated spec in the repo to re-emit, and no
      `minimum` constraint existed to remove)*
- [x] Task: tilmelding — verify self-healing on affected records *(done on the
      reported record: the pair appears, repeated GETs do not republish, a
      change-back collapses to zero lines, and a 43-line order without size
      changes is unaffected)*
- [x] Task: tilmelding — show the zero-sum pair on the page: split the
      `showOpenOrder` gate from `showPayable` in all four views, so an order with
      lines but nothing due is still itemised *(the backend fix was invisible
      without this)*
- [ ] Task: tilmelding — add the standing reconciliation query from §9 *(the
      ad-hoc version — "any negative line outside a zero-sum pair, any order with
      a negative total" — returned clean; making it standing is outstanding)*

Settling zero-value orders (§6 amendment, §8.1b / §8.2b) — shared-go first:

- [x] Task: shared-go — `Commands.Settle`: open + non-empty + zero-value →
      `NathejkOrderPaid` with `PaidAmount: 0`; idempotent; refuses an order that
      owes money *(landed in `v0.0.0-20260820141337-434b63e177b7`)*
- [x] Task: shared-go — test that a settled exchange leaves
      `PaidQuantityByVariant` netting correctly *(confirmed end-to-end from the
      BFF: the next change reclaims the newly settled size)*
- [ ] Task: shared-go — correct the `Settle` doc comment claiming only zero-sum
      pairs can total zero; `participation.crew` is priced 0, so free crew signups
      settle too — which is wanted (§11), but not for the reason the comment gives
- [ ] Task: shared-go — correct the saga comment claiming it is the only path to
      `StatusPaid`
- [x] Task: tilmelding — distinguish an explicit save from the silent size
      auto-sync on the PUT, with OpenAPI annotations
- [x] Task: tilmelding — settle zero-value orders on save in the personnel, crew,
      klan and patrulje update handlers
- [x] Task: tilmelding — return `paidOrders` from the update handlers so a
      settled order stops being shown as the editable one
- [x] Task: tilmelding — route `updatePersonnelHandler` through
      `setDerivedLinesAfterCreate` *(not in the original breakdown; it returned 400
      on a brand-new order, which settling makes the normal case)*
- [x] Task: decide whether free crew signups should settle *(yes — §11; verified
      end-to-end, including that a later t-shirt charges only the shirt)*
- [x] Task: tilmelding — settle the pre-existing zero-value open order left by the
      earlier code *(happened on the first explicit save)*
- [x] Task: tilmelding — confirm the negative row renders correctly *(covered by
      `vue/src/helpers/__tests__/order.spec.js`: two separate rows, sign preserved
      through `oreToDkk`, credit never collapsed into its paired charge)*
- [x] Task: tilmelding — amend tilmelding PRD 001 to point at PRD 002 for the
      fulfillment representation rule and the size-authority decision

## 11. Decisions & Open Questions

### Decided

- **Participant-facing display** (2026-08-20) — render the `+`/`−` rows exactly
  as they happen. Clean and honest; no exchange label, no hiding. Requires no
  frontend change. See §7.
- **Per-size stock** (2026-08-20) — per-size inventory is a **planned feature**,
  simply not built yet; tilmelding PRD 001's size-stock rule was ahead of the schema, not
  wrong. Out of scope here, but this PRD keeps the size on every line, positive
  and negative, so that feature can layer a variant-level `checkStock` on top
  without redesigning the representation. See §4.
- **Credit lines and reserved stock** (2026-08-20) — `ReservedQuantity` counts
  positive quantities only. A credit must not inflate available stock, and
  returning a *specific size* to inventory belongs to the per-size inventory
  feature. Since the pair nets to zero units per SKU, positive-only and net
  accounting agree at the SKU level today. See §8.1.
- **Pairing order** (2026-08-20) — consume unmatched paid sizes in catalogue
  `Sizes` order; slugs absent from the catalogue sort last, by slug. Meaningful
  to a human and total, hence deterministic. See §8.1.
- **The order is authoritative for size** (2026-08-20, Knud) — the roster is where
  a size is *entered*; the order is where it is *true*. Anything that has to know
  which shirt to hand over reads the order, via `ShippableByVariant`. The roster's
  size field is an input to the derived lines, not a second answer to consult
  alongside them.

  The corollary, recorded in the `order` package doc: **when the roster and the
  order disagree, the order is right and the roster is stale** — so reconciliation
  repairs the roster. This inverts §9's second metric, which was written the other
  way round.

  This also answers ahead of time the question per-size inventory would have
  forced: a variant-level stock check reads the order.

- **Consumer audit outcome** (2026-08-20) — done, and it does **not** invalidate
  the design. Findings:

  - `order/consumer.go` replaces an order's lines wholesale (`DELETE` then
    re-`INSERT`), confirming the snapshot semantics §6 depends on. Widening
    `lineId` needs no cleanup step and leaves no stale rows.
  - `hq` `patruljenumber/saga.go` sums `l.Quantity`, but only over **paid**
    orders and only **participation** SKUs. Credits are t-shirt lines on open
    orders, so it is unaffected. Worth a comment there to keep that true.
  - `tilmelding` `migrate_orders.go` hand-builds the old
    `derived:{sku}:{memberId}` form. It diverges from what the commander now
    generates, but harmlessly: snapshot replace means the next
    `SetDerivedLines` overwrites it. Not a blocker.
  - `ReservedQuantity` was the one genuine instance, and is now positive-only.
  - Nothing else in `hq`, `skan` or `tilmelding` sums `order_line.quantity`.

  No fulfillment or backoffice tooling reads t-shirt sizes off orders yet, which
  is the thing §8.3 warned to look for — so `ShippableByVariant` lands before the
  first consumer rather than after.

- **Free crew signups settle too** (2026-08-21, Knud) — `participation.crew` is
  priced **0**, so a crew signup is already a non-empty zero-value order (12 in the
  dev database, none of them size changes) and is settled on the owner's first
  explicit save. Accepted deliberately: the crew member has confirmed, nothing is
  owed, and the order records the agreement. Immutability costs them nothing —
  verified end-to-end that a later t-shirt selection opens a fresh order charging
  only the shirt (175 kr), with the free seat correctly treated as already
  covered rather than re-charged.

  Consequences accepted with it: anything counting *paid orders* rather than money
  now includes crew. Revenue is unaffected, since `paidAmount` is 0. The rule is
  value-based, so this is not specific to exchanges — any future zero-priced
  product behaves the same way.

  It does falsify shared-go's `Settle` doc comment ("today only zero-sum exchange
  pairs can total zero, because any surviving participation or merchandise line
  costs money"), which should be corrected — the behaviour is wanted, the stated
  reason for it being harmless is not the true one.

- **The "paid" label for a zero-value order is moot in the UI** (2026-08-21) — a
  settled order leaves the cart slot, and the `Status: PAID` badge only renders
  inside the open-order block, so it is never shown for one. It appears in
  "Tidligere betalte ordrer" as `1× T-shirt (3XL), -1× T-shirt (L)` at `0,-`,
  which asserts nothing about payment in words. Revisit only if a backoffice view
  starts printing the raw status.

- **The silent auto-sync never settles** (2026-08-21) — intermediate size
  selections are not decisions. Only `save()` sends `settle`. Confirmed by
  observation: browsing sizes leaves one open, mutable order that is rewritten in
  place, rather than a trail of frozen exchanges.

### Open

- **Is the member roster still the source of truth for size, or does the order
  become authoritative?** Answered by the size-authority decision above, but the
  reconciliation metric in §9 still assumes the roster is canonical. Restate it in
  whichever direction is kept.
