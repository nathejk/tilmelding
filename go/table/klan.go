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

type Klan struct {
	TeamID       types.TeamID       `sql:"teamId"`
	Name         string             `sql:"name"`
	GroupName    string             `sql:"groupName"`
	Korps        string             `sql:"korps"`
	SignupStatus types.SignupStatus `sql:"signupStatus"`
}

type klan struct {
	w tablerow.Consumer
}

func NewKlan(w tablerow.Consumer) *klan {
	table := &klan{w: w}
	if err := w.Consume(table.CreateTableSql()); err != nil {
		log.Fatalf("Error creating table %q", err)
	}
	return table
}

//go:embed klan.sql
var klanSchema string

func (t *klan) CreateTableSql() string {
	return klanSchema
}

func (c *klan) Consumes() (subjs []streaminterface.Subject) {
	return []streaminterface.Subject{
		streaminterface.SubjectFromStr("nathejk"),
	}
}

func (c *klan) HandleMessage(msg streaminterface.Message) {
	switch msg.Subject().Subject() {
	case "nathejk:klan.signedup":
		var body messages.NathejkTeamSignedUp
		if err := msg.Body(&body); err != nil {
			return
		}
		if body.TeamID == "" {
			return
		}
		sql := fmt.Sprintf("INSERT INTO klan SET teamId=%q, signedUpAt=%q, signupStatus='NEW' ON DUPLICATE KEY UPDATE signedUpAt=VALUES(signedUpAt)", body.TeamID, msg.Time().Format(time.RFC3339))
		if err := c.w.Consume(sql); err != nil {
			log.Fatalf("Error consuming sql %q", err)
		}

	case "nathejk:klan.updated":
		var body messages.NathejkKlanUpdated
		if err := msg.Body(&body); err != nil {
			return
		}
		err := c.w.Consume(fmt.Sprintf("UPDATE klan SET name=%q, groupName=%q, korps=%q WHERE teamId=%q", body.Name, body.GroupName, body.Korps, body.TeamID))
		if err != nil {
			log.Fatalf("Error consuming sql %q", err)
		}

	case "nathejk:klan.status.changed":
		var body messages.NathejkKlanStatusChanged
		if err := msg.Body(&body); err != nil {
			return
		}
		sql := fmt.Sprintf("UPDATE klan SET signupStatus=%q, statusChangedAt=%q WHERE teamId=%q", body.Status, msg.Time().Format(time.RFC3339), body.TeamID)
		if err := c.w.Consume(sql); err != nil {
			log.Fatalf("Error consuming sql %q", err)
		}
	}
}
