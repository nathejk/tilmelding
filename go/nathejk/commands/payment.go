package commands

import (
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/google/uuid"
	"github.com/nathejk/shared-go/messages"
	"github.com/nathejk/shared-go/types"
	"nathejk.dk/internal/payment/mobilepay"
	"nathejk.dk/superfluids/streaminterface"
)

type paymentQuerier interface {
	//ConfirmBySecret(string) (*data.Confirm, error)
	//GetKlan(types.TeamID) (*data.Klan, error)
	//RequestedSeniorCount() int
}
type payment struct {
	p  streaminterface.Publisher
	q  paymentQuerier
	pp mobilepay.Client
}

func NewPayment(p streaminterface.Publisher, q paymentQuerier, pp mobilepay.Client) *payment {
	return &payment{
		p:  p,
		q:  q,
		pp: pp,
	}
}

func (c *payment) Request(amount mobilepay.Amount, desc string, phone types.PhoneNumber, email types.EmailAddress, returnUrl string, orderForeignKey string, orderType string) (string, error) {
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
	msg := c.p.MessageFunc()(streaminterface.SubjectFromStr(fmt.Sprintf("NATHEJK:%s.payment.%s.requested", "2025", resp.Reference)))
	msg.SetBody(body)
	meta := messages.Metadata{Producer: "tilmelding-api"}
	msg.SetMeta(&meta)

	if err := c.p.Publish(msg); err != nil {
		return "", err
	}
	return resp.RedirectUrl, nil
}

func (c *payment) Capture(reference string) error {
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
	msg := c.p.MessageFunc()(streaminterface.SubjectFromStr(fmt.Sprintf("NATHEJK.%s.payment.%s.reserved", "2025", reference)))
	msg.SetBody(body)
	msg.SetMeta(&messages.Metadata{Producer: "tilmelding-api"})

	if err := c.p.Publish(msg); err != nil {
		return err
	}
	if _, err := c.pp.CapturePayment(mobilepay.PaymentReference(reference), availableAmount); err != nil {
		return err
	}
	msg = c.p.MessageFunc()(streaminterface.SubjectFromStr(fmt.Sprintf("NATHEJK:%s.payment.%s.received", "2025", reference)))
	msg.SetBody(&messages.NathejkPaymentReceived{
		Reference: reference,
		Amount:    int(availableAmount.Value),
		Currency:  string(availableAmount.Currency),
		Timestamp: time.Now(),
	})
	msg.SetMeta(&messages.Metadata{Producer: "tilmelding-api"})

	if err := c.p.Publish(msg); err != nil {
		return err
	}

	// TODO send mail
	return nil
}

func (c *payment) Signup(teamType types.TeamType, body *messages.NathejkTeamSignedUp) error {
	if body.TeamID == "" {
		body.TeamID = types.TeamID(uuid.New().String())
	}
	if body.Pincode == "" {
		body.Pincode = fmt.Sprintf("%d", rand.IntN(9000)+1000)
	}

	msg := c.p.MessageFunc()(streaminterface.SubjectFromStr(fmt.Sprintf("NATHEJK:%s.%s.%s.signedup", "2025", teamType, body.TeamID)))
	msg.SetBody(body)
	meta := messages.Metadata{Producer: "tilmelding-api"}
	msg.SetMeta(&meta)

	if err := c.p.Publish(msg); err != nil {
		return err
	}
	return nil
}
