package table

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/nathejk/shared-go/messages"
	"github.com/nathejk/shared-go/types"
	queries "nathejk.dk/nathejk/table/payment"
	"nathejk.dk/pkg/tablerow"
	"nathejk.dk/superfluids/streaminterface"

	_ "embed"
)

type Payment struct {
	TeamID       types.TeamID       `sql:"teamId"`
	Year         string             `sql:"year"`
	Name         string             `sql:"name"`
	GroupName    string             `sql:"groupName"`
	Korps        string             `sql:"korps"`
	ContactName  string             `sql:"contactName"`
	ContactPhone types.PhoneNumber  `sql:"contactPhone"`
	ContactEmail types.EmailAddress `sql:"contactEmail"`
	ContactRole  string             `sql:"contactRole"`
	SignupStatus types.SignupStatus `sql:"signupStatus"`
}

type payment struct {
	queries.Query

	w tablerow.Consumer
}

func NewPayment(w tablerow.Consumer, r *sql.DB) *payment {
	table := &payment{Query: queries.Query{DB: r}, w: w}
	if err := w.Consume(table.CreateTableSql()); err != nil {
		log.Fatalf("Error creating table %q", err)
	}
	return table
}

//go:embed payment.sql
var paymentSchema string

func (t *payment) CreateTableSql() string {
	return paymentSchema
}

func (c *payment) Consumes() (subjs []streaminterface.Subject) {
	return []streaminterface.Subject{
		//streaminterface.SubjectFromStr("monolith:nathejk_team"),
		//streaminterface.SubjectFromStr("nathejk"),
		streaminterface.SubjectFromStr("NATHEJK.2025.payment.*.requested"),
		streaminterface.SubjectFromStr("NATHEJK.2025.payment.*.reserved"),
		streaminterface.SubjectFromStr("NATHEJK.2025.payment.*.received"),
	}
}

func (c *payment) HandleMessage(msg streaminterface.Message) error {
	switch true {
	case msg.Subject().Match("NATHEJK.*.payment.*.requested"):
		var body messages.NathejkPaymentRequested
		if err := msg.Body(&body); err != nil {
			return err
		}
		if body.Reference == "" {
			return nil
		}
		sql := fmt.Sprintf("INSERT INTO payment SET reference=%q, receiptEmail=%q, returnUrl=%q, year=\"%d\", currency=%q, amount=%d, method=%q, createdAt=%q, changedAt=%q, status=%q, orderForeignKey=%q, orderType=%q ON DUPLICATE KEY UPDATE receiptEmail=VALUES(receiptEmail), returnUrl=VALUES(returnUrl), year=VALUES(year), currency=VALUES(currency), amount=VALUES(amount), method=VALUES(method), status=VALUES(status), orderForeignKey=VALUES(orderForeignKey), orderType=VALUES(orderType)", body.Reference, body.ReceiptEmail, body.ReturnUrl, msg.Time().Year(), body.Currency, body.Amount, body.Method, msg.Time(), msg.Time(), types.PaymentStatusRequested, body.OrderForeignKey, body.OrderType)
		if err := c.w.Consume(sql); err != nil {
			log.Fatalf("Error consuming sql %q", err)
		}

	case msg.Subject().Match("NATHEJK.*.payment.*.reserved"):
		var body messages.NathejkPaymentReserved
		if err := msg.Body(&body); err != nil {
			return err
		}
		err := c.w.Consume(fmt.Sprintf("UPDATE payment SET status=%q, changedAt=%q WHERE reference=%q", types.PaymentStatusReserved, msg.Time(), body.Reference))
		if err != nil {
			log.Fatalf("Error consuming sql %q", err)
		}

	case msg.Subject().Match("NATHEJK.*.payment.*.received"):
		var body messages.NathejkPaymentReceived
		if err := msg.Body(&body); err != nil {
			return err
		}
		err := c.w.Consume(fmt.Sprintf("UPDATE payment SET status=%q, changedAt=%q WHERE reference=%q", types.PaymentStatusReceived, msg.Time(), body.Reference))
		if err != nil {
			log.Fatalf("Error consuming sql %q", err)
		}
	default:
		log.Printf("Unhandled message %q", msg.Subject().Subject())
	}
	return nil
}
