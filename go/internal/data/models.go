package data

import (
	"context"
	"database/sql"
	"errors"

	"github.com/nathejk/shared-go/tables/crewmember"
	"github.com/nathejk/shared-go/tables/klan"
	"github.com/nathejk/shared-go/tables/order"
	"github.com/nathejk/shared-go/tables/patrulje"
	"github.com/nathejk/shared-go/tables/payment"
	"github.com/nathejk/shared-go/tables/product"
	"github.com/nathejk/shared-go/tables/section"
	"github.com/nathejk/shared-go/tables/signup"
	"github.com/nathejk/shared-go/types"
	"nathejk.dk/nathejk/table/personnel"
)

var (
	ErrRecordNotFound = errors.New("record not found")
	ErrEditConflict   = errors.New("edit conflict")
)

type PaymentInterface interface {
	GetAll(types.TeamID) ([]payment.Payment, payment.Metadata, error)
	GetByReference(string) (*payment.Payment, error)
	AmountPaidByTeamID(types.TeamID) int
}
type PersonnelInterface interface {
	GetAll(context.Context, personnel.Filter) ([]personnel.Staff, error)
	GetByID(context.Context, types.UserID) (*personnel.Staff, error)
}
type PatruljeInterface interface {
	GetAll(context.Context, patrulje.Filter) ([]*patrulje.Patrulje, error)
	GetByID(context.Context, types.TeamID) (*patrulje.Patrulje, error)
}

type Models struct {
	Teams interface {
		GetPatrulje(types.TeamID) (*Patrulje, error)
		GetKlan(types.TeamID) (*Klan, error)
		GetContact(types.TeamID) (*Contact, error)
	}
	Members interface {
		GetSpejdere(Filters) ([]*Spejder, Metadata, error)
		GetSeniore(Filters) ([]*Senior, Metadata, error)
	}
	Payment    PaymentInterface
	Personnel  PersonnelInterface
	Patrulje   PatruljeInterface
	Signup     signup.Queries
	Klan       klan.Queries
	Order      order.Queries
	Product    product.Queries
	Section    section.Queries
	Crewmember crewmember.Queries
}

func NewModels(db *sql.DB, payment PaymentInterface, personnel PersonnelInterface, patrulje PatruljeInterface, s signup.Queries, k klan.Queries, o order.Queries, pr product.Queries, sec section.Queries, cm crewmember.Queries) Models {
	return Models{
		Teams:      TeamModel{DB: db},
		Members:    MemberModel{DB: db},
		Payment:    payment,
		Personnel:  personnel,
		Patrulje:   patrulje,
		Signup:     s,
		Klan:       k,
		Order:      o,
		Product:    pr,
		Section:    sec,
		Crewmember: cm,
	}
}
