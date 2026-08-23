# 035 — frontend: closed merchandise state and read-only, order-derived t-shirt size

**Status:** done
**Priority:** high
**Created:** 2026-08-23
**Picked up by:** agent session (Zed)
**Started:** 2026-08-23
**Completed:** 2026-08-23

## Description

The Vue side of PRD 003. Two jobs: stop offering a control that cannot work, and
stop displaying a t-shirt as ordered when its unpaid line was cancelled.

Depends on task 030 (`config.closedProducts` on the wire) and reads best after 031
(which is what cancels the unpaid lines).

**1. `vue/src/components/Shop.vue` — closed state**

Gains an `open` prop, passed as
`:open="!config.closedProducts.includes('tshirt.adult')"` from the two views that
use it (`CrewView.vue:328`, `BadutView.vue:363`).

While closed: keep the merchandise copy and the `tshirt_2026.jpg` image, drop the
`Dropdown` entirely (not a disabled one — there is nothing to choose), and show the
user's ordered size when they have one. Suggested Danish copy for the notice:

> Salget af årets t-shirt er lukket — t-shirtene er sendt i produktion.

**2. Member dialogs — read-only size**

`PatruljeView.vue:600` area, `KlanView.vue:600`, `CrewView.vue:465`,
`BadutView.vue:500` each bind a `Dropdown` to `member.tshirtSize` with
`:options="config.tshirtSizes"`. While closed, render
`tshirtSizeLabel(member.tshirtSize)` as static text under the same "T-shirt" label
so the layout does not jump. `config.tshirtSizes` is still returned by the API and
is still needed for label lookup and the roster table columns.

**3. Show the size from the order, not from the member**

This is the subtle one. A member whose **unpaid** selection was cancelled by task
031 still carries `tshirtSize: "l"` on their projection — the BFF deliberately does
not delete it. Rendering that as "Large" would promise a shirt that will not be
produced.

While a product is closed, the displayed size must come from **order lines**: the
open order plus `paidOrders`. Lines already carry `memberId` and
`attributes.size` on the wire (see `newOrderResponse`, asserted in
`go/cmd/api/patrulje_response_test.go:82-86`), so this is a client-side lookup, not
new API surface. Add a small helper to `vue/src/helpers/order.js` alongside the
existing `aggregateOrderLines`:

```js
// The size this member has a t-shirt line for, across the open order and the
// paid history. Empty string when they have none.
export function orderedSize(orders, memberId, sku = 'tshirt.adult')
```

Sum quantities per size rather than taking the first hit: a zero-sum size-change
pair from PRD 002 has both a negative and a positive line for the same member, and
only the size with a positive net belongs on screen.

**4. Local config defaults**

`CrewView.vue:25` and `BadutView.vue:25` seed `config` with local defaults before
the first `load()`. Seed `closedProducts: ['tshirt.adult']` so no picker flashes
into view on a slow load.

**5. `paymentError` on crew and gøgler**

Task 032 adds `paymentError` to the crew and personnel save responses. Render it
the way `PatruljeView`/`KlanView` already render theirs, so a refused payment link
explains itself instead of looking like a dead button.

**Leave in place:** the `watch` on `tshirtSize` firing `syncOrder()` in
`CrewView.vue:157` and `BadutView.vue:196`. It becomes unreachable while closed and
is the open-product path — it must survive a re-open. (Task 034 makes it harmless if
it does fire.)

## Acceptance Criteria

- [x] No size picker rendered anywhere for a closed product: `Shop.vue` and all four
      member dialogs
- [x] Closed notice shown in `Shop.vue`, with the ordered size when the user has one
- [x] Member roster tables and dialogs show a size only when an order line backs it
- [x] A member whose unpaid shirt was cancelled shows **no** size
- [x] A member with a paid shirt still shows their size
- [x] A member whose size was changed via a PRD 002 zero-sum pair shows the **net**
      size, not the credited one
- [x] No picker flashes into view before the first `load()` resolves
- [x] `paymentError` rendered on the crew and gøgler views
- [x] With `closedProducts: []` the UI behaves exactly as today
- [x] Frontend lint/build pass (`npm run build` in `vue/`); note that task 026
      tracks pre-existing lint debt — do not fold it in here

