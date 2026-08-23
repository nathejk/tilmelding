# 030 — closed-product config flag and `config.closedProducts` on the show endpoints

**Status:** open
**Priority:** high
**Created:** 2026-08-23

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

- [ ] `CLOSED_PRODUCT_SKUS` parsed into the config, defaulting to `tshirt.adult`
- [ ] `app.skuClosed("tshirt.adult")` true by default; false when the variable is
      set to an empty string
- [ ] `config.closedProducts` present on `GET /api/patrulje/:id`,
      `GET /api/klan/:id`, `GET /api/crew/:id`, `GET /api/personnel/:id`
- [ ] `TestShowKlanResponseWireShape` and `TestShowPatruljeResponseWireShape`
      updated for the new key (they compare literal JSON strings, so they fail by
      design until then)
- [ ] OpenAPI `@Description` on the four show handlers mentions `closedProducts`
- [ ] No behavioural change: order lines, totals and t-shirt editing are exactly as
      before this task
- [ ] `go build ./...` / `go test ./...` pass in the workspace and with `GOWORK=off`

## Progress Log

- 2026-08-23 — Task created from PRD 003 §8.1. Blocks tasks 031-035.
