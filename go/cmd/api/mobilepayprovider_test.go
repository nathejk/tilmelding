package main

import (
	"testing"

	"github.com/nathejk/shared-go/types"
	"nathejk.dk/internal/payment/mobilepay"
	payments "nathejk.dk/nathejk/table/payment"
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

// The receipt is what the payer sees in the wallet. It was never populated
// before, so they got a description and a total and nothing else.
func TestMobilepayProviderBuildsReceipt(t *testing.T) {
	client := &fakeMobilepayClient{}
	p := newMobilepayProvider(client, "https://tilmelding.nathejk.dk")

	_, err := p.CreatePayment(payments.PaymentRequest{
		Reference: "ref-1",
		Amount:    payments.Amount{Currency: types.CurrencyDKK, Value: 42500},
		Lines: []payments.Line{
			{Label: "Patrulje-deltagelse", UnitCount: 1, UnitPrice: 25000, Amount: 25000},
			{Label: "T-shirt (Large)", UnitCount: 1, UnitPrice: 17500, Amount: 17500},
		},
	})
	if err != nil {
		t.Fatalf("CreatePayment: %v", err)
	}

	got := client.got.Receipt
	if got.BottomLine.Currency != mobilepay.CurrencyDKK {
		t.Errorf("bottom line currency = %q, want DKK", got.BottomLine.Currency)
	}
	if len(got.OrderLines) != 2 {
		t.Fatalf("got %d order lines, want 2", len(got.OrderLines))
	}

	// The lines must sum to the amount charged, or MobilePay may reject the
	// receipt. This is the assertion that protects that invariant end to end.
	var sum int64
	for _, l := range got.OrderLines {
		sum += l.TotalAmount
	}
	if sum != client.got.Amount.Value {
		t.Errorf("receipt sums to %d but the charge is %d", sum, client.got.Amount.Value)
	}

	l := got.OrderLines[1]
	if l.Name != "T-shirt (Large)" {
		t.Errorf("line name = %q; the size is the only thing telling two t-shirts apart", l.Name)
	}
	if l.ID == "" {
		t.Error("MobilePay requires an id per line")
	}
	if l.UnitInfo.UnitPrice != 17500 || l.UnitInfo.Quantity != "1" || l.UnitInfo.QuantityUnit != "PCS" {
		t.Errorf("unit info = %+v", l.UnitInfo)
	}
	// Participation fees carry no VAT, so a line is entirely ex-tax.
	if l.TotalAmountExcludingTax != l.TotalAmount || l.TotalTaxAmount != 0 || l.TaxPercentage != 0 {
		t.Errorf("expected a zero-rated line, got total %d exTax %d tax %d pct %d",
			l.TotalAmount, l.TotalAmountExcludingTax, l.TotalTaxAmount, l.TaxPercentage)
	}
}

// Line ids must be distinct; a provider keying on them would otherwise collapse
// the receipt.
func TestMobilepayProviderReceiptLineIDsAreDistinct(t *testing.T) {
	client := &fakeMobilepayClient{}
	p := newMobilepayProvider(client, "https://x")
	if _, err := p.CreatePayment(payments.PaymentRequest{
		Amount: payments.Amount{Currency: types.CurrencyDKK, Value: 3},
		Lines: []payments.Line{
			{Label: "T-shirt (Large)", UnitCount: 1, UnitPrice: 1, Amount: 1},
			{Label: "T-shirt (Large)", UnitCount: 1, UnitPrice: 1, Amount: 1},
			{Label: "T-shirt (Small)", UnitCount: 1, UnitPrice: 1, Amount: 1},
		},
	}); err != nil {
		t.Fatalf("CreatePayment: %v", err)
	}
	seen := map[string]bool{}
	for _, l := range client.got.Receipt.OrderLines {
		if seen[l.ID] {
			t.Fatalf("duplicate line id %q", l.ID)
		}
		seen[l.ID] = true
	}
}

// No lines means no receipt block, but the currency must still be declared.
func TestMobilepayProviderWithoutLinesStillSetsCurrency(t *testing.T) {
	client := &fakeMobilepayClient{}
	p := newMobilepayProvider(client, "https://x")
	if _, err := p.CreatePayment(payments.PaymentRequest{
		Amount: payments.Amount{Currency: types.CurrencyDKK, Value: 100},
	}); err != nil {
		t.Fatalf("CreatePayment: %v", err)
	}
	if got := client.got.Receipt.OrderLines; got != nil {
		t.Errorf("expected no order lines, got %+v", got)
	}
	if client.got.Receipt.BottomLine.Currency != mobilepay.CurrencyDKK {
		t.Error("the bottom line currency should be set even without lines")
	}
}