## Progress Log

- 2026-08-23 — Task created from PRD 003 §7/§8.4. Depends on 030; best done after
  031 and 032 so the cancelled-line and `paymentError` behaviour can actually be
  seen. Sole owner of the `vue/` write scope in this task set.
- 2026-08-23 — Picked up. `orderedSize(orders, memberId, sku)` added to
  `helpers/order.js`, summing quantities per size across every order passed so a
  zero-sum pair resolves to the surviving size.
- 2026-08-23 — `Shop.vue`: `open` prop defaulting to **false**, so a slow load
  cannot flash a dead picker. Closed branch drops the `Dropdown` entirely rather
  than disabling it — there is nothing left to choose.
- 2026-08-23 — Correction while wiring: the first version of the closed branch had
  an "du har ikke bestilt en års t-shirt" fallback. That is a claim this component
  cannot make on the team views, which render **one** `Shop` for a whole roster — it
  would have told a patrulje that had bought six shirts that it had bought none.
  Removed; only the banner is unconditional, and the "you ordered X" line appears
  just for single-person owners.
- 2026-08-23 — Related discovery: `PatruljeView` and `KlanView` render `<Shop />`
  with **no props at all**, so their picker was already dead (the old template
  guarded on `props.options`). They now get `:open` so the copy is at least honest
  about the sale being closed.
- 2026-08-23 — All four member dialogs gated, and the roster t-shirt column now
  reads through `memberTshirtLabel`, which sources from the orders while closed and
  from the member record while open. `paymentError` wired into `CrewView` and
  `BadutView` (the two that lacked it), matching the patrulje/klan pattern.
- 2026-08-23 — Prettier: `Shop.vue`, `order.js`, `order.spec.js`, `CrewView` and
  `BadutView` were clean before my change and are clean after. `KlanView` and
  `PatruljeView` were **already** failing prettier before I touched them (task 026
  debt), so I only reshaped my own added lines to prettier's output and left the
  pre-existing violations alone — verified by diffing a prettier-formatted copy and
  confirming every remaining delta is in code I did not write.
- 2026-08-23 — ✅ Criteria 1-2, 7-9: `vite build` and `vitest run` pass (13 tests).
  Ran both in the `tilmelding-ui-1` container — there is no host node/npm.
- 2026-08-23 — ✅ Criteria 3-6, verified against the running dev stack rather than
  by reading the code. The full stack was already up and the api container
  hot-reloads, so tasks 030-034 were live. Findings:
  - `config.closedProducts: ["tshirt.adult"]` is on the wire.
  - Patrulje `7d924d6c…`: had four unpaid t-shirt lines; after one GET the open
    order holds **participation lines only** and the shirt rows are gone from
    `order_line`. The unpaid cancellation works end to end.
  - `changedAt` is byte-identical across three consecutive GETs, so the recompute
    is stable and publishes nothing — this closes the two criteria task 031 had to
    leave unchecked for want of a stack.
  - Paid order `6ecfc73f…` is untouched by a GET of its owner (same `changedAt`),
    and its zero-sum pair survives intact.
  - The only four negative `tshirt.adult` lines in the database are all on **paid**
    orders and all dated 2026-08-21, i.e. pre-existing PRD 002 exchanges. Nothing
    in this change created a credit line.
- 2026-08-23 — The dev data turned up a case worth keeping: a gøgler who changed
  size twice, with the two exchanges on two **different** paid orders (3xl→l on one,
  l→xs on the other). Reading either order alone answers 'l' or 'xs' depending on
  which you look at; only summing across both gives 'xs'. Added as a regression
  test — it justifies the cross-order summing that would otherwise look like
  over-engineering.
- 2026-08-23 — Worth flagging for the rollout, not a defect: 85 unpaid t-shirt
  lines still sit on open orders that nobody has visited since the change. The
  exclusion is lazy — it happens when a request recomputes that order — so they
  drain as pages are loaded. Production numbers must therefore be drawn from
  **paid** orders, which are frozen, not from the open ones. Noted in PRD 003 §10.
