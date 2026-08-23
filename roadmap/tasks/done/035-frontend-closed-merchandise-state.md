# 035 — frontend: closed merchandise state and read-only, order-derived t-shirt size

**Status:** open
**Priority:** high
**Created:** 2026-08-23

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

- [ ] No size picker rendered anywhere for a closed product: `Shop.vue` and all four
      member dialogs
- [ ] Closed notice shown in `Shop.vue`, with the ordered size when the user has one
- [ ] Member roster tables and dialogs show a size only when an order line backs it
- [ ] A member whose unpaid shirt was cancelled shows **no** size
- [ ] A member with a paid shirt still shows their size
- [ ] A member whose size was changed via a PRD 002 zero-sum pair shows the **net**
      size, not the credited one
- [ ] No picker flashes into view before the first `load()` resolves
- [ ] `paymentError` rendered on the crew and gøgler views
- [ ] With `closedProducts: []` the UI behaves exactly as today
- [ ] Frontend lint/build pass (`npm run build` in `vue/`); note that task 026
      tracks pre-existing lint debt — do not fold it in here

## Progress Log

- 2026-08-23 — Task created from PRD 003 §7/§8.4. Depends on 030; best done after
  031 and 032 so the cancelled-line and `paymentError` behaviour can actually be
  seen. Sole owner of the `vue/` write scope in this task set.
