# 019 — Frontend: migrate KlanView to member endpoints

**Status:** done
**Priority:** high
**Created:** 2026-07-31
**Picked up by:** agent (opus)
**Started:** 2026-07-31
**Completed:** 2026-07-31

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

- [x] Add/edit/delete member call the dedicated klan endpoints; ad-hoc id removed.
- [x] Member state is keyed on the server-issued `memberId`.
- [~] The klan team `PUT` no longer carries member lifecycle. Deferred to task
      020 (flipped with the backend handler), same as patrulje — `putState`
      still sends members (all carry stable memberIds) for now.
- [x] Repeated edits/saves do not create duplicate members (verified in 016).
- [x] Cap errors (max on add, min on pay) are surfaced to the user.
- [x] `npm run build` (vue) passes.

## Progress Log

- 2026-07-31 10:00 — Task created from PRD 001.
- 2026-07-31 14:10 — Picked up. Plan: mirror task 018 for KlanView.vue, plus map the FE `vegitarian` bool ↔ `diet` string when calling the klan member endpoints.
- 2026-07-31 14:20 — Reworked saveMember/deleteMember to use POST/PUT/DELETE /api/klan/{id}/member[/{memberId}], mapping vegitarian→diet on the way out and diet→vegitarian on the returned member; keyed on memberId; removed createId/findIndexById/syncOrder; added memberError + paymentError refs, `<Message>` displays, and paymentError handling in save() (alongside the existing HOLD branch).
- 2026-07-31 14:22 — Kept `putState` sending members (interim, same rationale as 018; flipped in task 020).
- 2026-07-31 14:25 — ✅ `npm run build` passes; no dangling refs. Completed.
