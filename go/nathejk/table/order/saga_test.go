package order

import (
	"context"
	"testing"
	"time"

	"github.com/jrgensen/cqrs"
	"github.com/jrgensen/cqrs/cqrstest"
	"github.com/nathejk/shared-go/messages"
	"github.com/nathejk/shared-go/types"

	tables "nathejk.dk/nathejk/table"
	"nathejk.dk/nathejk/table/payment"
)

// sagaFakeQueries is a Queries whose GetByID the test controls. Only GetByID is
// exercised by the saga; the rest satisfy the interface and are never called.
type sagaFakeQueries struct {
	order *Order
	err   error
}

func (f sagaFakeQueries) GetByID(context.Context, string) (*Order, error) {
	return f.order, f.err
}
func (sagaFakeQueries) FindOpenOrder(context.Context, types.YearSlug, types.TeamType, string) (*Order, error) {
	return nil, tables.ErrRecordNotFound
}
func (sagaFakeQueries) ListByOwner(context.Context, types.YearSlug, types.TeamType, string) ([]Order, error) {
	return nil, nil
}
func (sagaFakeQueries) ReservedQuantity(context.Context, types.YearSlug, string) (int, error) {
	return 0, nil
}
func (sagaFakeQueries) PaidQuantityBySKU(context.Context, types.YearSlug, types.TeamType, string) (map[string]int, error) {
	return nil, nil
}

type sagaFakePayments struct{ pmt *payment.Payment }

func (f sagaFakePayments) GetByReference(string) (*payment.Payment, error) { return f.pmt, nil }

// receivedMsg builds a payment.received message the way production does, so
// HandleMessage decodes the same body it would off the wire.
func receivedMsg(t *testing.T, reference string) cqrs.Message {
	t.Helper()
	m := cqrstest.NewMessage(cqrs.SubjectFromStr("NATHEJK:2026.payment." + reference + ".received"))
	if err := m.SetBody(&messages.NathejkPaymentReceived{Reference: reference}); err != nil {
		t.Fatalf("set body: %v", err)
	}
	return m
}

// newTestSaga wires a saga with fakes and a recording sleep seam.
func newTestSaga(order *Order) (*saga, *cqrstest.Publisher, *[]time.Duration) {
	pub := &cqrstest.Publisher{}
	var slept []time.Duration
	s := &saga{
		p:        pub,
		q:        sagaFakeQueries{order: order},
		payments: sagaFakePayments{pmt: &payment.Payment{OrderForeignKey: "order-1"}},
		settle:   2 * time.Second,
		sleep:    func(d time.Duration) { slept = append(slept, d) },
	}
	return s, pub, &slept
}

func openPaidOrder() *Order {
	return &Order{OrderID: "order-1", Year: "2026", Status: StatusOpen, TotalAmount: 45000, PaidAmount: 45000}
}

// The whole point of task 006: the saga must expose CaughtUp() so the jetstream
// layer (which discovers it by runtime type assertion) can call it.
func TestSagaImplementsCatchupListener(t *testing.T) {
	var c cqrs.Consumer = NewSaga(&cqrstest.Publisher{}, sagaFakeQueries{}, sagaFakePayments{}, 0)
	if _, ok := c.(interface{ CaughtUp() }); !ok {
		t.Fatal("saga must implement CaughtUp() so replay catch-up can be signalled")
	}
}

func TestSagaSkipsSettleDuringReplay(t *testing.T) {
	s, pub, slept := newTestSaga(openPaidOrder())

	// Not yet live (replaying): the settle delay must be skipped entirely.
	if err := s.HandleMessage(receivedMsg(t, "ref-1")); err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if len(*slept) != 0 {
		t.Errorf("no settle sleep expected during replay, got %v", *slept)
	}
	// The transition still happens during replay — only the wait is skipped.
	if len(pub.Messages) != 1 {
		t.Fatalf("want the order.paid event, got %d messages", len(pub.Messages))
	}
}

func TestSagaWaitsSettleWhenLive(t *testing.T) {
	s, _, slept := newTestSaga(openPaidOrder())

	s.CaughtUp()
	if err := s.HandleMessage(receivedMsg(t, "ref-1")); err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if len(*slept) != 1 || (*slept)[0] != 2*time.Second {
		t.Errorf("live path should wait one settle interval, got %v", *slept)
	}
}

func TestSagaTransitionsFullyPaidOpenOrder(t *testing.T) {
	s, pub, _ := newTestSaga(openPaidOrder())
	s.CaughtUp()

	if err := s.HandleMessage(receivedMsg(t, "ref-1")); err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if len(pub.Messages) != 1 {
		t.Fatalf("want 1 order.paid event, got %d", len(pub.Messages))
	}
	if got := pub.Subjects()[0]; got != "NATHEJK.2026.order.order-1.paid" {
		t.Errorf("subject = %q, want the order paid subject", got)
	}
	var body messages.NathejkOrderPaid
	if err := pub.Messages[0].Body(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.OrderID != "order-1" || body.PaidAmount != 45000 {
		t.Errorf("unexpected paid body %+v", body)
	}
}

func TestSagaNoTransitionWhenNotFullyPaidOrNotOpen(t *testing.T) {
	for _, tc := range []struct {
		name  string
		order *Order
	}{
		{"under-paid", &Order{OrderID: "order-1", Status: StatusOpen, TotalAmount: 45000, PaidAmount: 20000}},
		{"already paid", &Order{OrderID: "order-1", Status: StatusPaid, TotalAmount: 45000, PaidAmount: 45000}},
		{"zero total", &Order{OrderID: "order-1", Status: StatusOpen, TotalAmount: 0, PaidAmount: 0}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, pub, _ := newTestSaga(tc.order)
			s.CaughtUp()
			if err := s.HandleMessage(receivedMsg(t, "ref-1")); err != nil {
				t.Fatalf("HandleMessage: %v", err)
			}
			if len(pub.Messages) != 0 {
				t.Errorf("no transition expected, got %v", pub.Subjects())
			}
		})
	}
}
