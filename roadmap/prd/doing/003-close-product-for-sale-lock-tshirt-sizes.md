# PRD 003 — Close a product for sale: lock t-shirt sizes and cancel unpaid t-shirts

**Status:** doing
**Author:** agent session (Zed)
**Created:** 2026-08-23
**Last updated:** 2026-08-23
**Approved:** 2026-08-23
**Shipped:**
**Target users:** participant, team leader (patrulje / klan), crew, gøgler — all signup types

---

## 1. Summary

Introduce a per-product "closed for sale" switch and apply it to `tshirt.adult`,
the 2026 year t-shirt, for every signup type. Once a product is closed nobody can
buy it and nobody can change the size of one they already bought: paid units are
frozen so they can go into production, unpaid selections are cancelled off the
open order, and no new payment request can be issued that includes the closed
product. MobilePay requests already issued are left alone and allowed to settle.

The switch is scoped to **one SKU**, not to the merchandise shop as a whole. The
year t-shirt happens to be the only thing in the shop today; a standard t-shirt,
a mug or anything else added later must remain sellable while the year tee is
closed.

## 2. Problem & Motivation

- **What problem does this solve?** The year t-shirts must be produced now, which
  needs a final, stable count per size. Today the size is a freely editable field
  on every member/person and the open order is re-derived from it on every page
  load (`derivedLinesForPatrulje`, `derivedLinesForKlan`, `derivedLinesForCrew`,
  `derivedLinesForPersonnel`). A size edited after the print file is cut produces
  a shirt nobody paid for and a paid shirt nobody wants.
- **Why now?** The ordering deadline has passed. `vue/src/components/Shop.vue`
  already tells users the year tee "skal bestilles og betales her inden August
  måned" — copy that promises a cut-off the code does not enforce.
- **Evidence.** The zero-sum credit-line mechanism from PRD 002 exists precisely
  because size changes on paid shirts are free and unbounded. That is the exact
  affordance production cannot tolerate any more.

## 3. Goals

- No new unit of a closed product can be ordered or paid for, by any signup type:
  no payment request may be issued that includes it.
- No existing unit of a closed product can change size.
- Paid units are untouched and remain exactly as recorded — they are the print run.
- Unpaid selections are cancelled: they stop being billed and stop being promised.
- A MobilePay request already issued can still be completed by its payer.
- Closing is per-product and operational: closing the year tee must not close the
  merchandise shop, and future products are unaffected by default.

## 4. Non-Goals

- Refunds, credit notes or cancellation of *paid* units. Out of scope.
- An organizer-facing admin UI for opening/closing products. The switch is
  configuration.
- Closing participation products, or changing prices, sizes or the klan seat cap.
- Print/production reporting (per-size counts for the printer). Separate concern;
  the data lives on the paid order lines.
- Cancelling outstanding MobilePay payment requests at the provider.

## 5. User Stories & Scenarios

- As a **team leader who ordered and paid for two shirts**, I want to still see
  which sizes I bought, and I must not be able to change them.
- As a **team leader who selected a size but never paid**, I want the shirt to
  disappear from what I owe, so I am not billed for something that will not be
  produced.
- As a **payer who just approved a MobilePay request**, I want that payment to
  settle my order as it stood when I approved it.
- As a **crew member who never ordered a shirt**, I want to understand that the
  online sale of the year tee has closed and merchandise is sold on site Sunday.
- As an **organizer**, I want the per-size distribution to stop moving.

Happy path (patrulje; the other three are the same shape): team leader opens
`/patrulje/:id`. The merchandise section renders closed — copy and image, no size
picker. The member table shows a shirt size only for members who actually have a
shirt on an order. Opening a member dialog shows the t-shirt size as read-only
text. Saving a member (name, phone, diet, …) succeeds, changes no order line, and
publishes no `lines.changed`.

Scenarios that define the behaviour:

| State of the unit | What happens |
|---|---|
| On a **paid** order | Untouched. Stays visible, cannot change size. |
| On the **open** order, no money in flight | Line removed; order total drops by the shirt price. |
| On the open order, **payment reserved/received but order not yet settled** | Order left entirely alone; the payment settles it as-is. |
| Link issued minutes before closing, not yet approved | Left alone and allowed to settle. Nothing new can be issued for that order while its shirt line stands (§8.2b). |
| Selected, member then deleted | Unchanged behaviour — the member's lines leave the open order. |
| No shirt at all | Nothing to do; no picker offered. |

