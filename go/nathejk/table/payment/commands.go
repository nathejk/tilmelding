package payment

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jrgensen/cqrs"
	"github.com/nathejk/shared-go/messages"
	"github.com/nathejk/shared-go/types"
)

// Commands is the payment write-side API. Methods publish payment events onto
// the stream and drive the payment provider.
type Commands interface {
	Request(amount Amount, desc string, phone types.PhoneNumber, email types.EmailAddress, returnUrl, orderForeignKey, orderType string) (string, error)
	Capture(reference string) error
}

type commander struct {
	p    cqrs.Publisher
	pp   Provider
	year types.YearSlug
}

// NewCommands wires a payment commander. The publisher emits the
// NathejkPayment* events that drive the projections; the provider creates and
// captures authorisations.
//
// year is the season the published events belong to and appears in their
// subjects. Both copies of this entity hard-coded "2026" here, which the
// projector then stored as the payment's year — so a new season needed a code
// change in two places, and the consumer's matching pattern was pinned to the
// same literal. It is a parameter for the same reason the order entity takes
// one.
func NewCommands(p cqrs.Publisher, pp Provider, year types.YearSlug) Commands {
	return &commander{p: p, pp: pp, year: year}
}

// Request authorises a payment with the provider and announces it on the
// stream, returning the URL the payer must be sent to.
//
// The event carries the provider's reference rather than the locally generated
// one: that is the identity every later payment event and the projection join
// on. The idempotency key is separate and deliberately distinct, so a retried
// request cannot look like the same payment being renamed.
func (c *commander) Request(amount Amount, desc string, phone types.PhoneNumber, email types.EmailAddress, returnUrl string, orderForeignKey string, orderType string) (string, error) {
	reference := uuid.New().String()
	resp, err := c.pp.CreatePayment(PaymentRequest{
		IdempotencyKey: uuid.New().String(),
		Reference:      reference,
		Amount:         amount,
		Description:    desc,
		PhoneNumber:    phone.InternationalNumber(),
	})
	if err != nil {
		return "", err
	}

	body := messages.NathejkPaymentRequested{
		Reference:       resp.Reference,
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
	msg := c.p.MessageFunc()(c.subject(resp.Reference, "requested"))
	msg.SetBody(body)

	if err := c.p.Publish(msg); err != nil {
		return "", err
	}
	return resp.RedirectURL, nil
}

// Capture claims the authorised-but-not-yet-taken funds of a payment.
//
// Reachable more than once — the provider's callback is a plain GET the payer
// can reload — so it must be safe to repeat: nothing is captured and nothing
// published once the authorisation is exhausted.
func (c *commander) Capture(reference string) error {
	auth, err := c.pp.GetAuthorization(reference)
	if err != nil {
		return err
	}

	// Capture only what is authorised and not yet taken. Both totals are
	// cumulative, so this stays correct across partial captures.
	available := Amount{
		Currency: auth.Currency,
		Value:    auth.AuthorizedAmount - auth.CapturedAmount,
	}

	if !auth.Authorized || available.Value <= 0 {
		return nil
	}

	// reserved is published before the capture and received after it, so a
	// crash mid-capture leaves evidence that money was expected.
	msg := c.p.MessageFunc()(c.subject(reference, "reserved"))
	msg.SetBody(&messages.NathejkPaymentReserved{
		Reference: reference,
		Amount:    int(available.Value),
		Currency:  string(available.Currency),
		Timestamp: time.Now(),
	})
	if err := c.p.Publish(msg); err != nil {
		return err
	}

	if err := c.pp.CapturePayment(reference, available); err != nil {
		return err
	}

	msg = c.p.MessageFunc()(c.subject(reference, "received"))
	msg.SetBody(&messages.NathejkPaymentReceived{
		Reference: reference,
		Amount:    int(available.Value),
		Currency:  string(available.Currency),
		Timestamp: time.Now(),
	})
	return c.p.Publish(msg)
}

// subject builds the event subject for a payment transition. Centralised
// because the three call sites previously disagreed on the domain separator
// ("NATHEJK:2026." vs "NATHEJK.2026."); both normalise to the same subject, but
// only one of them says so.
func (c *commander) subject(reference, event string) cqrs.Subject {
	return cqrs.SubjectFromStr(fmt.Sprintf("NATHEJK:%s.payment.%s.%s", c.year, reference, event))
}

var _ Commands = (*commander)(nil)
