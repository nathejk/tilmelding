package data

import (
	"context"
	"database/sql"

	"github.com/nathejk/shared-go/tables"
	"github.com/nathejk/shared-go/tables/crewmember"
	"github.com/nathejk/shared-go/tables/klan"
	"github.com/nathejk/shared-go/tables/order"
	"github.com/nathejk/shared-go/tables/patrulje"
	"github.com/nathejk/shared-go/tables/payment"
	"github.com/nathejk/shared-go/tables/product"
	"github.com/nathejk/shared-go/tables/section"
	"github.com/nathejk/shared-go/tables/senior"
	"github.com/nathejk/shared-go/tables/signup"
	"github.com/nathejk/shared-go/tables/spejder"
	"github.com/nathejk/shared-go/types"
	"nathejk.dk/nathejk/table/personnel"
)

// Aliases of the module-wide sentinels in github.com/nathejk/shared-go/tables.
//
// These MUST stay aliases, not fresh errors.New values. The entity queriers in
// shared-go (and the local personnel one) return tables.ErrRecordNotFound, so a
// distinct copy here makes errors.Is(err, data.ErrRecordNotFound) silently false
// for every one of them — which turned "not found" into a 500 instead of a 404
// in the crew, personnel, signup and payment show handlers.
var (
	ErrRecordNotFound = tables.ErrRecordNotFound
	ErrEditConflict   = tables.ErrEditConflict
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

// SpejderInterface is the scout roster read API, satisfied by the shared-go
// spejder entity. It replaces the deprecated MemberModel.GetSpejdere.
type SpejderInterface interface {
	GetAll(context.Context, spejder.Filter) ([]*spejder.Spejder, spejder.Metadata, error)
}

// SeniorInterface is the senior roster read API, satisfied by the shared-go
// senior entity. It replaces the deprecated MemberModel.GetSeniore.
//
// Declared here rather than imported because the senior package, unlike its
// siblings, does not export a Queries interface of its own.
type SeniorInterface interface {
	GetAll(context.Context, senior.Filter) ([]*senior.Senior, error)
}

// Models is the read-side facade handed to the HTTP handlers.
//
// Deprecated, and nearly gone: Teams is the last member backed by this
// package's own SQL. Everything else is a querier from the entity that owns the
// data — shared-go/tables/* (or nathejk/table/personnel, not yet shared). New
// reads belong on an entity querier, not here.
//
// What is left and why: Teams.GetContact is blocked upstream — the equivalent
// patrulje.querier.GetContact is commented out in shared-go, so there is
// nothing to migrate to yet.
type Models struct {
	Teams interface {
		GetContact(types.TeamID) (*Contact, error)
	}
	Payment    PaymentInterface
	Personnel  PersonnelInterface
	Patrulje   PatruljeInterface
	Spejder    SpejderInterface
	Senior     SeniorInterface
	Signup     signup.Queries
	Klan       klan.Queries
	Order      order.Queries
	Product    product.Queries
	Section    section.Queries
	Crewmember crewmember.Queries
}

func NewModels(db *sql.DB, payment PaymentInterface, personnel PersonnelInterface, patrulje PatruljeInterface, sp SpejderInterface, sn SeniorInterface, s signup.Queries, k klan.Queries, o order.Queries, pr product.Queries, sec section.Queries, cm crewmember.Queries) Models {
	return Models{
		Teams:      TeamModel{DB: db},
		Payment:    payment,
		Personnel:  personnel,
		Patrulje:   patrulje,
		Spejder:    sp,
		Senior:     sn,
		Signup:     s,
		Klan:       k,
		Order:      o,
		Product:    pr,
		Section:    sec,
		Crewmember: cm,
	}
}