Edge cases:

- **A crafted request sends a different `tshirtSize`.** The BFF ignores it and
  persists the stored value; the response carries the stored value back so a stale
  client self-corrects.
- **A new member is added while the product is closed.** Created with no shirt and
  no shirt line; the field is not offered.
- **A stale MobilePay link, generated before closing, is opened after the shirt
  line was dropped.** Allowed: it settles the order and the surplus is recorded on
  the payment row. Bounded to the 10 minutes a link lives, and only reachable when
  another request recomputed that order in the interim. Production numbers are
  drawn 15 minutes after closing, by which point every such link has resolved.
- **An underpaid open order** (`paidAmount > 0` but below total) keeps its shirt
  line indefinitely, by the in-flight rule. Rare; accepted.

## 6. Requirements

### Functional

- [ ] A server-side set of **closed product SKUs** governs sale of those products
      for all owner types. `tshirt.adult` is the only member of the set for 2026.
- [ ] `GET /api/patrulje/:id`, `GET /api/klan/:id`, `GET /api/crew/:id` and
      `GET /api/personnel/:id` expose the closed set in `config` as
      `closedProducts: string[]`.
- [ ] While a product is closed, every write endpoint that accepts a t-shirt size
      ignores the incoming value and persists the currently stored one:
      `POST/PUT /api/patrulje/:id/member[/:memberId]`,
      `POST/PUT /api/klan/:id/member[/:memberId]`, `PUT /api/klan/:id`,
      `PUT /api/crew/:id`, `PUT /api/personnel/:id`.
- [ ] While a product is closed, derived order lines for that SKU are **excluded**
      from the desired set, so unpaid units fall off the open order and the amount
      due drops accordingly.
- [ ] **No new payment request may include a closed product.** This is the
      invariant the whole change rests on, so it is enforced explicitly at every
      `Payment.Request` call site, not left as a consequence of line exclusion. An
      order that still carries a closed-SKU line (only possible under the in-flight
      exemption below) gets **no new payment link** until that line is gone.
- [ ] Payment requests already issued are left alone and allowed to settle. We do
      not cancel them at the provider, and we do not try to shrink an order out
      from under one.
- [ ] Exclusion is skipped for an open order with money in flight
      (`PaidAmount > 0`): such an order is left exactly as the payer saw it, and
      per the rule above it can issue no further links while that is true.
