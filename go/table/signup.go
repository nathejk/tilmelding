package table

import (
	"fmt"
	"log"
	"time"

	"github.com/nathejk/shared-go/messages"
	"github.com/nathejk/shared-go/types"
	"github.com/seaqrs/tablerow"
	"nathejk.dk/pkg/streaminterface"

	_ "embed"
)

type Signup struct {
	TeamType     types.TeamType `sql:"teamType"`
	IsOpen       bool           `sql:"isOpen"`
	MaxSeatCount int            `sql:"maxSeatCount"`
}

type signup struct {
	w tablerow.Consumer
}

func NewSignup(w tablerow.Consumer) *signup {
	table := &signup{w: w}
	if err := w.Consume(table.CreateTableSql()); err != nil {
		log.Fatalf("Error creating table %q", err)
	}
	return table
}

//go:embed signup.sql
var signupSchema string

func (t *signup) CreateTableSql() string {
	return signupSchema
}

func (c *signup) Consumes() (subjs []streaminterface.Subject) {
	return []streaminterface.Subject{
		streaminterface.SubjectFromStr("nathejk"),
	}
}

func (c *signup) HandleMessage(msg streaminterface.Message) {
	switch msg.Subject().Subject() {
	case "nathejk:patrulje.signup.opened":
		var body messages.NathejkPatruljeSignupOpened
		if err := msg.Body(&body); err != nil {
			return
		}
		sql := fmt.Sprintf("INSERT INTO signup SET teamType=%q, isOpen=%d, maxSeatCount=%d ON DUPLICATE KEY UPDATE isOpen=VALUES(isOpen), maxSeatCount=VALUES(maxSeatCount)", types.TeamTypePatrulje, 1, body.MaxSeatCount)
		if err := c.w.Consume(sql); err != nil {
			log.Fatalf("Error consuming sql %q", err)
		}
	case "nathejk:patrulje.signup.closed":
		var body messages.NathejkPatruljeSignupClosed
		if err := msg.Body(&body); err != nil {
			return
		}
		sql := fmt.Sprintf("INSERT INTO signup SET teamType=%q, isOpen=%d ON DUPLICATE KEY UPDATE isOpen=VALUES(isOpen)", types.TeamTypePatrulje, 0)
		if err := c.w.Consume(sql); err != nil {
			log.Fatalf("Error consuming sql %q", err)
		}
	case "nathejk:klan.signup.opened":
		var body messages.NathejkKlanSignupOpened
		if err := msg.Body(&body); err != nil {
			return
		}
		sql := fmt.Sprintf("INSERT INTO signup SET teamType=%q, isOpen=%d, maxSeatCount=%d ON DUPLICATE KEY UPDATE isOpen=VALUES(isOpen), maxSeatCount=VALUES(maxSeatCount)", types.TeamTypeKlan, 1, body.MaxSeatCount)
		if err := c.w.Consume(sql); err != nil {
			log.Fatalf("Error consuming sql %q", err)
		}
	case "nathejk:klan.signup.closed":
		var body messages.NathejkKlanSignupClosed
		if err := msg.Body(&body); err != nil {
			return
		}
		sql := fmt.Sprintf("INSERT INTO signup SET teamType=%q, isOpen=%d ON DUPLICATE KEY UPDATE isOpen=VALUES(isOpen)", types.TeamTypeKlan, 0)
		if err := c.w.Consume(sql); err != nil {
			log.Fatalf("Error consuming sql %q", err)
		}
	case "nathejk:klan.signup.start.specified":
		var body messages.NathejkKlanSignupStartSpecified
		if err := msg.Body(&body); err != nil {
			return
		}
		if body.Time == nil {
			return
		}
		sql := fmt.Sprintf("INSERT INTO signup SET teamType=%q, startDate=%q ON DUPLICATE KEY UPDATE startDate=VALUES(startDate)", types.TeamTypeKlan, body.Time.Format(time.RFC3339))
		if err := c.w.Consume(sql); err != nil {
			log.Fatalf("Error consuming sql %q", err)
		}
	}
}
