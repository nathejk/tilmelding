package main

import (
	"testing"

	"github.com/nathejk/shared-go/tables/order"
)

// closedApp is an application whose only configured behaviour is the closed-SKU
// set, which is all sellable / closedLines read.
func closedApp(skus ...string) *application {
	app := &application{}
	app.config.shop.closedSKUs = stringSet(skus)
	return app
}

func skus(lines []order.DesiredLine) []string {
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		out = append(out, l.ProductSKU)
	}
	return out
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestSellableDropsClosedSKUs(t *testing.T) {
	app := closedApp("tshirt.adult")
	desired := []order.DesiredLine{
		{ProductSKU: "participation.patrulje", MemberID: "a", Quantity: 1},
		{ProductSKU: "tshirt.adult", MemberID: "a", Quantity: 1, Attributes: map[string]any{"size": "l"}},
		{ProductSKU: "participation.patrulje", MemberID: "b", Quantity: 1},
		{ProductSKU: "tshirt.adult", MemberID: "b", Quantity: 1, Attributes: map[string]any{"size": "xxl"}},
	}

	got := skus(app.sellable(&order.Order{}, desired))
	want := []string{"participation.patrulje", "participation.patrulje"}
	if !equalStrings(got, want) {
		t.Errorf("sellable() = %v, want %v", got, want)
	}
}

// The point of a per-SKU switch. Closing the year shirt must not disturb the
// participation lines or any other merchandise the catalogue grows.
func TestSellableLeavesOtherProductsAlone(t *testing.T) {
	app := closedApp("tshirt.adult")
	desired := []order.DesiredLine{
		{ProductSKU: "participation.klan", MemberID: "a", Quantity: 1},
		{ProductSKU: "tshirt.plain", MemberID: "a", Quantity: 1},
		{ProductSKU: "mug.enamel", MemberID: "a", Quantity: 1},
		{ProductSKU: "tshirt.adult", MemberID: "a", Quantity: 1},
	}

	got := skus(app.sellable(&order.Order{}, desired))
	want := []string{"participation.klan", "tshirt.plain", "mug.enamel"}
	if !equalStrings(got, want) {
		t.Errorf("sellable() = %v, want %v", got, want)
	}
}

// Money already committed to this order (reserved or received, which is what
// PaidAmount counts) means the payer is mid-payment. Their order must not shrink
// underneath them, so the filter steps aside entirely.
func TestSellableSkipsOrdersWithMoneyInFlight(t *testing.T) {
	app := closedApp("tshirt.adult")
	desired := []order.DesiredLine{
		{ProductSKU: "participation.patrulje", MemberID: "a", Quantity: 1},
		{ProductSKU: "tshirt.adult", MemberID: "a", Quantity: 1},
	}

	o := &order.Order{OrderID: "o-1", TotalAmount: 42500, PaidAmount: 42500}
	if got := len(app.sellable(o, desired)); got != 2 {
		t.Errorf("sellable() kept %d lines, want 2 — an order with money in flight must be left alone", got)
	}

	// A partially paid order counts too: the payment may still be settling.
	o = &order.Order{OrderID: "o-2", TotalAmount: 42500, PaidAmount: 25000}
	if got := len(app.sellable(o, desired)); got != 2 {
		t.Errorf("sellable() kept %d lines, want 2 for a partially paid order", got)
	}

	// Nothing committed: the unpaid shirt goes.
	o = &order.Order{OrderID: "o-3", TotalAmount: 42500, PaidAmount: 0}
	if got := len(app.sellable(o, desired)); got != 1 {
		t.Errorf("sellable() kept %d lines, want 1 for an unpaid order", got)
	}
}

func TestSellableIsANoOpWhenNothingIsClosed(t *testing.T) {
	app := closedApp()
	desired := []order.DesiredLine{
		{ProductSKU: "participation.patrulje", MemberID: "a", Quantity: 1},
		{ProductSKU: "tshirt.adult", MemberID: "a", Quantity: 1},
	}

	got := app.sellable(&order.Order{}, desired)
	if len(got) != 2 {
		t.Fatalf("sellable() = %v, want the input unchanged", skus(got))
	}
	// Returned as-is rather than copied, so the open-shop path allocates nothing.
	if &got[0] != &desired[0] {
		t.Error("sellable() copied the slice when nothing is closed; it should return the input")
	}
}

func TestSellableHandlesNilOrderAndEmptyInput(t *testing.T) {
	app := closedApp("tshirt.adult")

	// A nil order means "no order yet", which cannot have money in flight, so
	// the filter still applies.
	got := app.sellable(nil, []order.DesiredLine{{ProductSKU: "tshirt.adult", MemberID: "a", Quantity: 1}})
	if len(got) != 0 {
		t.Errorf("sellable(nil, ...) = %v, want the closed line dropped", skus(got))
	}
	if got := app.sellable(nil, nil); got != nil {
		t.Errorf("sellable(nil, nil) = %v, want nil", got)
	}
}

func TestClosedLines(t *testing.T) {
	app := closedApp("tshirt.adult")

	withShirt := &order.Order{Lines: []order.Line{
		{ProductSKU: "participation.klan", MemberID: "a"},
		{ProductSKU: "tshirt.adult", MemberID: "a", Attributes: map[string]any{"size": "l"}},
	}}
	if !app.closedLines(withShirt) {
		t.Error("closedLines() = false, want true for an order carrying a closed SKU")
	}

	participationOnly := &order.Order{Lines: []order.Line{
		{ProductSKU: "participation.klan", MemberID: "a"},
	}}
	if app.closedLines(participationOnly) {
		t.Error("closedLines() = true, want false for a participation-only order")
	}

	if app.closedLines(nil) {
		t.Error("closedLines(nil) = true, want false")
	}
	if closedApp().closedLines(withShirt) {
		t.Error("closedLines() = true with an empty closed set, want false")
	}
}