- [ ] Paid units must not produce credit/refund lines. (`ApplyPaidOffset` already
      guarantees this — "a reduction is not a size change, and this mechanism does
      not refund" — the requirement is that we do not defeat it.)
- [ ] Closing must not, by itself, cause a `lines.changed` republish on an order
      with no unpaid units of the closed SKU.
- [ ] The frontend offers no picker for a closed product, and renders any t-shirt
      size read-only.
- [ ] While a product is closed, the size shown to the user is derived from **order
      lines** (open + paid), not from the member's stored `tshirtSize`, so a
      cancelled selection stops being displayed as ordered.
- [ ] Products not in the closed set behave exactly as today. Adding a second
      merchandise product later requires no change to this mechanism.
- [ ] Re-opening a product restores current behaviour with no data migration. A
      re-opened product's members still carry their stored sizes.

### Non-Functional

- **The BFF is the enforcement point.** The frontend hiding a control is an
  affordance, not a guarantee.
- **No new 500s.** Show handlers already degrade on catalogue read failure.
- **Idempotence.** Repeated saves while closed are no-ops on the order.
- **Nothing is deleted.** Stored sizes stay on the member/person projections, so a
  re-open or an audit can recover them.
- **Danish user-facing copy**, matching the tone in `Shop.vue`.

## 7. UX / UI Notes

`vue/src/components/Shop.vue` gains a closed state, driven by a prop
(`:open="!closedProducts.includes('tshirt.adult')"`):

- Keep the merchandise copy and the shirt image.
- Replace the `Dropdown` with a short closed notice ("Salget af årets t-shirt er
  lukket — t-shirtene er sendt i produktion"), plus the user's ordered size when
  they have one on an order.
- No empty control to click when they have nothing.

Member dialogs (`PatruljeView.vue`, `KlanView.vue`, `CrewView.vue`,
`BadutView.vue`) currently bind a `Dropdown` to `member.tshirtSize` with
`:options="config.tshirtSizes"`. While closed, render the size as static text
under the same "T-shirt" label so the layout does not jump. `config.tshirtSizes`
is still returned and still used for label lookup.

**Size display comes from the order while closed.** A member whose unpaid
selection was cancelled must not keep showing "Large" as if it were ordered.
Order lines already carry `memberId` and `attributes.size` on the wire (see
`newOrderResponse`, asserted in `patrulje_response_test.go`), and
`vue/src/helpers/order.js` already aggregates lines, so this is a client-side
lookup across the open order plus `paidOrders` rather than new API surface.

The `watch` on `tshirtSize` firing the background `syncOrder()` in `CrewView.vue`
and `BadutView.vue` becomes unreachable while closed. Leave it: it is the
open-product path and must survive a re-open.

## 8. Technical Considerations

### 8.1 BFF (Go) — the enforcement point

- `config` in `go/cmd/api/main.go` gains `shop.closedSKUs`, read via
  `getEnvAsSlice("CLOSED_PRODUCT_SKUS", []string{"tshirt.adult"}, ",")` into a
  set, plus `func (app *application) skuClosed(sku string) bool`. Default closed
  for the year tee so a deployment that forgets the variable fails safe.
- `TeamConfig` (`go/cmd/api/patrulje.go`) gains `ClosedProducts []string`;
  `buildTeamConfig` fills it. All four show handlers funnel through
  `buildTeamConfig`, so that is one edit. `newTeamConfigResponse` gains the
  `closedProducts` key — the two wire-shape tests (`klan_response_test.go`,
  `patrulje_response_test.go`) compare literal JSON and must be updated in the
  same change.
- **Line exclusion at the chokepoint, not the call sites.** There are eleven places
  a desired set is built (four show handlers, `updateKlanHandler`,
  `updatePatruljeHandler`, `rederiveKlanOrder`, `rederivePatruljeOrder`, the crew
  and personnel save paths, and `requestSeatHandler`), but only two places one is
  *consumed*: `app.syncNeeded` and `app.setDerivedLinesAfterCreate`, both in
  `orders.go`. Applying
  `func (app *application) sellable(o *order.Order, lines []order.DesiredLine) []order.DesiredLine`
  inside those two makes it impossible for the sync check and the write to see
  different sets — the drift that would republish the order's lines on every page
  load. It also means no producer function changes at all.
  - `setDerivedLinesAfterCreate` currently takes an `orderID`; it needs the order
    itself for the `PaidAmount` guard, so its signature becomes
    `(ctx, o *order.Order, desired)`. Every call site already has the order in
    scope, so this is mechanical.
  - `requestSeatHandler` (`klan.go:348`) calls `SetDerivedLines` directly with
    participation-only placeholder lines. Left as-is — it can never carry a
    merchandise SKU — with a comment saying why it bypasses the filter.
- **In-flight guard:** skip the filter when the open order has `PaidAmount > 0`.
  `paidAmount` is computed on the order projection as the sum of payments for that
  order in `('reserved','received')` (see `orderColumns` in shared-go's
  `order/querier.go`), so this needs no payment query, no new shared-go filter, and
  no new index. `requested` deliberately does not count — see §8.2b for why that is
  safe and why it has to be handled at deploy time rather than in code.
- **Size lock** in each write handler, substituting the stored value when closed.
  Per-handler because each reads the size from a different place:
  - `patrulje.go` — `TShirtSize` on the member body; stored value from `spejder`.
  - `klan.go` — `TShirtSize` on the senior body and on the roster in
    `updateKlanHandler`; stored value from the klan senior projection.
  - `crew.go` — `tshirtSize` folded into `additionals` under `tshirtSizeKey`. The
    delete branch (`delete(additionals, tshirtSizeKey)`) must not run while
    closed, or an empty field silently erases the record of a paid shirt.
  - `personnel.go` — `TshirtSize` on the person.
  A shared helper (`lockedSize(stored, requested string, sku string) string`)
  keeps the rule in one place and gives the tests one seam.
- **Payment requests must be gated explicitly.** With the shirt off the order,
  `Charge.Amount` (`o.DueAmount`) and `Charge.Lines` (`paymentLinesFromOrder(o)`)
  cannot contain it — but an order exempted by the in-flight guard *does* still
  carry the line, and a save on it would otherwise mint a fresh link including the
  shirt. So a helper
  `func (app *application) chargeable(o *order.Order) (bool, string)` reports
  whether an order may be charged, refusing any order holding a closed-SKU line,
  and gates all five `Payment.Request` call sites (`patrulje.go:513`,
  `klan.go:358` and `:477`, `crew.go:179`, `personnel.go:140`). Refusal is not an
  error: `paymentLink` comes back empty, with the existing `paymentError` field
  carrying a Danish explanation on the patrulje and klan responses. Such an order
  resumes issuing links as soon as its in-flight payment settles or expires and the
  next recompute strips the line.

### 8.2 Why paid units are safe

`ApplyPaidOffset` matches desired units against paid units per `(SKU, size)`.
Feeding it a desired set with no `tshirt.adult` lines leaves the paid shirt units
unmatched, and unmatched *paid* units produce nothing — no credit line, no
refund (documented on the function, and the netting in `netOutCredits` is
therefore never asked to handle an unpaired credit). Paid orders are immutable
anyway: `SetDerivedLines` rejects them with `ErrNotOpen`, and `loadOrders` only
re-derives the open one.

### 8.2b Letting existing payment requests settle

The rule is asymmetric on purpose: **no new charge may include a closed product,
but every charge already issued is allowed to settle.** Nothing cancels an
outstanding request, and nothing tries to shrink an order beneath one.

What happens to a link issued minutes before closing:

- **Payer approves before anything recomputes that order.** The order still holds
  its shirt line, the payment covers the total, the saga marks it paid, and the
  shirt is in the paid record. Correct outcome, no special handling.
- **Payer approves after a recompute dropped the line.** The payment exceeds the
  reduced total; the saga settles the order and the surplus is recorded on the
  payment row. Bounded to the 10 minutes a MobilePay link lives, and only when a
  concurrent request recomputed that specific order in the interim.
- **Payer never approves.** The link expires at the provider after 10 minutes. Our
  projection keeps the row at `requested` forever, because nothing writes
  `timedout` (see below), so the row is not evidence of anything live.

Why `requested` cannot be part of the runtime guard:

- **It is not intent.** All four save handlers call `Payment.Request` on every PUT
  where `due > 0`. Nearly every open order carries a `requested` payment, so
  treating it as in-flight would exempt nearly every order and cancel nothing.
- **It is indistinguishable from a dead link.** `types` defines
  `PaymentStatusTimedout` and `PaymentStatusRejected`, but nothing writes them: the
  payment projector (`shared-go/tables/payment/consumer.go`) subscribes only to
  `.requested`, `.reserved` and `.received`. An abandoned request stays `requested`
  in our projection forever.

So `reserved`/`received` — money actually committed, visible as `PaidAmount > 0` —
is the guard, and the 10-minute link lifetime is what bounds everything else. No
deploy-time blocker follows from this: closing is safe at any moment, because the
only thing a pre-existing link can do is complete a purchase the payer had already
started.

**Operational consequence:** production numbers must be drawn at least 15 minutes
after closing, so every link that could still add a shirt has either settled or
expired. That is a step in §10, not a constraint on the code.

### 8.3 API endpoints

No new routes. Changed response shape on the four show endpoints
(`config.closedProducts`) and changed request *semantics* on the seven write
endpoints in §6.

**OpenAPI annotations:** every touched handler already carries them and each must
be updated — the show handlers' `@Description` to mention `closedProducts`, the
write handlers' to state that `tshirtSize` is ignored for a closed product and
that unpaid units of a closed product are removed from the open order. Repo
requirement, not optional (`.rules`).

### 8.4 Frontend (Vue 3)

- `components/Shop.vue` — `open` prop, closed rendering, ordered-size display.
- `views/PatruljeView.vue`, `views/KlanView.vue`, `views/CrewView.vue`,
  `views/BadutView.vue` — pass the closed state to `Shop`, gate the dialog
  `Dropdown`, and source the displayed size from order lines while closed.
- `helpers/order.js` — small addition: "size ordered by member X", reading the
  open order plus `paidOrders`.
- The `config` refs in `CrewView.vue` / `BadutView.vue` are seeded with local
  defaults; seed `closedProducts: ['tshirt.adult']` so no picker flashes into
  view before the first `load()` resolves.

### 8.5 Rejected (for now): `active = 0` on the catalogue row

Marking the product out of stock is the intuitive framing, and the catalogue has
the column — `product.active`, settable from `Seeds2026()` via `Seed.Active`. It
does not work as the mechanism today:

`order.buildLines` rejects *any* desired line whose product is inactive
(`ErrProductInactive`), and both show and save handlers re-derive the full desired
set on every request. So the moment `tshirt.adult` goes inactive, every owner who
has a shirt size stored — including those whose shirts are **paid** — presents a
desired set the commander refuses: saves fail outright and show handlers log the
failure and render a stale order. `stock = 0` fails the same way via `checkStock`.

Note that this PRD's line exclusion removes the *unpaid* shirt lines, which is
most of what the inactive flag was wanted for, but not all: `participation.*`
lines for the same members still flow through `buildLines` on every request, and a
paid-shirt owner's desired set is only free of shirt lines *because* we filter it.
Flipping `active` would still break the moment anything re-derived a shirt line.

The clean version is a shared-go change: `buildLines` should tolerate an inactive
product for a line already on the order or covered by the paid offset, and reject
it only when genuinely new — which is what the comment on `product.GetBySKU`
already claims ("retiring a product doesn't break already-issued orders") and the
code does not implement. Once that lands, `closedProducts` can be derived from
`Product.Active` and the env var retired. Sequenced as a follow-up in §10, not a
blocker: production cannot wait on a two-repo change.

### 8.6 Dependencies & risks

- **No schema change, no migration, no shared-go bump.** Deliberate: deployable
  today.
- **Risk — stale MobilePay links** settling against a shrunk order. Accepted and
  bounded: 10-minute link lifetime, and production numbers are drawn 15 minutes
  after closing (§10).
- **Risk — an order stuck without a payment link.** An underpaid order keeps its
  shirt line and therefore cannot be issued a new link. Self-clears when the
  in-flight payment settles or expires; if one is genuinely wedged, the operator
  sees it as an order with a `tshirt.adult` line and `0 < paidAmount < total`.
- **Risk — wire-shape tests** fail by design on a new `config` key; update with
  the change.
- **Risk — client/server drift.** A cached frontend could still render a picker;
  harmless, the BFF ignores the value.
- **Risk — the in-flight guard is coarse.** An underpaid open order keeps its
  shirt line indefinitely. Deliberate (§8.2b): the alternative is a per-order
  payment lookup on every page load, needing a shared-go filter, to protect a
  window that expires by itself. The `chargeable` gate keeps such an order from
  turning that line into a new sale.
- **Interaction with PRD 002.** The zero-sum credit path stays in the code but
  becomes unreachable for `tshirt.adult`: no size can change, and no shirt line is
  derived for an unpaid unit. `settleIfFree` is unaffected.

## 9. Success Metrics

- Zero `tshirt.adult` lines created after the switch flips (order projection, by
  `createdAt`).
- Zero **negative** `tshirt.adult` lines created after the switch flips — the
  credit half of a size change. The direct measure of "sizes are frozen".
- Per-size paid shirt counts identical on two reads a week apart — measured from 15
  minutes after closing onwards.
- Count of open orders still carrying a `tshirt.adult` line trends to zero (only
  the in-flight/underpaid exceptions remain).
- Every payment that settles after closing matches the `tshirt.adult` lines on its
  order, i.e. nobody paid for a shirt the order does not promise.
- No increase in 5xx on the four show and seven write endpoints.

## 10. Rollout / Task Breakdown

Phase 1 (this PRD, no shared-go release needed) — created as tasks 030-035:

- [ ] Task 030: Add `CLOSED_PRODUCT_SKUS` config and expose `config.closedProducts` on the four show endpoints
- [ ] Task 031: Exclude closed-SKU derived lines from the desired set, with the `PaidAmount > 0` in-flight guard
- [ ] Task 032: Refuse to issue a payment request for an order holding a closed-SKU line (`chargeable` gate at the five call sites)
- [ ] Task 033: Lock t-shirt size server-side in the patrulje and klan write handlers
- [ ] Task 034: Lock t-shirt size server-side in the crew and personnel (gøgler) write handlers
- [ ] Task 035: Closed state for `Shop.vue`; read-only, order-derived size in the four member dialogs

OpenAPI annotation updates and the two wire-shape test fixes are folded into the
task that changes the contract in question, rather than trailing as a separate
task, so every task leaves the repo green on its own.

Sequencing: 030 first (everything reads `skuClosed`). Then 031 → 032, in that order
— 032 depends on the exemption 031 introduces. 033 and 034 have disjoint write
scopes and can run in parallel with the 031/032 pair once 030 has landed; whichever
of the two goes first introduces the shared `lockedSize` helper. 035 owns the whole
`vue/` scope and is best done last, so the cancelled-line and `paymentError`
behaviour can be seen rather than imagined.

Phase 2 (follow-up, catalogue as source of truth):

- [ ] Task: shared-go — allow an inactive product on a re-derived line that already exists or is paid
- [ ] Task: Mark `tshirt.adult` inactive in `Seeds2026`, derive `closedProducts` from `Product.Active`, retire `CLOSED_PRODUCT_SKUS`

### Closing procedure

No pre-close check and no deploy window: closing is safe at any moment. Payment
links are only ever issued on a save, expire after 10 minutes, and the code cannot
mint a new one containing a closed product — so the worst a pre-existing link can
do is complete a purchase its payer had already started, which is the intended
outcome.

1. **Deploy** with the default closed set.
2. **Verify** on one patrulje, one klan, one crew and one gøgler record: the picker
   is gone; a paid shirt still shows and its order line is unchanged; an unpaid
   shirt has left the open order and the due amount dropped by 175 kr; a save
   publishes no `lines.changed` on an order with no unpaid shirt; a fresh payment
   request contains no t-shirt row.
3. **Wait 15 minutes before drawing the production numbers.** Every link issued
   before the deploy has settled or expired by then, so the per-size counts stop
   moving. This is the one timing rule that matters.

Diagnostic — which payments were still live at closing time, and what became of
them:

```sql
SELECT p.reference, p.amount, p.status, p.createdAt, p.changedAt,
       p.orderForeignKey, o.ownerType, o.ownerId
FROM payment p
LEFT JOIN orders o ON o.orderId = p.orderForeignKey
WHERE p.year = '2026'
  AND p.createdAt > DATE_SUB(NOW(), INTERVAL 30 MINUTE)
ORDER BY p.createdAt;
```

Rows left at `requested` are abandoned links (nothing writes `timedout`); rows at
`reserved`/`received` are the purchases that completed on the way through. Cross-
check the latter against `tshirt.adult` lines on their orders to confirm each
settled payment matches what the order says will be shipped.

## 11. Decisions & Open Questions

### Decided

- **Stale MobilePay links (2026-08-23).** Decided: let them settle. The invariant
  that matters is that no *new* charge can include a closed product, enforced at
  the `Payment.Request` call sites (§8.1). Existing requests are neither cancelled
  nor undercut; they expire after 10 minutes on their own. Consequently there is no
  pre-close check and no deploy window — only the operational rule that production
  numbers are drawn 15 minutes after closing.

### Open

- **Do cancelled unpaid selections need to be communicated?** Right now the shirt
  simply stops appearing. An email to affected owners ("din t-shirt-bestilling
  blev ikke betalt inden fristen og er annulleret") may be warranted — it is a
  comms decision, and there is an existing mail pipeline if the answer is yes.
- **Should the stored `tshirtSize` be cleared for cancelled unpaid units?** This
  PRD says no (nothing is deleted; display comes from the order instead). Clearing
  would simplify the frontend but destroys the record of what someone wanted.
- **Closed-state copy** for `Shop.vue` — confirm wording with whoever owns
  merchandise comms.
