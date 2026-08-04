package table

import (
	"fmt"
	"log"

	"github.com/jrgensen/cqrs"
	"github.com/nathejk/shared-go/messages"

	_ "embed"
)

type registrant struct {
	w cqrs.Writer
}

func NewRegistrant(w cqrs.Writer) *registrant {
	table := &registrant{w: w}
	if err := w.Consume(table.CreateTableSql()); err != nil {
		log.Fatalf("Error creating table %q", err)
	}
	return table
}

//go:embed registrant.sql
var registrantSchema string

func (t *registrant) CreateTableSql() string {
	return registrantSchema
}

func (c *registrant) Consumes() (subjs []cqrs.Subject) {
	return []cqrs.Subject{
		cqrs.SubjectFromStr("nathejk"),
	}
}

func (c *registrant) HandleMessage(msg cqrs.Message) {
	switch msg.Subject().Subject() {
	case "nathejk:patrulje.signedup", "nathejk:klan.signedup":
		var body messages.NathejkTeamSignedUp
		if err := msg.Body(&body); err != nil {
			return
		}
		err := c.w.Consume(fmt.Sprintf("REPLACE INTO registrant SET registrantId=%q, email=%q, phone=%q, pincode=%q", body.TeamID, body.Email, body.Phone, body.Pincode))
		if err != nil {
			log.Fatalf("Error consuming sql %q", err)
		}
	}
}
