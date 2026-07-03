# 010 — Vue npm install && npm run build validation

**Status:** open
**Priority:** high
**Created:** 2026-06-04

## Description

The frontend changes haven't been validated against a real Vite build (no `node_modules` in the dev session). Run `cd vue && npm install && npm run lint && npm run build` and iron out anything that surfaces.

Highest-likelihood culprits:
- Unused `class List` in StaffView.vue / FriendView.vue (see task 007)
- Any missing imports in `helpers/order.js`

Related files:
- `vue/` (entire frontend)

## Acceptance Criteria

- [ ] `npm install` succeeds
- [ ] `npm run lint` passes (or only pre-existing warnings remain)
- [ ] `npm run build` produces a working dist output

## Progress Log

- 2026-06-04 21:54 — Task created.
