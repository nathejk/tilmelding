# 018 — Frontend: migrate PatruljeView to member endpoints

**Status:** done
**Priority:** high
**Created:** 2026-07-31
**Picked up by:** agent (opus)
**Started:** 2026-07-31
**Completed:** 2026-07-31

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

- [x] Add/edit/delete member call the dedicated endpoints; `createId()` removed.
- [x] Member state is keyed on the server-issued `memberId`.
- [~] The team `PUT` no longer carries member lifecycle. Deferred to task 020:
      the roster PUT handler still re-derives the order from body members, so
      removing members from the FE payload must flip with the backend in 020.
      For now `putState` still sends members (all carry stable memberIds, so no
      churn).
- [x] Repeated edits/saves do not create duplicate members (verified against dev
      in task 015).
- [x] Cap errors (max on add, min on pay) are surfaced to the user.
- [x] `npm run build` (vue) passes.

## Progress Log

- 2026-07-31 10:00 — Task created from PRD 001.
- 2026-07-31 13:45 — Picked up. Plan: rework PatruljeView.vue — add→POST /member, edit→PUT /member/:memberId, delete→DELETE /member/:memberId; adopt returned memberId; key list/dialog on memberId; drop createId(); stop sending members in the team PUT; surface 422 cap error on add and paymentError on save. Reading the full component first.
- 2026-07-31 13:55 — Reworked saveMember (POST for new / PUT for existing, adopts returned member + order), deleteMember (DELETE endpoint, drops from list), removed createId/findIndexById/syncOrder. Keyed all member ops on memberId. Added memberError (dialog) + paymentError (near save) refs and `<Message>` displays; save() now surfaces paymentError instead of redirecting when payment is blocked.
- 2026-07-31 14:00 — Decision: kept `putState` sending members for now. The roster PUT handler (updatePatruljeHandler) still re-derives the order from body members; if the FE stopped sending them the handler would clear the order until the next GET. Flipping both sides is task 020. All current members carry stable memberIds so this causes no churn.
- 2026-07-31 14:05 — ✅ `npm run build` passes (Message chunk bundled). No dangling refs to removed helpers. Criteria met except the PUT-payload change, which is owned by task 020.
- 2026-07-31 14:06 — Completed.
