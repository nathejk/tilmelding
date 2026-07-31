# PRD 001 — Seat-based team billing & stable member identity

**Status:** draft
**Author:** agent session
**Created:** 2026-07-31
**Last updated:** 2026-07-31
**Target users:** team leader (patrulje/klan contact person), organizer

---

## 1. Summary

A team pays for a number of **participation seats**, not for specific people.
Which member occupies a seat can change over time: if a member is deleted their
seat becomes vacant, and a later-added member reuses that already-paid seat with
no new charge. **T-shirts behave the same way** — a paid t-shirt is a reusable
unit that can be reassigned to another member, and its size can be changed free
of charge as long as the chosen size is in stock. Today the system bills both
participation and t-shirts per `memberId` and re-mints member identities on
almost every save, which both **double-charges** teams and produces **duplicate
member rows**. This PRD does two related things:

1. **Bill by paid-unit count, not identity** — participation seats and t-shirts
   are counted; the open order charges only for units beyond those already paid,
   so swapping members (or changing a t-shirt size) never re-charges.
2. **Make member identity stable** — members are added/updated/removed through
   explicit member endpoints, each mapping to one stream event, so identities are
   no longer re-created as a side effect of a save.

## 2. Problem & Motivation

- **Participation is billed against the wrong thing.** The order derives one
  `participation.patrulje` line per active member keyed by `memberId`, and the
  "already paid" guard is keyed on `(productSku, memberId)` (`PaidLineKeys` in
  `go/nathejk/table/order/querier.go`). That model assumes a paid seat belongs
  to one specific person forever. In reality a team pays for *N seats*; the
  people in them can change. When a member is swapped, the new member's
  `memberId` isn't in the paid set, so the open order bills their participation
  again even though a paid seat is sitting vacant.

- **Member identities churn.** The roster is saved through a single
  `PUT /api/patrulje/{id}` (and klan equivalent) that republishes a
  `spejder.updated` per member. New members arrive without a `memberId`, and
  `patrulje.Commands.Update` mints a fresh UUID for any empty ID
  (`go/nathejk/table/patrulje/commands.go:105`). The PUT response omits members
  (`go/cmd/api/patrulje.go:223`), so the client never learns the ID and re-sends
  without one; the projector inserts by `memberId` PK with `INSERT IGNORE`
  (`go/nathejk/table/spejder/consumer.go:46`), so each save creates a new row and
  orphans the old one. This both corrupts the roster and (combined with the
  memberId-keyed billing above) guarantees the double-charge.

- **Why now?** Live 2026 signup is producing wrong data and wrong bills:
  - Team `551c8afb-14cf-46c7-95f5-e4849a5c4bf9`: 4 real people materialised as
    **20 `spejder` rows** — one duplicate per save.
  - Team `fcc870d1-2c09-48a4-ba41-8a06b162b312`: a paid order (4 seats, 1000 kr)
    references member IDs no longer present; the open order re-bills those 4
    seats (1000 kr) on top of 700 kr of genuinely new t-shirts. Under a seat
    model, the 4 seats are paid and only the t-shirts are due.

- **Evidence.** Direct inspection of the live dev database (both team IDs) and a
  code walkthrough of the patrulje/klan update path, order commander/querier, and
  spejder projector.

## 3. Goals

- Participation is billed by **seat count**: the open order charges only for
  active members beyond the number of seats already paid. Deleting a member
  vacates a paid seat; adding a member fills a vacant paid seat before any new
  seat is charged.
- T-shirts are billed the same way by **paid-t-shirt count**: a paid t-shirt is
  reusable across members, and its size may be changed free of charge as long as
  the chosen size is in stock. Only t-shirts beyond the paid count are charged.
- A member has exactly one `memberId` for its lifetime, issued by the API on an
  explicit add (the frontend never invents IDs) and stable through edits.
- The member lifecycle runs through explicit endpoints — add → create, edit →
  update, delete → delete — one stream event per action. The roster/order-sync
  path can no longer create or delete member identities.
- Member counts are enforced server-side: a team cannot exceed its maximum, and
  cannot pay while it has fewer members than its minimum.

## 4. Non-Goals

- Redesigning the signup/registration flow or the payment provider integration
  beyond the seat-billing, member-lifecycle, and cap changes described here.
- **Cleaning up existing duplicate rows and reconciling already-corrupted open
  orders — the operator handles this manually.** This PRD stops new corruption;
  it does not repair historical data.
- Changing crew / badut / personnel flows structurally (single-person, stable
  `userId`); consistency review only.
- Sync-payload optimization (sending only changed data).

## 5. User Stories & Scenarios

- As a **team leader**, I pay for a set of seats and can freely swap who sits in
  them without being charged again, and my roster never duplicates people.

**Seat reuse (the key scenario):**
1. A team pays for 4 seats (4 members). The order is paid; 4 seats are paid.
2. Later they delete one member — a paid seat is now vacant. Nothing is refunded
   or re-charged.
