package main

import (
	"os"
	"sort"
	"strings"

	"github.com/nathejk/shared-go/tables/order"
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

// sellable drops desired lines for products that are closed for sale.
//
// This is what cancels an unpaid t-shirt: the size stays on the member's
// projection (nothing is deleted), but the line stops being derived, so it leaves
// the open order and the amount due falls by its price. A unit that has already
// been paid for is unaffected — it lives on an immutable paid order, and
// ApplyPaidOffset produces nothing for a paid unit with no desired counterpart
// ("a reduction is not a size change, and this mechanism does not refund"), so
// filtering here cannot emit a credit line for a shirt somebody owns.
//
// It also means no payment request can include a closed product, because a charge
// is built from the order's own lines and amount. `chargeable` covers the one case
// this does not: an order exempted below, which still carries its line.
//
// The exemption: an order with money in flight is left exactly as its payer saw
// it. PaidAmount counts payments in ('reserved','received') for that order — money
// the payer has actually committed — so a shrinking order can never undercut a
// payment on its way to settling. Merely 'requested' payments are not counted, and
// must not be: every save with something due issues one, so treating a request as
// in flight would exempt nearly every order and cancel nothing. Nor could we tell
// a live request from an abandoned one — nothing writes 'timedout' or 'rejected',
// so an abandoned link stays 'requested' forever. MobilePay expires links after
// ten minutes, which is what bounds the exposure instead. See PRD 003 §8.2b.
//
// Returns the input unchanged when nothing is closed, so the open-shop path
// allocates nothing.
func (app *application) sellable(o *order.Order, lines []order.DesiredLine) []order.DesiredLine {
	if len(app.config.shop.closedSKUs) == 0 || len(lines) == 0 {
		return lines
	}
	if o != nil && o.PaidAmount > 0 {
		return lines
	}
	out := make([]order.DesiredLine, 0, len(lines))
	for _, l := range lines {
		if app.skuClosed(l.ProductSKU) {
			continue
		}
		out = append(out, l)
	}
	return out
}

// closedLines reports whether an order carries any line for a product that is
// closed for sale.
func (app *application) closedLines(o *order.Order) bool {
	if o == nil || len(app.config.shop.closedSKUs) == 0 {
		return false
	}
	for _, l := range o.Lines {
		if app.skuClosed(l.ProductSKU) {
			return true
		}
	}
	return false
}

// chargeClosedProduct is what the user is told when their order cannot be turned
// into a payment request because it still contains something no longer for sale.
//
// It names a wait rather than a fault, because that is what it is: the order is
// holding a line for a closed product only because a payment is already in flight
// against it, and MobilePay drops an unapproved request after ten minutes. Either
// that payment lands or it expires; either way the next save issues a link.
const chargeClosedProduct = "afventer en igangværende betaling — prøv igen om 10 minutter"

// chargeable reports whether an order may be turned into a payment request, and
// if not, the Danish explanation to show the user.
//
// This is the invariant the whole closing mechanism rests on: **no new payment
// request may include a product that is closed for sale.** sellable gets that
// almost for free — with the line off the order, neither the charged amount
// (DueAmount) nor the receipt (paymentLinesFromOrder) can contain it — but not
// quite. An order with money already in flight is exempted from sellable and does
// still carry its line, and a save on such an order would otherwise mint a fresh
// link whose amount includes the closed product. That is a new sale of something
// withdrawn from sale, which is exactly what must not happen, so it is refused
// here rather than left to follow from a filter elsewhere.
//
// Refusing is not an error: the caller returns an empty payment link and this
// message. The state clears itself, since the exemption that caused it ends when
// the in-flight payment settles or expires and the next recompute drops the line.
func (app *application) chargeable(o *order.Order) (bool, string) {
	if app.closedLines(o) {
		return false, chargeClosedProduct
	}
	return true, ""
}
