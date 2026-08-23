# 037 — make the catalogue the source of truth for what is closed for sale

**Status:** open
**Priority:** low
**Created:** 2026-08-23

## Description

Phase 2 of PRD 003, part two. **Depends on task 036** — doing this first breaks
every save for anyone holding a t-shirt line.

Closing a product is currently driven by `CLOSED_PRODUCT_SKUS`, an env var read in
`go/cmd/api/shop.go`. The catalogue already has the field that means the same thing
(`product.active`), and having two answers to "is this for sale" is one too many.
This task moves the truth into the catalogue and retires the variable.

**Scope**

1. **shared-go `tables/product/seeds_2026.go`:** set `Active: &no` (a `*bool`
   pointing at false) on the `tshirt.adult` seed. `Seed` already supports it —
   "Active defaults to true unless explicitly set false".

   The seed is the only durable place to put this. `seeder.Seed` upserts with
   `active=VALUES(active)` on every startup, so a manual
   `UPDATE product SET active=0` is silently undone by the next deploy and the
   shop re-opens. Worth a comment at the seed so nobody tries the shortcut.

2. **tilmelding `shop.go`:** build the closed set from the catalogue instead of the
   environment. Keep it a **snapshot taken once at startup**, right after
   `tableProduct.Seed(...)` in `main.go`:

   ```go
   cfg.shop.closedSKUs = closedSKUsFromCatalogue(ctx, tableProduct, product.Seeds2026(), cfg.year)
   ```

   Iterate the seeded SKUs and call `GetBySKU`, collecting those with `!p.Active`.
   `GetBySKU` deliberately returns inactive rows; `ListEligibleFor` filters on
   `active = 1` and so cannot see them, and adding a "list including inactive"
   query to shared-go is not worth it when the seed list already names every SKU.

   **A snapshot, not a per-request read.** `app.skuClosed(sku)` is a pure map
   lookup with no `ctx` and no error, called from eleven places including
   `sellable` and the four size locks. Making it hit the catalogue would push a
   `ctx` and an error path through all of them, for a value that only changes on
   deploy — the seed overwrites `active` at startup, so startup is precisely when
   it can change. Keep the phase-1 shape.

3. **Retire `CLOSED_PRODUCT_SKUS`**: drop `closedSKUsFromEnv`,
   `defaultClosedSKUs`, `closedProductsEnv` and their tests. Check
   `docker-compose.yml` and any deployment config for the variable before
   removing it, so nothing is left setting a variable that no longer exists.

**What does not change.** `app.sellable`, `app.chargeable` and the four size locks
all stay exactly as they are — they read `skuClosed`, and only its *source* moves.
Task 036 makes the catalogue flag safe to set; this task makes it authoritative. It
is a refactor with no behavioural change, which is also how it should be tested:
the closed-product behaviour verified in tasks 031-035 must be identical
afterwards.

## Risks

- **Fails open, unlike phase 1.** The env var defaults to *closed*
  (`defaultClosedSKUs`), so a misconfiguration keeps the shop shut. A catalogue
  read that fails, or a seed someone edits, defaults to **open** — `Active` is true
  by default and an unreadable catalogue yields an empty closed set. Consider
  logging loudly at startup when the closed set comes back empty, and think about
  whether an empty set should be treated as suspicious rather than as "everything
  is for sale".
- **Ordering.** Landing this before task 036 breaks saves for every owner with a
  t-shirt line. The dependency is not advisory.

## Acceptance Criteria

- [ ] `tshirt.adult` is `Active: false` in `Seeds2026`, with a comment explaining
      that the seed — not a manual UPDATE — is the durable switch
- [ ] `config.shop.closedSKUs` is built from the catalogue at startup
- [ ] `app.skuClosed` is still a pure lookup: no `ctx`, no error, no new arguments
- [ ] `CLOSED_PRODUCT_SKUS` is gone from the code and from any deployment config
- [ ] Startup logs which products are closed for sale
- [ ] `config.closedProducts` still reports `["tshirt.adult"]` on all four show
      endpoints
- [ ] Behaviour is unchanged: unpaid units still cancelled, sizes still locked, no
      payment request includes a closed product, no credit lines for paid units
- [ ] `go build ./...` / `go test ./...` pass in the workspace and with `GOWORK=off`

## Progress Log

- 2026-08-23 — Task created as phase 2 of PRD 003. Low priority: phase 1 already
  closed the shop correctly, so this only removes the duplicate source of truth.
  The one substantive question it raises is the fail-open/fail-closed change noted
  above — worth deciding deliberately rather than inheriting.
