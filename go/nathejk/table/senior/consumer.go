package senior

import (
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

func (c *consumer) Consumes() []streaminterface.Subject {
	return []streaminterface.Subject{
		streaminterface.SubjectFromStr("NATHEJK.*.senior.*.updated"),
		streaminterface.SubjectFromStr("NATHEJK.*.senior.*.deleted"),
		streaminterface.SubjectFromStr("NATHEJK.*.bandit.*.armNumber.assigned"),
	}
}

func (c *consumer) HandleMessage(msg streaminterface.Message) error {
	switch true {
	case msg.Subject().Match("nathejk.*.senior.*.updated"):
		var body messages.NathejkSeniorUpdated
		if err := msg.Body(&body); err != nil {
			return err
		}
		query := `INSERT INTO senior
			(memberId, year, teamId, name, address, postalCode, city, email, phone, birthday, tshirtSize, diet,  createdAt, updatedAt)
			VALUES (%q,%q,%q,%q,%q,%q,%q,%q,%q,%q,%q,%q,%q,%q)
			ON DUPLICATE KEY UPDATE
			teamId=VALUES(teamId), name=VALUES(name), address=VALUES(address), postalCode=VALUES(postalCode),city=VALUES(city), email=VALUES(email), phone=VALUES(phone), birthday=VALUES(birthday), tshirtSize=VALUES(tshirtSize), diet=VALUES(diet), updatedAt=VALUES(updatedAt)`
		args := []any{
			body.MemberID,
			msg.Subject().Parts()[1],
			body.TeamID,
			body.Name,
			body.Address,
			body.PostalCode,
			body.City,
			body.Email,
			body.Phone,
			body.BirthDate,
			body.TShirtSize,
			body.Diet,
			msg.Time(),
			msg.Time(),
		}
		if err := c.w.Consume(fmt.Sprintf(query, args...)); err != nil {
			log.Fatalf("Error consuming sql %q", err)
		}

	case msg.Subject().Match("nathejk.*.senior.*.deleted"):
		var body messages.NathejkMemberDeleted
		if err := msg.Body(&body); err != nil {
			return err
		}
		query := "DELETE FROM senior WHERE memberId=%q"
		args := []any{body.MemberID}
		err := c.w.Consume(fmt.Sprintf(query, args...))
		if err != nil {
			log.Fatalf("Error consuming sql %q", err)
		}

	case msg.Subject().Match("NATHEJK.*.bandit.*.armNumber.assigned"):
		var body messages.NathejkLokArmNumberAssigned
		if err := msg.Body(&body); err != nil {
			return err
		}
		query := "UPDATE senior SET armNumber=%q WHERE memberId=%q"
		args := []any{
			body.ArmNumber,
			msg.Subject().Parts()[3],
		}
		err := c.w.Consume(fmt.Sprintf(query, args...))
		if err != nil {
			log.Fatalf("Error consuming sql %q", err)
		}

	default:
		log.Printf("Unhandled message %q", msg.Subject().Subject())
	}
	return nil
}
