package main

import (
	"testing"

	payments "github.com/nathejk/shared-go/tables/payment"
	"nathejk.dk/internal/payment/mobilepay"
)

// fakeMobilepayClient captures the Payment passed to CreatePayment so the test
// can assert on the callback URL the adapter builds.
type fakeMobilepayClient struct{ got mobilepay.Payment }

func (f *fakeMobilepayClient) CreatePayment(key string, p mobilepay.Payment) (*mobilepay.CreatePaymentResponse, error) {
	f.got = p
	return &mobilepay.CreatePaymentResponse{Reference: p.Reference, RedirectUrl: "https://mp/redirect"}, nil
}
func (f *fakeMobilepayClient) GetPayment(mobilepay.PaymentReference) (*mobilepay.GetPaymentResponse, error) {
	return &mobilepay.GetPaymentResponse{}, nil
}
func (f *fakeMobilepayClient) CapturePayment(mobilepay.PaymentReference, mobilepay.Amount) (*mobilepay.ModificationResponse, error) {
	return &mobilepay.ModificationResponse{}, nil
}

// The callback the payer is returned to must derive from this deployment's
// baseURL, not a hard-coded production host — task 024. Otherwise a staging
// deployment sends the payer to production and its capture never runs.
func TestMobilepayProviderCallbackDerivesFromBaseURL(t *testing.T) {
	for _, tc := range []struct {
		name    string
		baseURL string
		want    string
	}{
		{"production", "https://tilmelding.nathejk.dk", "https://tilmelding.nathejk.dk/callback/mobilepay/ref-1"},
		{"staging", "https://staging.example.com", "https://staging.example.com/callback/mobilepay/ref-1"},
		{"trailing slash trimmed", "https://staging.example.com/", "https://staging.example.com/callback/mobilepay/ref-1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := &fakeMobilepayClient{}
			p := newMobilepayProvider(client, tc.baseURL)

			if _, err := p.CreatePayment(payments.PaymentRequest{Reference: "ref-1"}); err != nil {
				t.Fatalf("CreatePayment: %v", err)
			}
			if got := client.got.ReturnUrl; got != tc.want {
				t.Errorf("callback ReturnUrl = %q, want %q", got, tc.want)
			}
		})
	}
}

var _ mobilepay.Client = (*fakeMobilepayClient)(nil)
