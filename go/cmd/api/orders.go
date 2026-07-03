package main

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/nathejk/shared-go/messages"
	"github.com/nathejk/shared-go/types"
	tables "nathejk.dk/nathejk/table"
	"nathejk.dk/nathejk/table/order"
)

// loadOrders fetches the "current" (open) order plus the list of paid
// orders for the given owner, used to populate the show endpoints'
// response envelope (`"order"` + `"paidOrders"`).
//
// The split mirrors the UX: the open order is the editable cart, while
// paidOrders is a historical, read-only list of completed transactions.
// Cancelled orders are intentionally excluded — they're terminal but not
// useful in the user's payment history.
//
// Errors are logged and treated as empty results rather than failing the
// whole request: the show endpoints already render gracefully when an
// order is missing, and a partial payment history is preferable to a 500.
func (app *application) loadOrders(ctx context.Context, ownerType types.TeamType, ownerID string) (*order.Order, []order.Order) {
	open, err := app.models.Order.FindOpenOrder(ctx, app.config.year, ownerType, ownerID)
	if err != nil && !errors.Is(err, tables.ErrRecordNotFound) {
		log.Printf("FindOpenOrder %q", err)
	}

	all, err := app.models.Order.ListByOwner(ctx, app.config.year, ownerType, ownerID)
	if err != nil {
		log.Printf("ListByOwner %q", err)
		return open, []order.Order{}
	}

	paid := make([]order.Order, 0, len(all))
	for _, o := range all {
		if o.Status == order.StatusPaid {
			paid = append(paid, o)
		}
	}
	return open, paid
}

// derivedLinesNeedSync reports whether the open order's derived lines
// diverge from the desired set computed off the current owner projection
// (members for klan/patrulje, the person record for badut/crew). Returns
// true when the show handler should call SetDerivedLines to bring them
// back into agreement, false when the order already matches — in which
// case the GET stays a pure read with no event publication.
//
// The comparison ignores manual lines (only derived lines are recomputed)
// and is keyed on (productSku, memberId, t-shirt size) — the same
// dimensions every derivedLinesFor* helper varies on. Quantity drift would
// not be detected, but the read-path helpers always emit quantity=1 so
// any difference there indicates a manual edit we shouldn't clobber.
func derivedLinesNeedSync(o *order.Order, desired []order.DesiredLine) bool {
	type key struct {
		sku      string
		memberID string
		size     string
	}
	current := map[key]bool{}
	for _, l := range o.Lines {
		if l.Origin != string(messages.LineOriginDerived) {
			continue
		}
		size, _ := l.Attributes["size"].(string)
		current[key{l.ProductSKU, l.MemberID, size}] = true
	}
	want := map[key]bool{}
	for _, d := range desired {
		size, _ := d.Attributes["size"].(string)
		want[key{d.ProductSKU, d.MemberID, size}] = true
	}
	if len(current) != len(want) {
		return true
	}
	for k := range want {
		if !current[k] {
			return true
		}
	}
	return false
}

// setDerivedLinesAfterCreate wraps Order.SetDerivedLines with a bounded
// retry on tables.ErrRecordNotFound. EnsureOpenOrder publishes
// NathejkOrderCreated through NATS asynchronously; the projector
// typically catches up in a few milliseconds, but on the very first GET
// after a brand-new order is created, SetDerivedLines (which reads the
// order via GetByID) can race the projector and return ErrRecordNotFound.
//
// The retry budget is generous enough to absorb a cold-start projector
// (~500ms total wall time) but bounded so a genuinely missing order does
// not hang the request. Errors other than ErrRecordNotFound are returned
// immediately — only the projection-lag race is retried.
//
// This mirrors the time.Sleep(s.settle) pattern in the order saga: both
// are read-after-write reconciliations against an eventually-consistent
// projection.
func (app *application) setDerivedLinesAfterCreate(ctx context.Context, orderID string, desired []order.DesiredLine) (*order.Order, error) {
	const (
		attempts = 10
		backoff  = 50 * time.Millisecond
	)
	var lastErr error
	for i := 0; i < attempts; i++ {
		o, err := app.commands.Order.SetDerivedLines(ctx, orderID, desired)
		if err == nil {
			return o, nil
		}
		if !errors.Is(err, tables.ErrRecordNotFound) {
			return nil, err
		}
		lastErr = err
		time.Sleep(backoff)
	}
	return nil, lastErr
}
