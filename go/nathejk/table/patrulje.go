package table

import (
	"fmt"
	"log"

	"github.com/nathejk/shared-go/messages"
	"github.com/nathejk/shared-go/types"
	"nathejk.dk/pkg/tablerow"
	"nathejk.dk/superfluids/streaminterface"

	_ "embed"
)

type Patrulje struct {
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
		//streaminterface.SubjectFromStr("monolith:nathejk_team"),
		//streaminterface.SubjectFromStr("nathejk"),
		streaminterface.SubjectFromStr("NATHEJK:2025.patrulje.*.updated"),
		streaminterface.SubjectFromStr("NATHEJK:2025.patrulje.*.signedup"),
	}
}

func (c *patrulje) HandleMessage(msg streaminterface.Message) error {
	//log.Printf("patrulje.go RECEIVED %q", msg.Subject().Subject())
	switch true {
	case msg.Subject().Match("NATHEJK.*.patrulje.*.signedup"):
		var body messages.NathejkTeamSignedUp
		if err := msg.Body(&body); err != nil {
			return err
		}
		if body.TeamID == "" {
			return nil
		}
		sql := fmt.Sprintf("INSERT INTO patrulje SET teamId=%q, year=\"%d\", contactName=%q, contactPhone=%q, contactEmail=%q ON DUPLICATE KEY UPDATE contactName=VALUES(contactName), contactPhone=VALUES(contactPhone), contactEmail=VALUES(contactEmail)", body.TeamID, msg.Time().Year(), body.Name, body.Phone, body.Email)
		if err := c.w.Consume(sql); err != nil {
			log.Fatalf("Error consuming sql %q", err)
		}

	case msg.Subject().Match("nathejk:patrulje.updated"):
		var body messages.NathejkTeamUpdated
		if err := msg.Body(&body); err != nil {
			return err
		}
		err := c.w.Consume(fmt.Sprintf("UPDATE patrulje SET name=%q, groupName=%q, korps=%q, contactName=%q, contactPhone=%q, contactEmail=%q, contactRole=%q WHERE teamId=%q", body.Name, body.GroupName, body.Korps, body.ContactName, body.ContactPhone, body.ContactEmail, body.ContactRole, body.TeamID))
		if err != nil {
			log.Fatalf("Error consuming sql %q", err)
		}

	case msg.Subject().Match("nathejk:patrulje.status.changed"):
		var body messages.NathejkPatruljeStatusChanged
		if err := msg.Body(&body); err != nil {
			return err
		}
		err := c.w.Consume(fmt.Sprintf("UPDATE klan SET signupStatus=%q WHERE teamId=%q", body.Status, body.TeamID))
		if err != nil {
			log.Fatalf("Error consuming sql %q", err)
		}
	case msg.Subject().Match("NATHEJK.*.patrulje.*.updated"):
		var body messages.NathejkTeamUpdated
		if err := msg.Body(&body); err != nil {
			return err
		}
		msg.Subject().Parts()
		query := "UPDATE patrulje SET name=%q, groupName=%q, korps=%q, liga=%q, contactName=%q, contactPhone=%q, contactEmail=%q, contactRole=%q WHERE teamId=%q"
		args := []any{body.Name, body.GroupName, body.Korps, body.AdvspejdNumber, body.ContactName, body.ContactPhone, body.ContactEmail, substr(body.ContactRole, 0, 90), body.TeamID}
		//query := "INSERT INTO patrulje SET teamId=%q, year=\"%d\", contactName=%q, contactPhone=%q, contactEmail=%q ON DUPLICATE KEY UPDATE contactName=VALUES(contactName), conta    ctPhone=VALUES(contactPhone), contactEmail=VALUES(contactEmail)"
		//args := []any{body.TeamID, msg.Time().Year(), body.Name, body.Phone, body.Email}
		//, body.Name, body.GroupName, body.Korps, body.ContactName, body.ContactPhone, body.ContactEmail, body.ContactRole, body.TeamID))

		err := c.w.Consume(fmt.Sprintf(query, args...))
		if err != nil {
			log.Fatalf("Error consuming sql %q", err)
		}
	default:
		log.Printf("Unhandled message %q", msg.Subject().Subject())
		/*
			case "monolith:nathejk_team":
				var body messages.MonolithNathejkTeam
				if err := msg.Body(&body); err != nil {
					spew.Dump(msg)
					log.Print(err)
					return nil
				}
				if body.Entity.TypeName != types.TeamTypePatrulje {
					return nil
				}
				var sql string
				if body.Entity.DeletedUts.Time() == nil {
					//spew.Dump(body, body.Entity.CreatedUts.Time())
					if body.Entity.CreatedUts.Time() == nil {
						return nil
					}
					var memberCount int64
					if body.Entity.MemberCount != "" {
						memberCount, _ = strconv.ParseInt(body.Entity.MemberCount, 10, 64)
					}

					query := "INSERT INTO patrulje SET teamId=%q, year=\"%d\", teamNumber=%q, name=%q, groupName=%q, korps=%q, memberCount=%d, contactName=%q, contactPhone=%q, contactEmail=%q, signupStatus=%q  ON DUPLICATE KEY UPDATE teamNumber=VALUES(teamNumber), name=VALUES(name), groupName=VALUES(groupName), korps=VALUES(korps), memberCount=VALUES(memberCount), contactName=VALUES(contactName), contactPhone=VALUES(contactPhone), contactEmail=VALUES(contactEmail), signupStatus=VALUES(signupStatus)"
					args := []any{
						body.Entity.ID,
						body.Entity.CreatedUts.Time().Year(),
						body.Entity.TeamNumber,
						body.Entity.Title,
						body.Entity.Gruppe,
						body.Entity.Korps,
						memberCount,
						body.Entity.ContactTitle,
						body.Entity.ContactPhone,
						body.Entity.ContactMail,
						body.Entity.SignupStatusTypeName,
					}

					sql = fmt.Sprintf(query, args...)
				} else {
					sql = fmt.Sprintf("DELETE FROM patrulje WHERE teamId=%q", body.Entity.ID)
				}
				if err := c.w.Consume(sql); err != nil {
					log.Printf("Error consuming sql %q", err)
				}
		*/

	}
	return nil
}
func substr(input string, start int, length int) string {
	asRunes := []rune(input)

	if start >= len(asRunes) {
		return ""
	}

	if start+length > len(asRunes) {
		length = len(asRunes) - start
	}

	return string(asRunes[start : start+length])
}
