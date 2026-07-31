# 018 — Frontend: migrate PatruljeView to member endpoints

**Status:** open
**Priority:** high
**Created:** 2026-07-31

## Description

Frontend half of PRD 001
(`roadmap/prd/001-seat-based-billing-and-member-identity.md`) for patrulje.

`vue/src/views/PatruljeView.vue` currently tracks members with an ad-hoc local
`createId()` key, never adopts the server `memberId`, and pushes the whole roster
through `PUT /api/patrulje/{id}` on every edit (`syncOrder`/`putState`), which is
what recreates members.

Migrate to the explicit member endpoints (task 015):

- Add member → `POST /api/patrulje/{teamId}/member`; store the returned
  `memberId` on the member.
- Edit member → `PUT /api/patrulje/{teamId}/member/{memberId}`.
- Delete member → `DELETE /api/patrulje/{teamId}/member/{memberId}`.
- Key all member operations (edit / delete / index lookup) on `memberId`; remove
  `createId()`.
- Stop sending members through the team `PUT` (it becomes team/contact only).
- Show the seat-based "at betale" (count-based due from the recomputed order).
- Surface the "team is full" (max) and "too few members to pay" (min) errors
  from the API (tasks 015/017).

Related files:
- `vue/src/views/PatruljeView.vue`

## Acceptance Criteria

- [ ] Add/edit/delete member call the dedicated endpoints; `createId()` removed.
- [ ] Member state is keyed on the server-issued `memberId`.
- [ ] The team `PUT` no longer carries member lifecycle.
- [ ] Repeated edits/saves do not create duplicate members (verified against dev).
- [ ] Cap errors (max on add, min on pay) are surfaced to the user.
- [ ] `npm run build` (vue) passes.

## Progress Log

- 2026-07-31 10:00 — Task created from PRD 001.
