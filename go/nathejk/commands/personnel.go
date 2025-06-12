package commands

import (
	"fmt"
	"math/rand/v2"

	"github.com/google/uuid"
	"github.com/nathejk/shared-go/messages"
	"github.com/nathejk/shared-go/types"
	"nathejk.dk/internal/data"
	"nathejk.dk/superfluids/streaminterface"
)

type personnelQuerier interface {
	//ConfirmBySecret(string) (*data.Confirm, error)
	GetKlan(types.TeamID) (*data.Klan, error)
	RequestedSeniorCount() int
}
type personnel struct {
	p streaminterface.Publisher
	q personnelQuerier
}

func NewPersonnel(p streaminterface.Publisher, q teamQuerier) *personnel {
	return &personnel{
		p: p,
		q: q,
	}
}

func (c *personnel) Signup(teamType types.TeamType, body *messages.NathejkTeamSignedUp) error {
	if body.TeamID == "" {
		body.TeamID = types.TeamID(uuid.New().String())
	}
	if body.Pincode == "" {
		body.Pincode = fmt.Sprintf("%d", rand.IntN(9000)+1000)
	}

	msg := c.p.MessageFunc()(streaminterface.SubjectFromStr(fmt.Sprintf("NATHEJK:%s.%s.%s.signedup", "2025", teamType, body.TeamID)))
	msg.SetBody(body)
	meta := messages.Metadata{Producer: "tilmelding-api"}
	msg.SetMeta(&meta)

	if err := c.p.Publish(msg); err != nil {
		return err
	}
	return nil
}

func (c *personnel) UpdatePerson(userID types.UserID, userType types.TeamType, person Person) error {
	msg := c.p.MessageFunc()(streaminterface.SubjectFromStr(fmt.Sprintf("NATHEJK:%s.%s.%s.updated", "2025", userType, userID)))
	msg.SetBody(&messages.NathejkPersonnelUpdated{
		UserID:      userID,
		Name:        person.Name,
		Phone:       person.Phone,
		Email:       person.Email,
		Group:       person.Group,
		Corps:       person.Korps,
		Klan:        person.Klan,
		TshirtSize:  person.TshirtSize,
		Additionals: person.Additionals,
	})
	msg.SetMeta(&messages.Metadata{Producer: "tilmelding-api"})
	if err := c.p.Publish(msg); err != nil {
		return err
	}

	return nil
}
