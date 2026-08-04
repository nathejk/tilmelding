package main

import (
	"github.com/nathejk/shared-go/types"
	"nathejk.dk/internal/payment/mobilepay"
	payments "nathejk.dk/nathejk/table/payment"
)

// mobilepayProvider adapts the MobilePay client to the payments.Provider port.
//
// It lives here, in the composition root, rather than in either of the packages
// it joins: payments must not know which provider it is talking to, and the
// mobilepay client must not know it is being used to satisfy a Nathejk port.
// Everything MobilePay-shaped about a payment — the WALLET payment method, the
// WEB_REDIRECT user flow, the aggregate amount fields — is confined to this
// file. Adding a second provider means adding a sibling, not editing a domain
// package.
type mobilepayProvider struct {
	client mobilepay.Client
}

func newMobilepayProvider(client mobilepay.Client) mobilepayProvider {
	return mobilepayProvider{client: client}
}

func (p mobilepayProvider) CreatePayment(req payments.PaymentRequest) (payments.PaymentCreated, error) {
	resp, err := p.client.CreatePayment(req.IdempotencyKey, mobilepay.Payment{
		Amount:             amountTo(req.Amount),
		PaymentMethod:      mobilepay.PaymentMethod{Type: mobilepay.PaymentMethodType("WALLET")},
		Customer:           mobilepay.Customer{PhoneNumber: req.PhoneNumber},
		Reference:          mobilepay.PaymentReference(req.Reference),
		ReturnUrl:          req.CallbackURL,
		UserFlow:           mobilepay.UserFlowWeb,
		PaymentDescription: req.Description,
	})
	if err != nil {
		return payments.PaymentCreated{}, err
	}
	return payments.PaymentCreated{
		Reference:   string(resp.Reference),
		RedirectURL: resp.RedirectUrl,
	}, nil
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
