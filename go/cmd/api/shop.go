package main

import (
	"os"
	"sort"
	"strings"
)

// Closing a product for sale.
//
// A closed product is one that may no longer be bought, while everything already
// bought stands: its sizes are frozen, its unpaid units are dropped from open
// orders, and no payment request may include it. The year t-shirt is closed once
// the shirts go into production, because production needs a final count per size
// and every size on the page is otherwise still editable.
//
// Closing is per SKU, not per shop. `tshirt.adult` is the only thing in the
// merchandise shop today, but a plain t-shirt or a mug added later must stay
// sellable while the year shirt is closed, so the switch names products rather
// than describing a shop.
//
// See roadmap/prd/doing/003-close-product-for-sale-lock-tshirt-sizes.md.

// closedProductsEnv is the environment variable naming the closed SKUs, comma
// separated.
const closedProductsEnv = "CLOSED_PRODUCT_SKUS"

// defaultClosedSKUs is what is closed when the environment says nothing.
//
// It defaults to closed rather than open on purpose: a deployment that forgets
// the variable must not quietly resume selling shirts that are already being
// printed. Re-opening is the explicit act.
var defaultClosedSKUs = []string{"tshirt.adult"}

// closedSKUsFromEnv reads the closed-product set from the environment.
//
// A variable that is *set but empty* means "nothing is closed", which is how the
// sale is re-opened without a code change. That is why this reads the environment
// directly instead of using getEnvAsSlice, which cannot tell an empty value from
// an absent one and would hand back the default in both cases.
func closedSKUsFromEnv() map[string]bool {
	raw, ok := os.LookupEnv(closedProductsEnv)
	if !ok {
		return skuSet(defaultClosedSKUs)
	}
	return skuSet(strings.Split(raw, ","))
}

// skuSet builds a lookup set from a list of SKUs, trimming whitespace and
// dropping empties so that "a, b," and "a,b" mean the same thing.
func skuSet(skus []string) map[string]bool {
	set := make(map[string]bool, len(skus))
	for _, sku := range skus {
		if sku = strings.TrimSpace(sku); sku != "" {
			set[sku] = true
		}
	}
	return set
}

// skuClosed reports whether the given product may no longer be bought.
//
// This is the single predicate the rest of the closing rules ask: line exclusion,
// the payment-request gate, the size lock and the config the frontend renders from
// all read it, so they cannot disagree about what is closed.
func (app *application) skuClosed(sku string) bool {
	return app.config.shop.closedSKUs[sku]
}

// closedProducts is the closed set as a sorted slice, for putting on the wire.
//
// Sorted because a map iteration order would make the show endpoints' JSON differ
// between two identical requests, which breaks response caching and any test that
// compares the payload.
func (app *application) closedProducts() []string {
	out := make([]string, 0, len(app.config.shop.closedSKUs))
	for sku := range app.config.shop.closedSKUs {
		out = append(out, sku)
	}
	sort.Strings(out)
	return out
}
