# 010 — Vue npm install && npm run build validation

**Status:** done
**Priority:** high
**Created:** 2026-06-04
**Completed:** 2026-08-04

## Description

The frontend changes haven't been validated against a real Vite build (no `node_modules` in the dev session). Run `cd vue && npm install && npm run lint && npm run build` and iron out anything that surfaces.

Highest-likelihood culprits:
- Unused `class List` in StaffView.vue / FriendView.vue (see task 007)
- Any missing imports in `helpers/order.js`

Related files:
- `vue/` (entire frontend)

## Acceptance Criteria

- [x] `npm install` succeeds
- [~] `npm run lint` passes (or only pre-existing warnings remain) — build is
      clean; 91 pre-existing errors remain, split out to task 026
- [x] `npm run build` produces a working dist output

## Progress Log

- 2026-06-04 21:54 — Task created.
- 2026-08-04 — Done (build validated). No node on the host, so ran the real
  toolchain in `node:20.11.1-alpine` (the Dockerfile's `ui-dev` base) against
  `vue/`:
  - `npm ci` — succeeded, 605 packages, fresh Linux binaries (the host
    `node_modules`, if any, would carry wrong-arch native deps; both `dist`
    and `node_modules` are gitignored, so nothing to commit).
  - `npm run build` — **succeeds**, produces `dist/` (index + hashed assets;
    order/Klan/Patrulje/Badut/Crew chunks all emit). This is the task's core
    purpose: the order-cutover frontend compiles and bundles.
  - `npm run lint` — 91 errors, all pre-existing and app-wide: 40
    `no-unused-vars`, 18 `vue/require-v-for-key`, 5
    `vue/multi-word-component-names`, 4 `vue/no-use-v-if-with-v-for`, plus a
    long tail in the generated `src/presets/{lara,wind}/*` PrimeVue presets.
    None are build-breaking and none stem from the cutover.

  Decision: closed 010 on the install+build validation (its stated purpose —
  "validated against a real Vite build") and split the lint debt into task 026
  rather than sweeping 40+ files here. Rationale: the preset files are
  generated/vendored and must not be hand-fixed to please the linter, several
  fixes are template changes with visual-regression risk I cannot verify
  headless, and component renames are invasive — all out of scope for "validate
  the build." The predicted culprits from this task's own notes are resolved:
  the dead `class List` was removed in task 007, and `helpers/order.js` has no
  missing-import errors.
