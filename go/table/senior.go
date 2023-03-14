package table

import (
	"fmt"
	"log"
	"time"

	"nathejk.dk/pkg/messages"
	"nathejk.dk/pkg/streaminterface"
	"nathejk.dk/pkg/tablerow"
	"nathejk.dk/pkg/types"

	_ "embed"
)

type Senior struct {
	MemberID   types.MemberID
	TeamID     types.TeamID
	Name       string
	Address    string
	PostalCode string
	City       string
	Email      types.Email
	Phone      types.PhoneNumber
	Birthday   types.Date
	Returning  bool
	Created    time.Time
}

type senior struct {
	w tablerow.Consumer
}

func NewSenior(w tablerow.Consumer) *senior {
	table := &senior{w: w}
	if err := w.Consume(table.CreateTableSql()); err != nil {
		log.Fatalf("Error creating table %q", err)
	}
	return table
}

//go:embed senior.sql
var seniorSchema string

func (t *senior) CreateTableSql() string {
	return seniorSchema
}

func (c *senior) Consumes() (subjs []streaminterface.Subject) {
	return []streaminterface.Subject{
		streaminterface.SubjectFromStr("nathejk"),
	}
}

func (c *senior) HandleMessage(msg streaminterface.Message) {
	switch msg.Subject().Subject() {
	case "nathejk:senior.updated":
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
		err := c.w.Consume(fmt.Sprintf("INSERT INTO senior (memberId, teamId, name, address, postalCode, city, email, phone, birthday, `returning`, createdAt, updatedAt) VALUES (%q,%q,%q,%q,%q,%q,%q,%q,%q,%q,%q,%q) ON DUPLICATE KEY UPDATE teamId=VALUES(teamId), name=VALUES(name), address=VALUES(address), postalCode=VALUES(postalCode),city=VALUES(city),email=VALUES(email),phone=VALUES(phone), birthday=VALUES(birthday), `returning`=VALUES(`returning`),  updatedAt=VALUES(updatedAt)", body.MemberID, body.TeamID, body.Name, body.Address, body.PostalCode, body.City, body.Email, body.Phone, body.Birthday, returning, msg.Time(), msg.Time()))
		if err != nil {
			log.Fatalf("Error consuming sql %q", err)
		}
	case "nathejk:senior.deleted":
		var body messages.NathejkMemberDeleted
		if err := msg.Body(&body); err != nil {
			return
		}
		err := c.w.Consume(fmt.Sprintf("DELETE FROM senior WHERE memberId=%q", body.MemberID))
		if err != nil {
			log.Fatalf("Error consuming sql %q", err)
		}
	}
}
