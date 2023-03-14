package table

import (
	"fmt"
	"log"

	"nathejk.dk/pkg/messages"
	"nathejk.dk/pkg/streaminterface"
	"nathejk.dk/pkg/tablerow"
	"nathejk.dk/pkg/types"

	_ "embed"
)

type Patrulje struct {
	TeamID       types.TeamID       `sql:"teamId"`
	Name         string             `sql:"name"`
	GroupName    string             `sql:"groupName"`
	Korps        string             `sql:"korps"`
	ContactName  string             `sql:"contactName"`
	ContactPhone types.PhoneNumber  `sql:"contactPhone"`
	ContactEmail types.Email        `sql:"contactEmail"`
	ContactRole  string             `sql:"contactRole"`
	SignupStatus types.SignupStatus `sql:"signupStatus"`
}

type patrulje struct {
	w tablerow.Consumer
}

func NewPatrulje(w tablerow.Consumer) *patrulje {
	table := &patrulje{w: w}
	if err := w.Consume(table.CreateTableSql()); err != nil {
		log.Fatalf("Error creating table %q", err)
	}
	return table
}

//go:embed patrulje.sql
var patruljeSchema string

func (t *patrulje) CreateTableSql() string {
	return patruljeSchema
}

func (c *patrulje) Consumes() (subjs []streaminterface.Subject) {
	return []streaminterface.Subject{
		streaminterface.SubjectFromStr("nathejk"),
	}
}

func (c *patrulje) HandleMessage(msg streaminterface.Message) {
	switch msg.Subject().Subject() {
	case "nathejk:patrulje.signedup":
		var body messages.NathejkTeamSignedUp
		if err := msg.Body(&body); err != nil {
			return
		}
		if body.TeamID == "" {
			return
		}
		sql := fmt.Sprintf("INSERT INTO patrulje SET teamId=%q, contactName=%q, contactPhone=%q, contactEmail=%q ON DUPLICATE KEY UPDATE contactName=VALUES(contactName), contactPhone=VALUES(contactPhone), contactEmail=VALUES(contactEmail)", body.TeamID, body.Name, body.Phone, body.Email)
		if err := c.w.Consume(sql); err != nil {
			log.Fatalf("Error consuming sql %q", err)
		}

	case "nathejk:patrulje.updated":
		var body messages.NathejkTeamUpdated
		if err := msg.Body(&body); err != nil {
			return
		}
		err := c.w.Consume(fmt.Sprintf("UPDATE patrulje SET name=%q, groupName=%q, korps=%q, contactName=%q, contactPhone=%q, contactEmail=%q, contactRole=%q WHERE teamId=%q", body.Name, body.GroupName, body.Korps, body.ContactName, body.ContactPhone, body.ContactEmail, body.ContactRole, body.TeamID))
		if err != nil {
			log.Fatalf("Error consuming sql %q", err)
		}

	case "nathejk:patrulje.status.changed":
		var body messages.NathejkPatruljeStatusChanged
		if err := msg.Body(&body); err != nil {
			return
		}
		err := c.w.Consume(fmt.Sprintf("UPDATE klan SET signupStatus=%q WHERE teamId=%q", body.Status, body.TeamID))
		if err != nil {
			log.Fatalf("Error consuming sql %q", err)
		}
	}
}
