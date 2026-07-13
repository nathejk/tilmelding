# 007 — Remove dead `class List` in StaffView.vue

**Status:** open
**Priority:** low
**Created:** 2026-06-04

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
