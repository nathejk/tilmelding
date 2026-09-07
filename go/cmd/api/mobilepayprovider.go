package main

import (
	"strconv"
	"strings"

	payments "github.com/nathejk/shared-go/tables/payment"
	"github.com/nathejk/shared-go/types"
	"nathejk.dk/internal/payment/mobilepay"
)

// mobilepayProvider adapts the MobilePay client to the payments.Provider port.
//
// It lives here, in the composition root, rather than in either of the packages
// it joins: payments must not know which provider it is talking to, and the
// mobilepay client must not know it is being used to satisfy a Nathejk port.
// Everything MobilePay-shaped about a payment — the WALLET payment method, the
// WEB_REDIRECT user flow, the aggregate amount fields, and the callback route
// the payer is returned to — is confined to this file. Adding a second provider
// means adding a sibling, not editing a domain package.
type mobilepayProvider struct {
	client  mobilepay.Client
	baseURL string
}

// newMobilepayProvider builds the adapter. baseURL is the public host of this
// deployment (cfg.baseurl); the MobilePay callback route is derived from it, so
// a non-production deployment returns the payer to itself rather than to
// production. It is stored without a trailing slash.
func newMobilepayProvider(client mobilepay.Client, baseURL string) mobilepayProvider {
	return mobilepayProvider{client: client, baseURL: strings.TrimRight(baseURL, "/")}
}

func (p mobilepayProvider) CreatePayment(req payments.PaymentRequest) (payments.PaymentCreated, error) {
	resp, err := p.client.CreatePayment(req.IdempotencyKey, mobilepay.Payment{
		Amount:        amountTo(req.Amount),
		PaymentMethod: mobilepay.PaymentMethod{Type: mobilepay.PaymentMethodTypeWallet},
		Customer:      mobilepay.Customer{PhoneNumber: req.PhoneNumber},
		Reference:     mobilepay.PaymentReference(req.Reference),
		// Where MobilePay returns the payer after they approve/reject. This is
		// the /callback/mobilepay/:ref route (routes.go), which drives Capture.
		ReturnUrl:          p.baseURL + "/callback/mobilepay/" + req.Reference,
		UserFlow:           mobilepay.UserFlowWeb,
		PaymentDescription: req.Description,
		Receipt:            receiptFor(req),
	})
	if err != nil {
		return payments.PaymentCreated{}, err
	}
	return payments.PaymentCreated{
		Reference:   string(resp.Reference),
		RedirectURL: resp.RedirectUrl,
	}, nil
}

// receiptFor renders the payer-visible receipt, which MobilePay shows in the
// wallet alongside the amount. Previously never populated, so a payer saw only
// "Nathejk tilmelding" and a total.
//
// The bottom line always carries the currency; order lines are added only when
// the entity supplied a reconciled set (Request drops one that does not sum to
// the charge). Amounts pass through unchanged — both sides are in minor units.
//
// Tax is reported as zero throughout: Nathejk charges participation fees, not
// VAT-bearing sales, so there is no tax component to split out. TotalAmount and
// TotalAmountExcludingTax are therefore the same number, which is what a
// zero-rated line looks like.
func receiptFor(req payments.PaymentRequest) mobilepay.Receipt {
	receipt := mobilepay.Receipt{
		BottomLine: mobilepay.BottomLine{Currency: mobilepay.Currency(req.Amount.Currency)},
	}
	if len(req.Lines) == 0 {
		return receipt
	}
	receipt.OrderLines = make([]mobilepay.OrderLine, 0, len(req.Lines))
	for i, l := range req.Lines {
		line := mobilepay.OrderLine{
			// MobilePay requires an id per line; the payer never sees it, so
			// position is enough and keeps it stable for a given receipt.
			ID:                      strconv.Itoa(i + 1),
			Name:                    l.Label,
			TotalAmount:             int64(l.Amount),
			TotalAmountExcludingTax: int64(l.Amount),
		}
		line.UnitInfo.UnitPrice = int64(l.UnitPrice)
		line.UnitInfo.Quantity = strconv.Itoa(l.UnitCount)
		line.UnitInfo.QuantityUnit = "PCS"
		receipt.OrderLines = append(receipt.OrderLines, line)
	}
	return receipt
}

func (p mobilepayProvider) GetAuthorization(reference string) (payments.Authorization, error) {
	mpp, err := p.client.GetPayment(mobilepay.PaymentReference(reference))
	if err != nil {
		return payments.Authorization{}, err
	}
	return payments.Authorization{
		Authorized: mpp.State == mobilepay.PaymentStateAuthorized,
		// The currency of the payment as a whole; the aggregate amounts are
		// denominated in it.
		Currency:         types.Currency(mpp.Amount.Currency),
		AuthorizedAmount: mpp.Aggregate.AuthorizedAmount.Value,
		CapturedAmount:   mpp.Aggregate.CapturedAmount.Value,
	}, nil
}

func (p mobilepayProvider) CapturePayment(reference string, amount payments.Amount) error {
	_, err := p.client.CapturePayment(mobilepay.PaymentReference(reference), amountTo(amount))
	return err
}

func amountTo(a payments.Amount) mobilepay.Amount {
	return mobilepay.Amount{
		Currency: mobilepay.Currency(a.Currency),
		Value:    a.Value,
	}
}

var _ payments.Provider = mobilepayProvider{}
