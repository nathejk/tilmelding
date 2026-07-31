# 019 — Frontend: migrate KlanView to member endpoints

**Status:** open
**Priority:** high
**Created:** 2026-07-31

## Description

Klan equivalent of task 018, per PRD 001
(`roadmap/prd/001-seat-based-billing-and-member-identity.md`).

Migrate `vue/src/views/KlanView.vue` to the explicit klan member endpoints
(task 016):

- Add member → `POST /api/klan/{teamId}/member`; store the returned `memberId`.
- Edit member → `PUT /api/klan/{teamId}/member/{memberId}`.
- Delete member → `DELETE /api/klan/{teamId}/member/{memberId}`.
- Key member state on `memberId`; remove any ad-hoc local id.
- Team `PUT` no longer carries member lifecycle.
- Seat-based "at betale"; surface max/min cap errors.

Related files:
- `vue/src/views/KlanView.vue`

## Acceptance Criteria

- [ ] Add/edit/delete member call the dedicated klan endpoints; ad-hoc id removed.
- [ ] Member state is keyed on the server-issued `memberId`.
- [ ] The klan team `PUT` no longer carries member lifecycle.
- [ ] Repeated edits/saves do not create duplicate members.
- [ ] Cap errors (max on add, min on pay) are surfaced to the user.
- [ ] `npm run build` (vue) passes.

## Progress Log

- 2026-07-31 10:00 — Task created from PRD 001.
