package main

import (
	"testing"

	"github.com/nathejk/shared-go/tables/order"
	payments "github.com/nathejk/shared-go/tables/payment"
)

// The receipt is the only place the payer sees what they are buying, so the
// labels matter more than usual: two identical "T-shirt" rows tell them
// nothing, which is why the size is folded in — and why the sizes stay separate
// rows while identical products collapse.
func TestPaymentLinesFromOrder(t *testing.T) {
	o := &order.Order{Lines: []order.Line{
		{ProductSKU: "participation.patrulje", ProductName: "Patrulje-deltagelse", UnitPrice: 25000, Quantity: 1, LineTotal: 25000},
		{ProductSKU: "tshirt.adult", ProductName: "T-shirt", UnitPrice: 17500, Quantity: 1, LineTotal: 17500, Attributes: map[string]any{"size": "l"}},
		{ProductSKU: "participation.patrulje", ProductName: "Patrulje-deltagelse", UnitPrice: 25000, Quantity: 1, LineTotal: 25000},
		{ProductSKU: "tshirt.adult", ProductName: "T-shirt", UnitPrice: 17500, Quantity: 1, LineTotal: 17500, Attributes: map[string]any{"size": "xxl"}},
		{ProductSKU: "tshirt.adult", ProductName: "T-shirt", UnitPrice: 17500, Quantity: 1, LineTotal: 17500, Attributes: map[string]any{"size": "l"}},
	}}

	got := paymentLinesFromOrder(o)

	want := []payments.Line{
		{Label: "Patrulje-deltagelse", UnitCount: 2, UnitPrice: 25000, Amount: 50000},
		{Label: "T-shirt (Large)", UnitCount: 2, UnitPrice: 17500, Amount: 35000},
		{Label: "T-shirt (XX-Large)", UnitCount: 1, UnitPrice: 17500, Amount: 17500},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d rows, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("row %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

// Aggregation must not change the total, or a previously reconciled receipt
// stops matching the amount charged and gets dropped.
func TestPaymentLinesFromOrderPreservesTheSum(t *testing.T) {
	var lines []order.Line
	sum := 0
	for i := 0; i < 22; i++ {
		lines = append(lines, order.Line{ProductSKU: "participation.patrulje", ProductName: "Patrulje-deltagelse", UnitPrice: 25000, Quantity: 1, LineTotal: 25000})
		sum += 25000
	}
	for _, size := range []string{"s", "s", "m"} {
		lines = append(lines, order.Line{ProductSKU: "tshirt.adult", ProductName: "T-shirt", UnitPrice: 17500, Quantity: 1, LineTotal: 17500, Attributes: map[string]any{"size": size}})
		sum += 17500
	}

	got := paymentLinesFromOrder(&order.Order{Lines: lines})
	// 25 stored lines collapse to three rows, which is the point.
	if len(got) != 3 {
		t.Errorf("got %d rows from 25 lines, want 3: %+v", len(got), got)
	}
	total := 0
	for _, l := range got {
		total += l.Amount
	}
	if total != sum {
		t.Errorf("aggregated total = %d, want %d", total, sum)
	}
}

// Row order follows the order's own line order, so the same order does not
// produce a differently-ordered receipt on each request.
func TestPaymentLinesFromOrderIsDeterministic(t *testing.T) {
	o := &order.Order{Lines: []order.Line{
		{ProductSKU: "tshirt.adult", ProductName: "T-shirt", Quantity: 1, Attributes: map[string]any{"size": "m"}},
		{ProductSKU: "participation.klan", ProductName: "Senior-deltagelse", Quantity: 1},
		{ProductSKU: "tshirt.adult", ProductName: "T-shirt", Quantity: 1, Attributes: map[string]any{"size": "l"}},
	}}
	want := []string{"T-shirt (Medium)", "Senior-deltagelse", "T-shirt (Large)"}
	for range 20 {
		got := paymentLinesFromOrder(o)
		for i := range want {
			if got[i].Label != want[i] {
				t.Fatalf("row %d = %q, want %q", i, got[i].Label, want[i])
			}
		}
	}
}

// nil rather than an empty slice, so a caller passes "no receipt" instead of
// "a receipt of nothing".
func TestPaymentLinesFromOrderEmptyCases(t *testing.T) {
	if got := paymentLinesFromOrder(nil); got != nil {
		t.Errorf("nil order should give nil lines, got %+v", got)
	}
	if got := paymentLinesFromOrder(&order.Order{}); got != nil {
		t.Errorf("order with no lines should give nil lines, got %+v", got)
	}
}

// A free t-shirt size change is stored as a credit/charge pair. Both halves
// belong on the order — it is the fulfillment record — but neither belongs on
// the receipt, so the pair nets out entirely and there is nothing left to show.
func TestPaymentLinesFromOrderNetsOutAFreeSizeChange(t *testing.T) {
	o := &order.Order{Lines: []order.Line{
		{ProductSKU: "tshirt.adult", ProductName: "T-shirt", UnitPrice: 17500, Quantity: -1, LineTotal: -17500, Attributes: map[string]any{"size": "xxl"}},
		{ProductSKU: "tshirt.adult", ProductName: "T-shirt", UnitPrice: 17500, Quantity: 1, LineTotal: 17500, Attributes: map[string]any{"size": "3xl"}},
	}}

	if got := paymentLinesFromOrder(o); got != nil {
		t.Errorf("a wholly zero-sum order should give no receipt, got %+v", got)
	}
}

// The case that actually reaches a payment provider: one size changed for free
// *and* one genuinely new shirt bought. The credit must cancel a paid-for unit
// rather than the new one's price, leaving a receipt whose sum is still the
// order's total — which is what Charge.linesReconcile checks.
func TestPaymentLinesFromOrderKeepsTheChargeBesideACredit(t *testing.T) {
	o := &order.Order{Lines: []order.Line{
		{ProductSKU: "tshirt.adult", ProductName: "T-shirt", UnitPrice: 17500, Quantity: 1, LineTotal: 17500, Attributes: map[string]any{"size": "m"}},
		{ProductSKU: "tshirt.adult", ProductName: "T-shirt", UnitPrice: 17500, Quantity: -1, LineTotal: -17500, Attributes: map[string]any{"size": "xl"}},
		{ProductSKU: "tshirt.adult", ProductName: "T-shirt", UnitPrice: 17500, Quantity: 1, LineTotal: 17500, Attributes: map[string]any{"size": "3xl"}},
	}}

	got := paymentLinesFromOrder(o)

	if len(got) != 1 {
		t.Fatalf("got %d rows, want 1: %+v", len(got), got)
	}
	if got[0].UnitCount != 1 || got[0].Amount != 17500 {
		t.Errorf("row = %+v, want 1 unit at 17500", got[0])
	}
	for _, l := range got {
		if l.UnitCount < 0 || l.Amount < 0 {
			t.Errorf("receipt still carries a negative row: %+v", l)
		}
	}
	assertReceiptReconciles(t, o, got)
}

// A credit may only cancel units of its own product. A reclaimed t-shirt must
// never discount a participation seat.
func TestPaymentLinesFromOrderCreditDoesNotCrossSKUs(t *testing.T) {
	o := &order.Order{Lines: []order.Line{
		{ProductSKU: "participation.patrulje", ProductName: "Patrulje-deltagelse", UnitPrice: 25000, Quantity: 1, LineTotal: 25000},
		{ProductSKU: "tshirt.adult", ProductName: "T-shirt", UnitPrice: 17500, Quantity: 1, LineTotal: 17500, Attributes: map[string]any{"size": "m"}},
		{ProductSKU: "tshirt.adult", ProductName: "T-shirt", UnitPrice: 17500, Quantity: -1, LineTotal: -17500, Attributes: map[string]any{"size": "xl"}},
	}}

	got := paymentLinesFromOrder(o)

	want := []payments.Line{{Label: "Patrulje-deltagelse", UnitCount: 1, UnitPrice: 25000, Amount: 25000}}
	if len(got) != len(want) || got[0] != want[0] {
		t.Errorf("got %+v, want %+v", got, want)
	}
	assertReceiptReconciles(t, o, got)
}

// An unpaired credit means the pairing invariant was broken upstream. Rather
// than quietly shipping a receipt that disagrees with the amount charged, the
// rows are passed through untouched: the provider rejecting a negative line is
// a failure someone can see.
func TestPaymentLinesFromOrderPassesThroughAnUnpairedCredit(t *testing.T) {
	o := &order.Order{Lines: []order.Line{
		{ProductSKU: "tshirt.adult", ProductName: "T-shirt", UnitPrice: 17500, Quantity: -1, LineTotal: -17500, Attributes: map[string]any{"size": "xl"}},
	}}

	got := paymentLinesFromOrder(o)

	if len(got) != 1 {
		t.Fatalf("got %d rows, want the unfiltered 1: %+v", len(got), got)
	}
	assertReceiptReconciles(t, o, got)
}

// assertReceiptReconciles is the invariant netting must never break: the rows
// sum to the order's total, so a receipt that reconciled before still does.
func assertReceiptReconciles(t *testing.T, o *order.Order, lines []payments.Line) {
	t.Helper()
	total := 0
	for _, l := range o.Lines {
		total += l.LineTotal
	}
	sum := 0
	for _, l := range lines {
		sum += l.Amount
	}
	if sum != total {
		t.Errorf("receipt sums to %d, order total is %d", sum, total)
	}
}

func TestReceiptLabelSizeHandling(t *testing.T) {
	for _, tc := range []struct {
		name string
		line order.Line
		want string
	}{
		{
			name: "no attributes at all",
			line: order.Line{ProductName: "Senior-deltagelse"},
			want: "Senior-deltagelse",
		},
		{
			name: "attributes without a size",
			line: order.Line{ProductName: "T-shirt", Attributes: map[string]any{"colour": "red"}},
			want: "T-shirt",
		},
		{
			name: "empty size is not rendered as ()",
			line: order.Line{ProductName: "T-shirt", Attributes: map[string]any{"size": ""}},
			want: "T-shirt",
		},
		{
			// A size the label map doesn't know still reads sensibly rather
			// than vanishing, so adding a catalogue size cannot silently
			// produce an unlabelled line.
			name: "unknown size falls back to the slug",
			line: order.Line{ProductName: "T-shirt", Attributes: map[string]any{"size": "4xl"}},
			want: "T-shirt (4xl)",
		},
		{
			// The attribute map is untyped on the wire; a non-string must not
			// panic or produce "%!s(int=42)".
			name: "non-string size is ignored",
			line: order.Line{ProductName: "T-shirt", Attributes: map[string]any{"size": 42}},
			want: "T-shirt",
		},
	} {
		if got := receiptLabel(tc.line); got != tc.want {
			t.Errorf("%s: receiptLabel() = %q, want %q", tc.name, got, tc.want)
		}
	}
}
