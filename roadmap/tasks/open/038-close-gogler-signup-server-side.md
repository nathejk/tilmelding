# 038 — close the gøgler signup server-side

**Status:** open
**Priority:** high
**Created:** 2026-08-23

## Description

The gøgler signup is supposed to be closed. `home.go:32` marks it
`Status: "CLOSED"`, but that only disables a button on the front page
(`HomeView.vue:145`) — **nothing on the server refuses a gøgler signup.**

`createSignupHandler` (`go/cmd/api/signup.go:78`) reads `input.TeamType` straight
from the request body and hands it to `signup.Signup`, which interpolates it into
the NATS subject without any validation. So this still creates a gøgler today:

```
POST /api/signup {"type":"gøgler","name":"…","emailPending":"…","phonePending":"…"}
```

A seat is claimed at signup, before any order exists, so this is the only place
that can refuse one.

**Why not the product catalogue.** `participation.gogler.active = 0` or
`stock = 0` looks like the natural switch and is the wrong tool here, for two
reasons:

- **Too late.** The catalogue is consulted when order *lines* are built — after the
  person has signed up, verified email, verified phone and reached the badut page.
  They would get all the way there and then hit a failure.
- **It breaks the wrong people.** `order.buildLines` rejects any desired line whose
  product is inactive, and `derivedLinesForPersonnel` emits `participation.gogler`
  on every show and save for everyone *already* signed up — so their page breaks
  while new signups sail through. `stock = 0` is worse: `checkStock`'s participation
  branch is `reservedElsewhere >= *p.Stock`, which with stock 0 is always true, so
  it rejects every gøgler order including paid ones. This is PRD 003 §8.5 again;
  task 036 is the prerequisite for the catalogue being usable at all, and task 037
  is where this switch should eventually move.

**Scope**

1. `config.signup.closedTypes` in `main.go`, read from `CLOSED_SIGNUP_TYPES`,
   defaulting to the gøgler type so a deployment that forgets the variable fails
   **closed**. `CLOSED_SIGNUP_TYPES=""` re-opens everything.

   Watch the naming split: the route is `/indskrivning/badut` and the Go identifier
   is `TeamTypeBadut`, but the wire and DB value is `types.TeamTypeBadut ==
   "gøgler"`. Accept `badut` as an alias when parsing, so setting
   `CLOSED_SIGNUP_TYPES=badut` cannot silently do nothing — and so nobody has to
   type `ø` into a deployment config.

2. `app.signupClosed(types.TeamType) bool` — one predicate, same shape as
   `app.skuClosed`.

3. `createSignupHandler` refuses a closed type before publishing anything, with a
   Danish message.

4. `homeHandler` derives the gøgler button's status from the same set instead of
   hard-coding `"CLOSED"`. This is the actual bug: the displayed status and the
   (missing) enforcement were two separate facts, which is how they came to
   disagree.

**Non-goals**

- **Klan.** Also hard-closed by hand (`home.go:20`, which throws away the
  catalogue-derived cap computed just above it) and has the same hole. Out of scope
  by request; the mechanism this task adds will fit it later.
- **No waiting list.** A closed gøgler signup is refused outright, not recorded as
  interested. Klan's `WAITINGLIST` status and `VentelisteView` stay klan-only.
- **Unknown team types.** `createSignupHandler` accepts *any* string as `type` and
  publishes it into a subject (`NATHEJK:<year>.<type>.<id>.signedup`). Worth
  fixing — `types.TeamTypes` already lists the valid four — but it is a separate
  hole from this one and should not ride along silently.

## Acceptance Criteria

- [ ] `POST /api/signup` with `type=gøgler` is refused, and no `signedup` event is
      published
- [ ] `POST /api/signup` with `type=patrulje` still works exactly as before
- [ ] The refusal carries a Danish message the frontend can show
- [ ] `/api/home` reports the gøgler signup as `CLOSED` derived from the same
      config, not hard-coded
- [ ] With `CLOSED_SIGNUP_TYPES=""` the gøgler signup works again, with no code
      change
- [ ] `CLOSED_SIGNUP_TYPES=badut` closes the gøgler signup (alias accepted)
- [ ] Unit tests for the parsing (including the alias and the empty case) and the
      predicate
- [ ] OpenAPI annotations on `POST /api/signup` state that a closed signup type is
      refused
- [ ] `go build ./...` / `go test ./...` pass in the workspace and with `GOWORK=off`

## Progress Log

- 2026-08-23 — Task created. Found while answering "can I use the product catalogue
  to close the gøgler seats?" — the answer is no, not until task 036, and the real
  gap was that no signup type has ever been enforced server-side.
