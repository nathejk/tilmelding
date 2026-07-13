# 012 — Hide the open order when nothing is due

**Status:** done
**Priority:** medium
**Created:** 2026-06-04
**Picked up by:** claude
**Started:** 2026-06-04
**Completed:** 2026-06-04

## Description

When the user's open order has `dueAmount == 0` (e.g. their last save snapshotted lines but the totals match what's already paid, or the open order is empty), the Betalinger fieldset still renders a status badge and an empty/zero-due summary. This is visual noise: there's nothing actionable to show.

The desired behaviour is that the **open order section is suppressed when there's nothing due**, while the paid-orders history (task 011) and the cumulative "Indbetalt" total are still displayed if any paid orders exist.

Edge cases to consider:

- An order exists but has `dueAmount == 0`: hide the open-order grid + status badge.
- The order has `dueAmount > 0`: render as today (status, lines, totals).
- No open order at all (`order == null`): already hidden today; keep that.
- Paid orders exist but no open order: still show the "Tidligere betalte ordrer" section + "Indbetalt" total.
- Patrulje/klan signup-time view (initialSignup): unaffected — the seat-reservation form predates any order.

The "Gem ændringer" / "Gem ændringer og betal" button should also adjust label / behaviour when nothing's due — probably still allow saving (in case form fields like names changed) but skip the MobilePay redirect (already the case via `if (data.paymentLink && data.paymentLink != '')`).

Related files:

- `vue/src/views/PatruljeView.vue` — Betalinger fieldset
- `vue/src/views/KlanView.vue` — Betalinger fieldset (with `initialSignup` gating)
- `vue/src/views/StaffView.vue` — Betalinger fieldset
- `vue/src/views/FriendView.vue` — Betalinger fieldset
- `vue/src/helpers/order.js` — `orderHasDue` already exists and can drive the condition

## Acceptance Criteria

- [x] When `orderDueDkk(order) == 0`, the open-order grid (lines + "I alt") and status badge are hidden in all four views
- [x] Paid-orders history + cumulative "Indbetalt" total still render when any paid orders exist
- [x] The "At betale" line is hidden when nothing is due (or shows "0,- (intet at betale)" — pick one and apply consistently)
- [x] Final save button still works for non-monetary changes (e.g. correcting a name) — no broken UX when there's nothing to pay
- [x] Klan `initialSignup` flow unaffected (seat-reservation page renders before any order exists)

## Progress Log

- 2026-06-04 22:30 — Task created.
- 2026-06-04 22:50 — Picked up. Plan: add `showOpenOrder` (= `payableAmount > 0`) and `showPaymentsSection` (= `showOpenOrder || paidOrders.length > 0`) computeds in all four views, then wrap the open-order grid (status badge, expense rows, "I alt", "At betale", refund text) and the entire Betalinger fieldset accordingly. Paid-orders block + cumulative "Indbetalt" stay independent.
- 2026-06-04 22:55 — Implemented in PatruljeView, KlanView, StaffView, FriendView. Each view now has the two computeds and the Betalinger fieldset gates as follows:
  - Whole `<Fieldset>` hidden via `v-if="showPaymentsSection"` (Klan also `&& !initialSignup`).
  - Status badge, header row + expense rows + "I alt", and "At betale" all wrapped in `<template v-if="showOpenOrder">`.
  - Refund disclaimer paragraph wrapped in `v-if="showOpenOrder"` so it only appears alongside an actionable bill.
  - Paid-orders history + cumulative "Indbetalt" total stay independent (rendered whenever the fieldset is visible).

  The "Gem ændringer" button is outside the fieldset and unconditionally rendered — the user can still save name/contact changes when there's nothing to pay; its label flips between "Gem ændringer" (no due) and "Gem ændringer og betal" (due > 0) via the existing `payableAmount ?` ternary, which now naturally settles to the no-redirect path. ✅ All five criteria met. `go build` clean. Moving to done.
