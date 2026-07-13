# 004 — Per-size t-shirt inventory

**Status:** open
**Priority:** low
**Created:** 2026-06-04

## Description

Product stock is per-SKU only — `tshirt.adult` has one stock pool for all sizes. The line's `Attributes.size` is already captured, so the data is there for finer-grained enforcement.

If needed ("50 XL, 200 M" enforcement):
- Add a `product_variant_stock` table keyed on `(sku, year, attribute_set)`.
- Update `order.commander.checkStock` to read variant stock when the product's attribute schema declares variants.
- Update `aggregateOrderLines` in `vue/src/helpers/order.js` for per-variant remaining-stock display.

Related files:
- `go/nathejk/table/product/` — schema, querier
- `go/nathejk/table/order/commander.go` — `checkStock`

## Acceptance Criteria

- [ ] `product_variant_stock` table (or equivalent) exists
- [ ] `checkStock` enforces per-variant stock for products that declare variants
- [ ] Seed for `tshirt.adult` specifies per-size stock values
- [ ] FE can display remaining stock per size (optional, depending on UX needs)

## Progress Log

- 2026-06-04 21:54 — Task created.
