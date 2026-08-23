package main

import (
	"testing"

	"github.com/nathejk/shared-go/tables/order"
)

// The invariant: no new payment request may include a product that is closed for
// sale. sellable keeps closed lines off an order in the ordinary case, but an
// order with money in flight is exempt from it and still carries its line — and
// that is the order a save would otherwise turn into a fresh link selling the
// closed product.
func TestChargeableRefusesOrdersHoldingAClosedProduct(t *testing.T) {
	app := closedApp("tshirt.adult")

	o := &order.Order{OrderID: "o-1", TotalAmount: 42500, PaidAmount: 17500, Lines: []order.Line{
		{ProductSKU: "participation.patrulje", MemberID: "a", Quantity: 1, LineTotal: 25000},
		{ProductSKU: "tshirt.adult", MemberID: "a", Quantity: 1, LineTotal: 17500,
			Attributes: map[string]any{"size": "l"}},
	}}

	ok, refusal := app.chargeable(o)
	if ok {
		t.Error("chargeable() = true for an order holding a closed product; it would sell one")
	}
	if refusal == "" {
		t.Error("chargeable() gave no explanation to show the user")
	}
	if refusal != chargeClosedProduct {
		t.Errorf("chargeable() refusal = %q, want %q", refusal, chargeClosedProduct)
	}
}

func TestChargeableAllowsOrdinaryOrders(t *testing.T) {
	app := closedApp("tshirt.adult")

	tests := []struct {
		name string
		o    *order.Order
	}{
		{
			name: "participation only",
			o: &order.Order{OrderID: "o-1", Lines: []order.Line{
				{ProductSKU: "participation.patrulje", MemberID: "a", Quantity: 1},
				{ProductSKU: "participation.patrulje", MemberID: "b", Quantity: 1},
			}},
		},
		{
			// Closing the year shirt must not block payment for anything else the
			// catalogue sells.
			name: "other merchandise",
			o: &order.Order{OrderID: "o-2", Lines: []order.Line{
				{ProductSKU: "participation.klan", MemberID: "a", Quantity: 1},
				{ProductSKU: "mug.enamel", MemberID: "a", Quantity: 1},
			}},
		},
		{
			name: "empty order",
			o:    &order.Order{OrderID: "o-3"},
		},
		{
			name: "nil order",
			o:    nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ok, refusal := app.chargeable(tc.o)
			if !ok {
				t.Errorf("chargeable() = false (%q), want true", refusal)
			}
			if refusal != "" {
				t.Errorf("chargeable() refusal = %q, want empty", refusal)
			}
		})
	}
}

// With nothing closed, an order carrying a t-shirt is chargeable as it always was.
func TestChargeableIgnoresAnEmptyClosedSet(t *testing.T) {
	app := closedApp()
	o := &order.Order{OrderID: "o-1", Lines: []order.Line{
		{ProductSKU: "tshirt.adult", MemberID: "a", Quantity: 1},
	}}
	if ok, refusal := app.chargeable(o); !ok {
		t.Errorf("chargeable() = false (%q) with nothing closed, want true", refusal)
	}
}

// Belt and braces on the structural half of the invariant: once sellable has run
// on an unpaid order, the receipt built from that order has no closed-product row,
// so a charge cannot describe one either.
func TestPaymentReceiptCannotContainAClosedProductAfterFiltering(t *testing.T) {
	app := closedApp("tshirt.adult")

	desired := []order.DesiredLine{
		{ProductSKU: "participation.patrulje", MemberID: "a", Quantity: 1},
		{ProductSKU: "tshirt.adult", MemberID: "a", Quantity: 1, Attributes: map[string]any{"size": "l"}},
	}
	kept := app.sellable(&order.Order{OrderID: "o-1", PaidAmount: 0}, desired)

	// Stand in for what the order projection would hold after those lines were
	// written, and build the wallet receipt from it.
	o := &order.Order{OrderID: "o-1"}
	for _, d := range kept {
		o.Lines = append(o.Lines, order.Line{
			ProductSKU: d.ProductSKU, ProductName: d.ProductSKU, MemberID: d.MemberID,
			UnitPrice: 25000, Quantity: d.Quantity, LineTotal: 25000, Attributes: d.Attributes,
		})
	}

	if ok, _ := app.chargeable(o); !ok {
		t.Fatal("chargeable() = false for an order whose closed lines were already filtered out")
	}
	for _, l := range paymentLinesFromOrder(o) {
		if l.Label == "T-shirt" || l.Label == "tshirt.adult" {
			t.Errorf("payment receipt contains a closed product: %+v", l)
		}
	}
}
