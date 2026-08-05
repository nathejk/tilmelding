# 026 — Frontend lint debt (91 eslint errors)

**Status:** open
**Priority:** low
**Created:** 2026-08-04

## Description

`npm run lint` reports 91 eslint errors across the `vue/` frontend, surfaced
while validating the Vite build in task 010. None block the build (`npm run
build` succeeds) and they are pre-existing — spread app-wide, not caused by the
order cutover — so they were split out here rather than fixed under 010.

Breakdown by rule (as of 2026-08-04, eslint 8 via `node:20.11.1-alpine`):

| Count | Rule | Notes |
|---|---|---|
| 40 | `no-unused-vars` | unused imports/vars; mostly safe to delete, but eslint-plugin-vue already accounts for template refs, so verify each |
| 18 | `vue/require-v-for-key` | needs a correct `:key` expression per loop — judgement, visual-regression risk |
| 5 | `vue/multi-word-component-names` | `Test.vue`, `Shop.vue`, etc.; renaming a component touches every usage |
| 4 | `vue/no-use-v-if-with-v-for` | `v-if` + `v-for` on the same element; restructuring templates |
| 2 | `vue/no-useless-template-attributes` | |
| 1 | `vue/no-parsing-error` | a real parse issue — locate and fix first |
| 1 | `vue/no-dupe-keys` | duplicate key in an object/component option |

Files with the most: `PatruljeView.vue`, `KlanView.vue`, `CrewView.vue`,
`BadutView.vue` (10 each), then the `Payment/Thankyou/Venteliste/Verify` views
and a long tail in `src/presets/lara/*` and `src/presets/wind/*`.

## Constraints / gotchas

- **`src/presets/lara/*` and `src/presets/wind/*` are PrimeVue theme presets**
  (vendored/generated). Do not hand-fix them to satisfy the linter; either add
  them to `.eslintignore` (they are not our source) or fix upstream/regenerate.
  Decide this explicitly.
- Template changes (`require-v-for-key`, `no-use-v-if-with-v-for`) can change
  rendering. Verify visually — a wrong `:key` can reintroduce list-diffing
  bugs.
- `vue/multi-word-component-names` on `Test.vue`/`Shop.vue`: renaming is
  invasive; consider whether `Test.vue` is even still used (dead file?).
- No node/npm on the host in this repo's dev flow; run lint in the container:
  `docker run --rm -v <repo>/vue:/app -w /app node:20.11.1-alpine sh -c "npm ci && npm run lint"`
  or via the `ui` compose service.

## Acceptance Criteria

- [ ] Decision recorded on the preset files (eslintignore vs fix vs regenerate)
- [ ] `vue/no-parsing-error` located and fixed
- [ ] Remaining `no-unused-vars` cleared (safe deletions)
- [ ] `require-v-for-key` / `no-use-v-if-with-v-for` fixed with visual checks
- [ ] `npm run lint` exits clean (0 errors), `npm run build` still succeeds

## Progress Log

- 2026-08-04 — Created from task 010. The build was validated there; this
  captures the pre-existing lint debt so it is tracked rather than dropped, and
  so 010 (build validation) can close without a risky 40-file lint sweep.
