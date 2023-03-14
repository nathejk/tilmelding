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

type Spejder struct {
	MemberID    types.MemberID
	TeamID      types.TeamID
	Name        string
	Address     string
	PostalCode  string
	City        string
	Email       types.Email
	Phone       types.PhoneNumber
	PhoneParent types.PhoneNumber
	Birthday    types.Date
	Returning   bool
	Created     time.Time
}

type spejder struct {
	w tablerow.Consumer
}

func NewSpejder(w tablerow.Consumer) *spejder {
	table := &spejder{w: w}
	if err := w.Consume(table.CreateTableSql()); err != nil {
		log.Fatalf("Error creating table %q", err)
	}
	return table
}

//go:embed spejder.sql
var spejderSchema string

func (t *spejder) CreateTableSql() string {
	return spejderSchema
}

func (c *spejder) Consumes() (subjs []streaminterface.Subject) {
	return []streaminterface.Subject{
		streaminterface.SubjectFromStr("nathejk"),
	}
}

func (c *spejder) HandleMessage(msg streaminterface.Message) {
	switch msg.Subject().Subject() {
	case "nathejk:spejder.updated":
		var body messages.NathejkMemberUpdated
		if err := msg.Body(&body); err != nil {
			return
		}
		//log.Printf("spejder %q", body.Type)
		//if body.Type != "spejder" {
		//	return
		//}
		returning := "0"
		if body.Returning {
			returning = "1"
		}
		err := c.w.Consume(fmt.Sprintf("INSERT INTO spejder (memberId, teamId, name, address, postalCode, city, email, phone, phoneParent, birthday, `returning`, createdAt, updatedAt) VALUES (%q,%q,%q,%q,%q,%q,%q,%q,%q,%q,%q,%q,%q) ON DUPLICATE KEY UPDATE teamId=VALUES(teamId), name=VALUES(name), address=VALUES(address), postalCode=VALUES(postalCode),city=VALUES(city),email=VALUES(email),phone=VALUES(phone), phoneParent=VALUES(phoneParent), birthday=VALUES(birthday), `returning`=VALUES(`returning`),  updatedAt=VALUES(updatedAt)", body.MemberID, body.TeamID, body.Name, body.Address, body.PostalCode, body.City, body.Email, body.Phone, body.PhoneParent, body.Birthday, returning, msg.Time(), msg.Time()))
		if err != nil {
			log.Fatalf("Error consuming sql %q", err)
		}
	case "nathejk:spejder.deleted":
		var body messages.NathejkMemberDeleted
		if err := msg.Body(&body); err != nil {
			return
		}
		err := c.w.Consume(fmt.Sprintf("DELETE FROM spejder WHERE memberId=%q", body.MemberID))
		if err != nil {
			log.Fatalf("Error consuming sql %q", err)
		}
	}
}
