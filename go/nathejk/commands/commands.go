package commands

import (
	"github.com/nathejk/shared-go/messages"
	"github.com/nathejk/shared-go/types"
	"nathejk.dk/internal/data"
	"nathejk.dk/internal/payment/mobilepay"
	"nathejk.dk/superfluids/streaminterface"
)

type Commands struct {
	Team interface {
		Signup(types.TeamType, *messages.NathejkTeamSignedUp) error
		UpdatePatrulje(types.TeamID, Patrulje, Contact, []Spejder) error
		UpdateKlan(types.TeamID, Klan, []Senior) error
	}
	Payment interface {
		Request(amount mobilepay.Amount, desc string, phone types.PhoneNumber, email types.EmailAddress, returnUrl, orderForeignKey, orderType string) (string, error)
		Capture(reference string) error
	}
	Personnel interface {
		UpdatePerson(types.UserID, types.TeamType, Person) error
	}
}

func New(stream streaminterface.Publisher, models data.Models, pp mobilepay.Client) Commands {
	return Commands{
		Team:      NewTeam(stream, models.Teams),
		Payment:   NewPayment(stream, models.Teams, pp),
		Personnel: NewPersonnel(stream, models.Teams),
	}
}
