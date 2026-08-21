package main

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/nathejk/shared-go/tables"
	"github.com/nathejk/shared-go/tables/order"
	"github.com/nathejk/shared-go/types"
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

// syncNeeded asks the order entity whether the open order's derived lines
// already say what the desired set (computed off the current owner
// projection — members for klan/patrulje, the person record for
// badut/crew) implies. True means the show handler should call
// SetDerivedLines; false keeps the GET a pure read with no event
// publication.
//
// The comparison deliberately lives in shared-go rather than here. It has
// to apply the same paid-unit offset SetDerivedLines applies, and that
// offset now depends on per-variant paid counts and on catalogue size
// order; a second implementation on this side would drift from the
// command and republish the order's lines on every page load. Commands.
// SyncNeeded and SetDerivedLines share one offset function, so they
// cannot disagree. See shared-go PRD 001 §8.1a (this repo's PRD 002).
//
// An error is logged and reported as "no sync needed": it means the paid
// counts or the catalogue could not be read, in which case SetDerivedLines
// would fail on the same read anyway. Rendering the order as it stands is
// better than writing lines computed from a half-known offset.
func (app *application) syncNeeded(ctx context.Context, o *order.Order, desired []order.DesiredLine) bool {
	need, err := app.commands.Order.SyncNeeded(ctx, o, desired)
	if err != nil {
		log.Printf("SyncNeeded %s: %v", o.OrderID, err)
		return false
	}
	return need
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
