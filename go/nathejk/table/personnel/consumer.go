package personnel

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/nathejk/shared-go/messages"
	"nathejk.dk/pkg/tablerow"
	"nathejk.dk/superfluids/streaminterface"

	_ "embed"
)

type consumer struct {
	w tablerow.Consumer
}

func (*consumer) Consumes() []streaminterface.Subject {
	return []streaminterface.Subject{
		streaminterface.SubjectFromStr("NATHEJK.*.gøgler.*.signedup"),
		streaminterface.SubjectFromStr("NATHEJK.*.gøgler.*.updated"),
		streaminterface.SubjectFromStr("NATHEJK.*.gøgler.*.status.changed"),
		// Crew signups are projected into the crewmember table now, not here.
		// See nathejk/table/crewmember/consumer.go. The gøgler (badut) flow
		// still lives on personnel.
		//
		// Legacy subjects from the pre-rename era. The personnel projection
		// continues to consume these so old staff/friend events can still flow
		// into the table for historical records.
		streaminterface.SubjectFromStr("NATHEJK.*.staff.*.signedup"),
		streaminterface.SubjectFromStr("NATHEJK.*.staff.*.updated"),
		streaminterface.SubjectFromStr("NATHEJK.*.staff.*.status.changed"),
		streaminterface.SubjectFromStr("NATHEJK.*.friend.*.signedup"),
		streaminterface.SubjectFromStr("NATHEJK.*.friend.*.updated"),
		streaminterface.SubjectFromStr("NATHEJK.*.friend.*.status.changed"),
	}
}

func (c *consumer) HandleMessage(msg streaminterface.Message) error {
	switch true {

	case msg.Subject().Match("NATHEJK.*.*.*.signedup"):
		var body messages.NathejkTeamSignedUp
		if err := msg.Body(&body); err != nil {
			return err
		}
		if body.TeamID == "" {
			return nil
		}
		subject := msg.Subject().Parts()
		// Normalise legacy userType values onto "crew". The pre-rename code
		// emitted NATHEJK.*.staff.* and NATHEJK.*.friend.*; both now project
		// to userType="crew" so the runtime layer only ever sees a single
		// crew identifier on the personnel table.
		userType := subject[2]
		if userType == "staff" || userType == "friend" {
			userType = "crew"
		}
		args := []any{body.TeamID, userType, subject[1], body.Name, body.Phone, body.Email}
		sql := fmt.Sprintf("INSERT IGNORE INTO personnel SET userId=%q, userType=%q, year=%q, name=%q, phone=%q, email=%q", args...)
		if err := c.w.Consume(sql); err != nil {
			log.Fatalf("Error consuming sql %q", err)
		}

	case msg.Subject().Match("NATHEJK.*.*.*.updated"):
		var body messages.NathejkPersonnelUpdated
		if err := msg.Body(&body); err != nil {
			return err
		}
		additionals, _ := json.Marshal(body.Additionals)
		msg.Subject().Parts()
		query := "UPDATE personnel SET name=%q, groupName=%q, korps=%q, klan=%q, phone=%q, email=%q, tshirtSize=%q, additionals=%q  WHERE userId=%q"
		args := []any{body.Name, body.Group, string(body.Corps), body.Klan, body.Phone, body.Email, body.TshirtSize, additionals, body.UserID}

		err := c.w.Consume(fmt.Sprintf(query, args...))
		if err != nil {
			log.Fatalf("Error consuming sql %q", err)
		}
		/*
			case msg.Subject().Match("NATHEJK.*.staff.*.status.changed"):
				var body messages.NathejkStaffStatusChanged
				if err := msg.Body(&body); err != nil {
					return err
				}
				err := c.w.Consume(fmt.Sprintf("UPDATE staff SET signupStatus=%q WHERE staffId=%q", body.Status, body.StaffID))
				if err != nil {
					log.Fatalf("Error consuming sql %q", err)
				}
		*/
	default:
		log.Printf("Unhandled message %q", msg.Subject().Subject())
	}
	return nil
}
