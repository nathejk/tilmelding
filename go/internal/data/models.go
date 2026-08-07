package data

import (
	"context"

	"github.com/nathejk/shared-go/tables"
	"github.com/nathejk/shared-go/tables/crewmember"
	"github.com/nathejk/shared-go/tables/klan"
	"github.com/nathejk/shared-go/tables/order"
	"github.com/nathejk/shared-go/tables/patrulje"
	"github.com/nathejk/shared-go/tables/product"
	"github.com/nathejk/shared-go/tables/section"
	"github.com/nathejk/shared-go/tables/senior"
	"github.com/nathejk/shared-go/tables/signup"
	"github.com/nathejk/shared-go/tables/spejder"
	"github.com/nathejk/shared-go/types"
	"nathejk.dk/nathejk/table/payment"
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

// PersonnelInterface is the staff read API. Declared here because the personnel
// entity is still local and exports no Queries interface of its own.
type PersonnelInterface interface {
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
// Every member is now a querier from the entity that owns the data —
// shared-go/tables/* (or nathejk/table/personnel, not yet shared). This package
// holds no SQL of its own any more; it is only the wiring and the error
// aliases, and new reads belong on an entity querier.
//
// The last local query, TeamModel.GetContact, went when patrulje.GetByID
// started selecting the four contactXxx columns: the contact is part of the
// patrulje row, not a table of its own.
type Models struct {
	Payment    payment.Queries
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

func NewModels(payment payment.Queries, personnel PersonnelInterface, patrulje PatruljeInterface, sp SpejderInterface, sn SeniorInterface, s signup.Queries, k klan.Queries, o order.Queries, pr product.Queries, sec section.Queries, cm crewmember.Queries) Models {
	return Models{
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
