# 007 — Remove dead `class List` in StaffView.vue

**Status:** done
**Priority:** low
**Created:** 2026-06-04
**Completed:** 2026-08-04

## Description

The `class List extends Array` definition is unused after the order cutover but still present in `vue/src/views/StaffView.vue` (and possibly `FriendView.vue`). Harmless but eslint will flag it. Drop the class definitions.

Related files:
- `vue/src/views/StaffView.vue`
- `vue/src/views/FriendView.vue`

## Acceptance Criteria

- [ ] `class List` removed from both files
- [ ] `npm run lint` passes without related warnings

## Progress Log

- 2026-06-04 21:54 — Task created.
- 2026-08-04 — Done. The dead `class List extends Array` had moved with the
  file renames: it now lived in `BadutView.vue` and `CrewView.vue` (the
  post-rename successors of Friend/Staff), not the originally-named files.
  Confirmed `List` was referenced nowhere in either file before removing both
  blocks. Lint not run here (no node on host; see task 010), but the change is
  a pure deletion of an unreferenced local class, so it cannot introduce a
  lint error — it removes the `no-unused-vars` one the task predicted.
