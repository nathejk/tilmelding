package personnel

import (
	"context"
	"fmt"

	"github.com/nathejk/shared-go/messages"
	"github.com/nathejk/shared-go/types"
	"nathejk.dk/superfluids/streaminterface"
)

// Commands is the personnel write-side API. It publishes a per-user
// NathejkPersonnelUpdated event into the stream; the personnel projection
// (consumer.go) is the read-side.
type Commands interface {
	Update(ctx context.Context, userID types.UserID, userType types.TeamType, person Person) error
}

// Person is the input shape of an Update command. All fields except ID
// are taken at face value — there is no merge with existing values, so
// callers should pass the full edited record.
type Person struct {
	ID          types.UserID       `json:"personId"`
	Name        string             `json:"name"`
	Email       types.EmailAddress `json:"email"`
	Phone       types.PhoneNumber  `json:"phone"`
	TshirtSize  string             `json:"tshirtSize"`
	Group       string             `json:"group"`
	Korps       types.CorpsSlug    `json:"korps"`
	Klan        string             `json:"klan"`
	Additionals map[string]any     `json:"additionals"`
}

type commander struct {
	p streaminterface.Publisher
}

func (c *commander) Update(ctx context.Context, userID types.UserID, userType types.TeamType, person Person) error {
	msg := c.p.MessageFunc()(streaminterface.SubjectFromStr(fmt.Sprintf("NATHEJK:%s.%s.%s.updated", "2026", userType, userID)))
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
	return c.p.Publish(msg)
}
