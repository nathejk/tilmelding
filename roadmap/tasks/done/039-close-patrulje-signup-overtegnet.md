# 039 — close the patrulje signup: overtegnet at 185 teams

**Status:** done
**Priority:** high
**Created:** 2026-09-07
**Picked up by:** agent session (Zed)
**Started:** 2026-09-07
**Completed:** 2026-09-07

## Description

Production reached **185 patruljer**. The patrulje signup must close, in all three
places a person can still reach it:

1. The front page button must be disabled.
2. A deeplink past the front page (`/indskrivning/patrulje`) must refuse the signup
   with a visible error rather than a button that does nothing.
3. A team page link handed out earlier must lock the page and lead with a very
   visible box: *"Nathejk er overtegnet for i år, tak for interessen"*.

Task 038 already built the mechanism for (1) and (2) — `CLOSED_SIGNUP_TYPES`, one
set read by both `homeHandler` and `createSignupHandler` so the displayed status and
the enforcement cannot drift. This task reuses it and adds two things it did not
need:

**A separate set, `OVERSUBSCRIBED_SIGNUP_TYPES`, defaulting to `patrulje`.** Not an
extra entry in `CLOSED_SIGNUP_TYPES`, for two reasons:

- Production sets `CLOSED_SIGNUP_TYPES` explicitly (`badut`) and its env lives
  outside this repo. Changing `defaultClosedSignupTypes` would therefore have been a
  **no-op in production** — the explicit value wins — while looking like a fix. A new
  variable's default applies because nothing sets it.
- "Closed" and "full" are the same refusal with different things to say. Closed is
  administrative and terse ("tilmelding er lukket"); overtegnet is the event being
  full, is worded once in `signupOversubscribedMessage`, and is the only state that
  reaches past the signup flow into an existing team's page.

`signupClosed` is `closedTypes || oversubscribedTypes`, so the front-page button and
the create endpoint follow from the new set without either learning about it.

**A `config.oversubscribed` flag on `GET /api/patrulje/{id}`**, decided **per team**,
not per type. The 185 patruljer that signed up before the close keep their page in
full — editing the team, adding and removing members, and paying. Locking every
patrulje page would have taken their roster and their payment button away, which is
the opposite of what closing is for. So the page asks a narrower question than the
front page: not "is this type full" but "is *this team* one that did not make it",
answered by `signup.createdAt` against `OVERSUBSCRIBED_SINCE` (the moment of the
close). Anything created at or after the cut-off gets the banner; everything before
it is untouched.

`signup.createdAt` is the only timestamp available — the patrulje projection has no
created column, and status, team number and roster size are all things an admitted
team can legitimately lack, so keying on any of them would have locked out a team
that had simply not added its members yet. The column is a VARCHAR holding
`time.Time`'s `String()` output (the projection writes `createdAt=%q` from the message
time), not RFC 3339, hence `parseSignupCreatedAt` and its layout list.

Unlike the type set, the cut-off **fails open**: an empty, missing or unparseable
value, a missing signup row, or an unreadable timestamp all leave the page unlocked.
Wrongly locking an admitted team costs it its roster and its payment; wrongly leaving
one page open costs one page, and the front door is shut regardless.

The page locks with `inert` plus a greyed-out class — `inert` because
`pointer-events-none` alone leaves every field reachable by keyboard, i.e. a form that
looks dead and is still editable. `save`, `saveMember` and `pay` also guard on the
flag, since `inert` is unsupported in older browsers and those are the clicks that
would take money for a place that no longer exists.

**Non-goals**

- **Server-side write refusal on the team page.** The lockdown is frontend-only. The
  patrulje write endpoints are the same ones an admitted team uses to edit its roster
  and to pay, and the server already knows which teams are affected, so this would
  now be a small addition — but it is a second refusal surface and belongs in its own
  change.
- **Klan.** Still hard-closed by hand in `home.go` (and still has 038's hole: the
  front page says CLOSED while `POST /api/signup {"type":"klan"}` is accepted).
  Unchanged here; the new set would fit it in one line.
- **No waiting list.** An overtegnet patrulje signup is refused outright, not
  recorded as interested.

## Acceptance Criteria

- [x] The front page reports the patrulje signup as `CLOSED`, derived from the new
      set, so the button is disabled
- [x] `POST /api/signup {"type":"patrulje"}` is refused with 422 and the overtegnet
      message, publishing nothing
- [x] `IndskrivningView` surfaces that message (already did, via 038's toast)
- [x] `GET /api/patrulje/{id}` carries `config.oversubscribed`
- [x] A team whose signup predates the close is **never** flagged: it can still edit
      the team, add and remove members, and pay
- [x] A team whose signup is at or after the close leads with the overtegnet box and
      is read-only
- [x] `OVERSUBSCRIBED_SIGNUP_TYPES=""` re-opens everything, with no code change
- [x] `OVERSUBSCRIBED_SINCE=""` locks no team page, and an unparseable value does the
      same rather than locking every page
- [x] Unit tests for the parsing, the closed-implies-full predicate, the message
      choice, the cut-off parsing (including `createdAt`'s non-RFC-3339 format) and
      the wire flag
- [x] Wire-shape golden tests updated for the new config key
- [x] OpenAPI annotations mention the new variable and the new config field
- [x] `go build ./...` and `go test ./cmd/api/` pass

## Progress Log

- 2026-09-07 — Considered changing `defaultClosedSignupTypes` to include `patrulje`
  and stopped: production pins `CLOSED_SIGNUP_TYPES` outside this repo, so the
  default is dead code there. A separate variable is the only version of this that
  works without a deployment change.
- 2026-09-07 — Frontend lockdown uses `inert` rather than `disabled` on ~20
  PrimeVue components. One attribute on the container, and it also removes the
  fields from the tab order, which `pointer-events-none` does not.
- 2026-09-07 — **Caveat resolved.** The first cut derived the flag from the type
  alone, which locked *every* patrulje page — including the 185 teams that are in,
  who could then no longer edit their roster or pay. Flagged rather than shipped
  quietly; the requester confirmed those teams must keep editing and adding members,
  so the flag became per team via `signup.createdAt` and `OVERSUBSCRIBED_SINCE`.
  Consequence worth knowing: with the cut-off set to the close, the teams that see
  the banner are the signups that were mid-funnel at the close and finish their
  verification afterwards. Nobody who was already on a team page is affected.
- 2026-09-07 — `teamOversubscribed` costs one extra `signup` read per patrulje page
  load, by team ID on the primary key. Cheap, and the alternative was carrying a
  created timestamp into the patrulje projection, which is a schema change and a
  replay.
- 2026-09-07 — Not verified against the running dev stack (no container run in this
  session); the parsing and the predicates are unit-tested, and `docker-compose.yml`
  documents both variables for dev. The one thing worth checking live is that
  production's `signup.createdAt` really is in `time.Time.String()` format for old
  rows — if some are not, those pages fail open, which is the harmless direction.
