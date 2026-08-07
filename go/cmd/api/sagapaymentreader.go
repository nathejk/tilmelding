package main

import (
	"context"

	sharedpayment "github.com/nathejk/shared-go/tables/payment"
	payments "nathejk.dk/nathejk/table/payment"
)

// sagaPaymentReader lets the shared order saga read the consolidated payment
// entity.
//
// It exists for exactly one reason: shared-go's order.PaymentReader is declared
// as
//
//	GetByReference(reference string) (*payment.Payment, error)
//
// naming shared-go's own payment.Payment. Until the consolidated entity lands in
// shared-go and the order package is repointed at it, the two packages describe
// the same rows with two unrelated Go types, and nothing but a conversion can
// bridge that.
//
// Delete this file when that happens; the entity satisfies the interface
// directly once both sides agree on the type.
//
// The saga reads only OrderForeignKey — it uses that to look up the order and
// then works entirely from the order's own totals (see attemptTransition in
// shared-go/tables/order/saga.go). The rest of the fields are mapped anyway so
// that a future change there cannot silently observe a zero value.
type sagaPaymentReader struct {
	q payments.Queries
}

func (a sagaPaymentReader) GetByReference(reference string) (*sharedpayment.Payment, error) {
	p, err := a.q.GetByReference(context.Background(), reference)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, nil
	}
	return &sharedpayment.Payment{
		Reference:    p.Reference,
		Year:         p.Year,
		ReceiptEmail: p.ReceiptEmail,
		ReturnUrl:    p.ReturnUrl,
		Currency:     p.Currency,
		// Note: minor units. shared-go's own query divided this by 100 before
		// returning it; the saga never read it, so nothing depended on the
		// difference. Anything that starts reading it must expect øre.
		Amount:          p.Amount,
		Method:          p.Method,
		Status:          p.Status,
		CreatedAt:       p.CreatedAt,
		ChangedAt:       p.ChangedAt,
		OrderForeignKey: p.OrderForeignKey,
		OrderType:       p.OrderType,
		// Operations has no counterpart in the shared struct and is dropped.
	}, nil
}
