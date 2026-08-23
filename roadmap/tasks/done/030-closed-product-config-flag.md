# 030 — closed-product config flag and `config.closedProducts` on the show endpoints

**Status:** done
**Priority:** high
**Created:** 2026-08-23
**Picked up by:** agent session (Zed)
**Started:** 2026-08-23
**Completed:** 2026-08-23

## Description

Foundation for PRD 003 (close a product for sale). Introduces the switch every
other task in the set reads, and puts it on the wire so the frontend can react.

Nothing behavioural changes in this task: no line is excluded, no size is locked.
It ships the flag, the helper and the wire key, and leaves the repo green.

**Scope**

1. `config` in `go/cmd/api/main.go` (struct at L49-74) gains a `shop` group:

   ```go
   shop struct {
       closedSKUs map[string]bool
   }
   ```

   Populated from `getEnvAsSlice("CLOSED_PRODUCT_SKUS", []string{"tshirt.adult"}, ",")`
   — the helper already exists in `go/cmd/api/env.go`. The default is
   **`tshirt.adult` closed**, so a deployment that forgets the variable fails safe
   rather than fails open. Setting `CLOSED_PRODUCT_SKUS=""` re-opens everything.

2. `func (app *application) skuClosed(sku string) bool` — the single predicate the
   rest of PRD 003 asks. Keep it on `*application` so it is trivially stubbed in
   handler tests.

3. `TeamConfig` in `go/cmd/api/patrulje.go` (L268-296) gains
   `ClosedProducts []string`; `buildTeamConfig` fills it from the config, sorted so
   the JSON is deterministic. All four show handlers already funnel through
   `buildTeamConfig` (`patrulje.go:332`, `klan.go:229`, `crew.go:69`,
   `personnel.go:34`), so this is one edit for four endpoints.

4. `newTeamConfigResponse` (`go/cmd/api/response.go`) gains the `closedProducts`
   JSON key.

**Deliberately unchanged:** `tshirtPrice` and `tshirtSizes` stay on the config and
stay populated. A closed product still has a price and sizes — they are needed to
render what somebody already bought, and `tshirtSizeLabels` lookups depend on
them.

**Why an env var and not `product.active`:** see PRD 003 §8.5. `order.buildLines`
rejects *any* desired line whose product is inactive, and every request re-derives
the full desired set, so flipping `active` breaks saves for everyone holding a
shirt — including those whose shirts are paid. Phase 2 of the PRD fixes that
upstream and retires this variable.

## Acceptance Criteria

- [x] `CLOSED_PRODUCT_SKUS` parsed into the config, defaulting to `tshirt.adult`
- [x] `app.skuClosed("tshirt.adult")` true by default; false when the variable is
      set to an empty string
- [x] `config.closedProducts` present on `GET /api/patrulje/:id`,
      `GET /api/klan/:id`, `GET /api/crew/:id`, `GET /api/personnel/:id`
- [x] `TestShowKlanResponseWireShape` and `TestShowPatruljeResponseWireShape`
      updated for the new key (they compare literal JSON strings, so they fail by
      design until then)
- [x] OpenAPI `@Description` on the four show handlers mentions `closedProducts`
- [x] No behavioural change: order lines, totals and t-shirt editing are exactly as
      before this task
- [x] `go build ./...` / `go test ./...` pass in the workspace and with `GOWORK=off`

## Progress Log

- 2026-08-23 — Task created from PRD 003 §8.1. Blocks tasks 031-035.
- 2026-08-23 — Picked up. Plan: config group + `shop.go` for the closing rules,
  then the wire key, then tests and annotations.
- 2026-08-23 — Created `go/cmd/api/shop.go` as the home for all closed-product
  rules (tasks 031-034 add `sellable`, `chargeable` and `lockedSize` beside
  `skuClosed`). Added `config.shop.closedSKUs` in `main.go`.
- 2026-08-23 — Did **not** use `getEnvAsSlice` for the flag, despite it existing.
  It cannot distinguish an unset variable from one set to `""` — both return the
  default — which would have made `CLOSED_PRODUCT_SKUS=""` mean "tshirt.adult is
  closed" instead of "nothing is closed", i.e. no way to re-open the sale without
  a code change. `closedSKUsFromEnv` reads `os.LookupEnv` directly and documents
  why.
- 2026-08-23 — ✅ Criteria 1-2: `shop_test.go` covers trimming, the unset/empty
  distinction, per-SKU scoping (closing the year shirt leaves
  `participation.*`, `tshirt.plain` and `mug.enamel` open) and sorted,
  never-nil `closedProducts`.
- 2026-08-23 — `ClosedProducts` added to both `TeamConfig` (`patrulje.go`) and
  `teamConfigResponse` (`response.go`), normalised to `[]` rather than `null` so
  the frontend can call `.includes()` without a nil guard.
- 2026-08-23 — Noted while wiring: the crew and personnel show handlers serialise
  the raw `TeamConfig`, while patrulje and klan go through
  `newTeamConfigResponse`. Pre-existing inconsistency, left alone — the json tag
  is the same on both types, so all four endpoints expose the key identically.
  Verified by marshalling both shapes: `"closedProducts":["tshirt.adult"]`.
- 2026-08-23 — ✅ Criteria 3-5. The crew and personnel show handlers had **no**
  OpenAPI annotations at all, so they got a full block rather than an edited
  description. Other unannotated endpoints (signup, sections, payment) are left
  as they are: out of scope here, and worth their own task.
- 2026-08-23 — ✅ Criteria 6-7: `go build ./...` and `go test ./...` pass both in
  the workspace and with `GOWORK=off`. No behavioural change in this task — the
  flag is read but nothing consumes it yet. Moving to done; 031 is unblocked.