3. Later still they add a new member. The new member fills the vacant paid seat.
   Participation due stays 0. If the deleted member had a paid t-shirt, that
   t-shirt is reusable too: the new member may take it (any size in stock) with
   no new charge.
4. If they add a 5th member (beyond 4 paid seats), only that one extra seat is
   charged. Likewise a 5th t-shirt beyond the paid t-shirt count is charged,
   while changing sizes on the existing paid t-shirts stays free.

**Add / Edit / Delete (identity):**
- Add → `POST /api/patrulje/{teamId}/member`: API issues the `memberId`,
  publishes one create event, enforces the maximum, returns the created member.
- Edit → `PUT /api/patrulje/{teamId}/member/{memberId}`: one update event, never
  a new identity.
- Delete → `DELETE /api/patrulje/{teamId}/member/{memberId}`: one delete event;
  the row is removed and the seat is vacated.

**Cap enforcement:**
- Adding beyond the maximum is rejected by the add endpoint.
- Paying while below the minimum active member count is rejected on every
  payment-initiating path — no payment started, no payment link issued.

**Edge cases:**
- Retrying an add must not create two members (keyed on the server-issued
  `memberId`, which the client adopts).
- The roster/team PUT can no longer create or delete members.
- Deleting a member who was paid for: the paid order is immutable; the seat
  simply becomes vacant and reusable.

## 6. Requirements

### Functional

- [ ] Participation is billed by seat count. The open order's participation
      charge = max(0, active member count − seats already paid). Vacated paid
      seats are reused by later-added members before any new seat is charged.
- [ ] T-shirts are billed by paid-t-shirt count: charge = max(0, active t-shirt
      count − t-shirts already paid). A paid t-shirt is reassignable to any
      member and its size is changeable free of charge, subject to available
      stock for the chosen size.
- [ ] Adding a member goes through a dedicated endpoint that issues the
      `memberId`, persists it (one create event), enforces the maximum, and
      returns the created member including its `memberId`.
- [ ] Editing a member goes through a per-member endpoint that emits an update
      event only and never creates a new identity.
- [ ] Deleting a member goes through a per-member endpoint that emits a delete
      event and vacates the seat.
- [ ] The roster/team save (`PUT /api/patrulje/{id}`, `PUT /api/klan/{id}`) no
      longer creates or deletes members; ad-hoc UUID minting on that path is
      removed. It handles team/contact fields and order recomputation only.
- [ ] The frontend keys member operations on the server-issued `memberId`
      (`createId()` removed).
- [ ] Server-side caps: adds beyond the maximum are rejected; payment is blocked
      for any team below the minimum active member count (no payment initiated,
      no payment link issued).

### Non-Functional

- **Data integrity:** a save/sync of an unchanged roster produces no new member
  rows and no create/delete events.
- **Correct billing:** swapping members (delete + add within paid seats) never
  changes the participation amount due.
- **Backwards compatibility:** existing paid orders remain immutable; seats-paid
  is derived from them, not by rewriting them.

## 7. UX / UI Notes

- `vue/src/views/PatruljeView.vue` and `vue/src/views/KlanView.vue`:
  - Add/edit/delete members via the new member endpoints; store the returned
    `memberId`; key state on `memberId`; remove `createId()`.
  - The bill shows participation for the current members, but "at betale"
    reflects only unpaid seats — swapping members shows 0 additional
    participation due.
  - Surface "team is full" and "too few members to pay" errors from the API.
- No major layout change intended.

## 8. Technical Considerations

- **Order model (BFF, the core change):**
  - Replace per-`memberId` participation dedup with seat accounting. "Seats
    paid" = total participation quantity across the team's **paid** orders for
    the year. The open order's participation due = max(0, activeMemberCount −
    seatsPaid). This supersedes the `(productSku, memberId)` logic in
    `PaidLineKeys` / `SetDerivedLines` for participation.
  - Decide the line representation: either a single participation line with
    quantity = billable seats, or keep per-member display lines but compute the
    paid offset by count. Either way the *amount due* is seat-based, not
    identity-based.
  - **Merchandise (t-shirts)** is accounted the same way, by count:
    t-shirts-paid = total `tshirt.adult` quantity across the team's paid orders;
    t-shirt due = max(0, activeTshirtCount − tshirtsPaid). Size is a fulfillment
    attribute that may change freely without re-charging. Stock is enforced on
    the chosen size via the existing `checkStock` merchandise rule, so a size
    swap is allowed only if that size has stock.

- **Member lifecycle (BFF):**
  - New endpoints for patrulje (spejder) and klan (senior):
    `POST /api/patrulje/{teamId}/member`,
    `PUT /api/patrulje/{teamId}/member/{memberId}`,
    `DELETE /api/patrulje/{teamId}/member/{memberId}`, and klan equivalents.
  - Split `patrulje.Commands.Update` (and klan/senior) so member create (issues
    the ID) is distinct from update/delete; remove ad-hoc UUID minting from the
    roster path.
  - Adding/removing a member re-derives the open order using seat accounting;
    tolerate projection lag as `setDerivedLinesAfterCreate` already does.
  - Confirm the spejder and senior projectors: create→insert, update→update,
    delete→delete; no create path reachable from a routine save.

