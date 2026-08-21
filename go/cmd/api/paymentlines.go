package main

import (
	"github.com/nathejk/shared-go/tables/order"
	payments "github.com/nathejk/shared-go/tables/payment"
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
// Zero-sum credit/charge pairs are netted out before the receipt is built — see
// netOutCredits.
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
	byKey := map[string]*receiptGroup{}

	for _, l := range o.Lines {
		key := l.ProductSKU
		if size := lineSize(l); size != "" {
			key += "|" + size
		}
		if existing, ok := byKey[key]; ok {
			existing.line.UnitCount += l.Quantity
			existing.line.Amount += l.LineTotal
			continue
		}
		keys = append(keys, key)
		byKey[key] = &receiptGroup{
			sku: l.ProductSKU,
			line: payments.Line{
				Label:     receiptLabel(l),
				UnitCount: l.Quantity,
				UnitPrice: l.UnitPrice,
				Amount:    l.LineTotal,
			},
		}
	}

	groups := make([]receiptGroup, 0, len(keys))
	for _, k := range keys {
		groups = append(groups, *byKey[k])
	}
	return netOutCredits(groups)
}

// receiptGroup is one aggregated receipt row plus the SKU it came from. The SKU
// is not on payments.Line (a receipt row is just words and numbers to the payer)
// but netOutCredits needs it: a credit may only cancel units of the same product.
type receiptGroup struct {
	sku  string
	line payments.Line
}

// netOutCredits removes zero-sum credit/charge pairs from the receipt.
//
// A free t-shirt size change is recorded on the open order as a pair of derived
// lines — one negative for the size handed back, one positive for the size now
// wanted (see shared-go PRD 001). Both belong on the order, which is the
// fulfillment record. Neither belongs on the receipt: the payment provider's
// line-item API is not expected to accept a negative quantity, and a payer shown
// a credit they will never receive is being misinformed.
//
// The pair sums to zero, so dropping *both* halves leaves the total untouched and
// a reconciled set stays reconciled. Dropping only one half would not, which is
// why the cancellation is by unit count per SKU rather than by simply filtering
// negative rows: for each credited unit, one paid-for unit of the same product is
// removed too. Both halves carry the same catalogue unit price, so the amounts
// cancel exactly.
//
// Which positive row absorbs a credit is not recoverable from the aggregated rows
// — a credit records the size returned, not the charge it was paired with — so
// rows are consumed in receipt order. Any choice yields the same total; only the
// surviving row's size label differs.
//
// If credits remain unmatched after every positive row of that SKU is exhausted,
// the pairing invariant has been broken upstream. The receipt is then returned
// unfiltered: a receipt the provider may reject is a visible failure, while one
// that silently disagrees with the amount charged is not.
func netOutCredits(groups []receiptGroup) []payments.Line {
	cancel := map[string]int{}
	for _, g := range groups {
		if g.line.UnitCount < 0 {
			cancel[g.sku] -= g.line.UnitCount
		}
	}
	if len(cancel) == 0 {
		return receiptLines(groups)
	}

	kept := make([]receiptGroup, 0, len(groups))
	for _, g := range groups {
		if g.line.UnitCount < 0 {
			continue // the credit half of a pair
		}
		if n := cancel[g.sku]; n > 0 {
			take := min(n, g.line.UnitCount)
			cancel[g.sku] -= take
			g.line.UnitCount -= take
			g.line.Amount -= take * g.line.UnitPrice
			if g.line.UnitCount == 0 {
				continue
			}
		}
		kept = append(kept, g)
	}

	for _, n := range cancel {
		if n > 0 {
			return receiptLines(groups) // unpaired credit; keep the sum honest
		}
	}
	if len(kept) == 0 {
		return nil
	}
	return receiptLines(kept)
}

func receiptLines(groups []receiptGroup) []payments.Line {
	if len(groups) == 0 {
		return nil
	}
	lines := make([]payments.Line, 0, len(groups))
	for _, g := range groups {
		lines = append(lines, g.line)
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
