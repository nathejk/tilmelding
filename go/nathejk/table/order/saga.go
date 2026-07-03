package order

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/nathejk/shared-go/messages"
	tables "nathejk.dk/nathejk/table"
	"nathejk.dk/nathejk/table/payment"
	"nathejk.dk/superfluids/streaminterface"
)

// PaymentReader is the small slice of the payment read API that the order
// saga consumes. Declared as an interface so this package doesn't take a
// hard dependency on the payment package's concrete value type — the
// existing *payment.Query already satisfies it.
type PaymentReader interface {
	GetByReference(reference string) (*payment.Payment, error)
}

// DefaultSagaSettle is the brief delay the saga waits between receiving a
// payment event and reading the (eventually consistent) projection. The
// payment projector and the saga subscribe to the same NATS subject and
// run in independent goroutines; in practice the projection lands within
// milliseconds, but we leave a generous default to match the existing
// 2-second wait in mobilepayCallbackHandler.
const DefaultSagaSettle = 2 * time.Second

// saga listens for payment events and transitions the corresponding order
// to status="paid" once cumulative payments cover its total. This is the
// only path by which orders reach StatusPaid; once there, the
// SetDerivedLines / AddManualLine / RemoveLine / Cancel commands all
// reject mutations with ErrNotOpen, giving the immutability guarantee
// users asked for.
//
// The saga is idempotent at multiple layers:
//
//   - It only emits NathejkOrderPaid when the order is currently
//     StatusOpen (a paid order returns early).
//   - The projector's handlePaid uses WHERE status='open' so a replayed
//     event is a no-op.
//
// It is *not* perfectly race-free: a partial-refund flow could land an
// order back below its total after status=paid. That isn't on the
// roadmap — flag for revisit if it ever is.
type saga struct {
	p        streaminterface.Publisher
	q        Queries
	payments PaymentReader
	settle   time.Duration
}

// NewSaga wires the payment->order paid saga. Pass the order Queries
// (typically the *table returned by order.New), a PaymentReader (typically
// the *table returned by table.NewPayment), and the JetStream publisher.
// settle lets you tune or zero out the projection-catchup delay; pass 0
// to fall back to DefaultSagaSettle.
func NewSaga(p streaminterface.Publisher, q Queries, payments PaymentReader, settle time.Duration) streaminterface.Consumer {
	if settle <= 0 {
		settle = DefaultSagaSettle
	}
	return &saga{p: p, q: q, payments: payments, settle: settle}
}

func (s *saga) Consumes() []streaminterface.Subject {
	// Subscribing to .received only is sufficient: by that point the
	// payment.received event has already been published, the prior
	// .reserved row carries the same amount, and the order's joined
	// paidAmount sums both reserved and received states. .reserved would
	// trigger the same transition slightly earlier, but at the cost of
	// firing twice per payment for no benefit.
	return []streaminterface.Subject{
		streaminterface.SubjectFromStr("NATHEJK:*.payment.*.received"),
	}
}

func (s *saga) HandleMessage(msg streaminterface.Message) error {
	var body messages.NathejkPaymentReceived
	if err := msg.Body(&body); err != nil {
		return err
	}
	if body.Reference == "" {
		return nil
	}

	// Wait briefly so the payment projector can update payment.status to
	// 'received'. The order's joined paidAmount only counts payments in
	// {'reserved','received'}; if we read before the projector has
	// caught up we may see status='requested' and miss the just-received
	// amount.
	time.Sleep(s.settle)

	pmt, err := s.payments.GetByReference(body.Reference)
	if err != nil {
		// Reference not found / not interesting; nothing to do.
		return nil
	}
	if pmt == nil || pmt.OrderForeignKey == "" {
		return nil
	}

	// Legacy payments use the team/user ID as OrderForeignKey rather
	// than an order ID. GetByID will return ErrRecordNotFound for those —
	// silently skip; the saga is a no-op for the legacy flow by design.
	o, err := s.q.GetByID(context.Background(), pmt.OrderForeignKey)
	if err != nil {
		if errors.Is(err, tables.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	if o.Status != StatusOpen {
		return nil
	}
	// Only transition when the order has a positive total fully covered.
	// A free order (TotalAmount == 0) shouldn't auto-transition on a
	// random payment hitting it — it'd never be in this code path
	// without a positive payment, but guard anyway.
	if o.TotalAmount <= 0 || o.PaidAmount < o.TotalAmount {
		return nil
	}

	paid := messages.NathejkOrderPaid{
		OrderID:    o.OrderID,
		PaidAmount: o.PaidAmount,
		Timestamp:  time.Now(),
	}
	subj := streaminterface.SubjectFromStr(fmt.Sprintf("NATHEJK:%s.order.%s.paid", o.Year, o.OrderID))
	out := s.p.MessageFunc()(subj)
	out.SetBody(&paid)
	out.SetMeta(&messages.Metadata{Producer: "tilmelding-api"})
	if err := s.p.Publish(out); err != nil {
		log.Printf("order saga: publish paid for %s: %v", o.OrderID, err)
		return err
	}
	return nil
}