- **Member caps:** maximum on the add endpoint; minimum on every
  payment-initiating path (the payment-link creation in
  `updatePatruljeHandler`/`updateKlanHandler`, and `PUT /api/pay/{teamId}`).
  Bounds from the same source as `buildTeamConfig` (patrulje 3–7; klan per its
  config — confirm before wiring).

- **API endpoints (all new/changed endpoints need OpenAPI annotations):**
  - New: `POST/PUT/DELETE /api/patrulje/{teamId}/member[/{memberId}]` and the
    klan equivalents.
  - Changed: `PUT /api/patrulje/{id}`, `PUT /api/klan/{id}` — member lifecycle
    removed; participation amount now seat-based.

- **Relationship to the earlier "mint an ID without the stream" idea:** with
  explicit endpoints the add action persists immediately, so the ID is issued and
  persisted together by the add endpoint — no separate ID-only endpoint is
  needed, while still honoring "the frontend does not invent IDs" and "the stream
  is written when the member is persisted".

- **Data / storage:** no migration in this PRD. Cleanup of existing duplicate
  rows and reconciliation of already-corrupted open orders are done manually by
  the operator (see Non-Goals).

- **Dependencies & risks:**
  - Klan uses the `senior` table. Its `reservedMemberCount` is legacy naming for
    the same concept — buying X seats — so klan seat accounting is the same as
    patrulje (seats-paid from paid participation). Verify the senior projector
    shares the spejder projector's `INSERT IGNORE` + `memberId` PK behavior.
  - Seat accounting must count only **paid** orders as seats-paid; cancelled and
    open orders do not contribute paid seats.
  - Splitting the roster PUT is an API-contract change; migrate the frontend in
    lockstep.
  - Eventual consistency: an add returns before the projection reflects the new
    member; seat re-derivation must tolerate the lag.

## 9. Success Metrics

- Deleting a member and adding another within the paid seat count results in **0
  additional participation due** (verified by test and on team `fcc870d1…`, which
  should owe only the 700 kr t-shirts).
- Changing a t-shirt size, or reassigning a paid t-shirt to a new member, adds
  **0 additional t-shirt due** as long as the chosen size is in stock.
- No save or order-sync of an unchanged roster creates member rows or emits
  create/delete events.
- Adding beyond the maximum, and paying below the minimum, are both rejected
  server-side.
- Each member action (add/edit/delete) emits exactly one stream event.

## 10. Rollout / Task Breakdown

Sequencing: land the seat-billing change and the member-endpoint/roster-PUT
change together with the frontend migration (lockstep contract change). Operator
performs manual data cleanup afterwards.

Proposed tasks for `roadmap/tasks/open/`:

- [ ] Task: BFF — count-based billing for participation and t-shirts:
      seats-paid / t-shirts-paid from paid orders, open-order due =
      max(0, activeCount − paidCount) for each; t-shirt size changeable free of
      charge subject to stock. Replace memberId dedup.
- [ ] Task: BFF — patrulje member endpoints (`POST`/`PUT`/`DELETE`) issuing IDs,
      one event each, max enforced; OpenAPI.
- [ ] Task: BFF — klan/senior member endpoints (same shape) + senior projector
      parity; confirm klan seat counting.
- [ ] Task: BFF — remove member create/delete + UUID minting from roster `PUT`;
      keep team/contact + seat-based order re-derivation.
- [ ] Task: BFF — caps: max on add, block payment below minimum (patrulje + klan).
- [ ] Task: Frontend — migrate `PatruljeView.vue` to member endpoints, key on
      `memberId`, show seat-based "at betale", render cap errors.
- [ ] Task: Frontend — migrate `KlanView.vue` likewise.
- [ ] Task: Tests — seat reuse (delete+add = 0 due), one-event-per-action,
      save/sync idempotency, cap enforcement.

## 11. Open Questions

- **Line representation.** Single participation/t-shirt lines with a quantity, or
  keep per-member display lines with a count-based paid offset? (Implementation
  detail; flagging for the implementer.)

### Resolved

- **Single PUT vs. explicit endpoints** → explicit per-member endpoints.
- **memberId generation** → API issues it on add; frontend never invents IDs; no
  separate ID-only endpoint needed.
- **Existing data cleanup** → manual, out of scope.
- **Member caps** → server-side; max on add, and payment blocked below minimum.
- **Klan/senior parity** → same treatment as patrulje/spejder;
  `reservedMemberCount` is legacy naming for seats.
- **T-shirt accounting** → counted like seats; a paid t-shirt is reusable across
  members and its size is changeable free of charge, subject to stock for the
  chosen size.
- **Over-payment / seat reduction** → a paid-but-vacant seat is left in place (no
  refund), as today.
