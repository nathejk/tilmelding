package payment

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jrgensen/stream"
	"github.com/jrgensen/stream/subject"
	"github.com/nathejk/shared-go/messages"
	"github.com/nathejk/shared-go/types"
	"nathejk.dk/internal/payment/mobilepay"
)

// Commands is the payment write-side API. Methods publish payment events
// onto the stream and (where relevant) drive the MobilePay client.
type Commands interface {
	Request(amount mobilepay.Amount, desc string, phone types.PhoneNumber, email types.EmailAddress, returnUrl, orderForeignKey, orderType string) (string, error)
	Capture(reference string) error
}

type commander struct {
	p  stream.Publisher
	pp mobilepay.Client
}

// NewCommands wires a payment commander. The publisher is used for
// emitting the NathejkPayment* events that drive the projections; the
// MobilePay client is used to create and capture authorisations.
func NewCommands(p stream.Publisher, pp mobilepay.Client) Commands {
	return &commander{p: p, pp: pp}
}

func (c *commander) Request(amount mobilepay.Amount, desc string, phone types.PhoneNumber, email types.EmailAddress, returnUrl string, orderForeignKey string, orderType string) (string, error) {
	reference := uuid.New().String()
	p := mobilepay.Payment{
		Amount:             amount,
		PaymentMethod:      mobilepay.PaymentMethod{Type: mobilepay.PaymentMethodType("WALLET")},
		Customer:           mobilepay.Customer{PhoneNumber: phone.InternationalNumber()},
		Reference:          mobilepay.PaymentReference(reference),
		ReturnUrl:          "https://tilmelding.nathejk.dk/callback/mobilepay/" + reference,
		UserFlow:           mobilepay.UserFlowWeb,
		PaymentDescription: desc,
	}
	key := uuid.New().String()
	resp, err := c.pp.CreatePayment(key, p)
	if err != nil {
		return "", err
	}

	body := messages.NathejkPaymentRequested{
		Reference:       string(resp.Reference),
		ReceiptEmail:    email,
		ReturnUrl:       returnUrl,
		Amount:          int(amount.Value),
		Currency:        string(amount.Currency),
		Timestamp:       time.Now(),
		Method:          "mobilepay",
		OrderLines:      []messages.NathejkPayment_OrderLine{},
		OrderForeignKey: orderForeignKey,
		OrderType:       orderType,
	}
	msg := c.p.MessageFunc()(subject.FromStr(fmt.Sprintf("NATHEJK:%s.payment.%s.requested", "2026", resp.Reference)))
	msg.SetBody(body)

	if err := c.p.Publish(msg); err != nil {
		return "", err
	}
	return resp.RedirectUrl, nil
}

func (c *commander) Capture(reference string) error {
	mpp, err := c.pp.GetPayment(mobilepay.PaymentReference(reference))
	if err != nil {
		return err
	}

	availableAmount := mpp.Amount
	availableAmount.Value = mpp.Aggregate.AuthorizedAmount.Value - mpp.Aggregate.CapturedAmount.Value

	if (mpp.State != mobilepay.PaymentStateAuthorized) || (availableAmount.Value <= 0) {
		return nil
	}

	body := &messages.NathejkPaymentReserved{
		Reference: reference,
		Amount:    int(availableAmount.Value),
		Currency:  string(availableAmount.Currency),
		Timestamp: time.Now(),
	}
	msg := c.p.MessageFunc()(subject.FromStr(fmt.Sprintf("NATHEJK.%s.payment.%s.reserved", "2026", reference)))
	msg.SetBody(body)

	if err := c.p.Publish(msg); err != nil {
		return err
	}
	if _, err := c.pp.CapturePayment(mobilepay.PaymentReference(reference), availableAmount); err != nil {
		return err
	}
	msg = c.p.MessageFunc()(subject.FromStr(fmt.Sprintf("NATHEJK:%s.payment.%s.received", "2026", reference)))
	msg.SetBody(&messages.NathejkPaymentReceived{
		Reference: reference,
		Amount:    int(availableAmount.Value),
		Currency:  string(availableAmount.Currency),
		Timestamp: time.Now(),
	})

	if err := c.p.Publish(msg); err != nil {
		return err
	}

	// TODO send mail
	return nil
}
