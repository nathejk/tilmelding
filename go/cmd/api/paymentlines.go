package main

import (
	"github.com/nathejk/shared-go/tables/order"
	payments "nathejk.dk/nathejk/table/payment"
)

// paymentLinesFromOrder projects an order's lines onto the receipt the payer
// sees in their wallet.
//
// This mapping lives in the composition root, not in either entity, for the same
// reason mobilepayProvider does: shared-go's order package imports payment (the
// saga's PaymentReader), so payment cannot import order without closing an import
// cycle. Each side describes what it needs and this file joins them.
//
// Lines are aggregated, not passed through one per member. An order is stored
// with one derived line per member per product — the largest live order has 43 —
// and a wallet receipt listing "Patrulje-deltagelse" twenty-two times tells the
// payer nothing they cannot get from the total. UnitCount exists for exactly
// this. The grouping deliberately matches aggregateOrderLines in
// vue/src/helpers/order.js, so the receipt shows the same rows as the page the
// payer just came from.
//
// Aggregation preserves the sum, so a reconciled set stays reconciled — see
// Charge.linesReconcile.
//
// Returns nil for a nil or empty order rather than an empty slice, so callers
// pass "no receipt" rather than "a receipt of nothing".
func paymentLinesFromOrder(o *order.Order) []payments.Line {
	if o == nil || len(o.Lines) == 0 {
		return nil
	}

	// Insertion-ordered grouping: the receipt should list products in the order
	// the order itself does, not in map-iteration order, or two runs of the same
	// order produce different receipts.
	var keys []string
	byKey := map[string]*payments.Line{}

	for _, l := range o.Lines {
		key := l.ProductSKU
		if size := lineSize(l); size != "" {
			key += "|" + size
		}
		if existing, ok := byKey[key]; ok {
			existing.UnitCount += l.Quantity
			existing.Amount += l.LineTotal
			continue
		}
		keys = append(keys, key)
		byKey[key] = &payments.Line{
			Label:     receiptLabel(l),
			UnitCount: l.Quantity,
			UnitPrice: l.UnitPrice,
			Amount:    l.LineTotal,
		}
	}

	lines := make([]payments.Line, 0, len(keys))
	for _, k := range keys {
		lines = append(lines, *byKey[k])
	}
	return lines
}

// lineSize reads the t-shirt size off a line's attributes. The map is untyped on
// the wire, so a non-string value is treated as absent rather than formatted
// into the label.
func lineSize(l order.Line) string {
	size, _ := l.Attributes["size"].(string)
	return size
}

// receiptLabel is what the payer reads for one line.
//
// T-shirts carry their size, because a receipt with two "T-shirt" rows tells the
// payer nothing about what they bought — the size is the only thing
// distinguishing them, and it is the field most likely to be queried later ("I
// ordered a large"). It is also what keeps the sizes as separate receipt rows:
// the grouping key includes the size, so the label has to as well or two rows
// would read identically.
//
// The size is rendered with the same human label the size dropdown offers
// (tshirtSizeLabels), so the payer sees the word they picked. Unknown sizes fall
// back to the slug, so adding a catalogue size cannot produce an unlabelled row.
func receiptLabel(l order.Line) string {
	size := lineSize(l)
	if size == "" {
		return l.ProductName
	}
	label, ok := tshirtSizeLabels[size]
	if !ok {
		label = size
	}
	return l.ProductName + " (" + label + ")"
}
