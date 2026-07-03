package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/nathejk/shared-go/types"
	jsonapi "nathejk.dk/cmd/api/app"
	"nathejk.dk/internal/data"
	"nathejk.dk/internal/payment/mobilepay"
	"nathejk.dk/nathejk/table/crewmember"
	"nathejk.dk/nathejk/table/order"
	"nathejk.dk/nathejk/table/section"
)

// crewMemberView is the wire shape for a crew member on the /api/crew
// endpoints. It flattens the projection plus the t-shirt size (which lives
// inside additionals) into the keys the crew form binds to, and exposes
// additionals as a parsed object rather than the raw JSON string stored in
// the table.
type crewMemberView struct {
	UserID      types.UserID       `json:"userId"`
	Name        string             `json:"name"`
	Phone       types.PhoneNumber  `json:"phone"`
	Email       types.EmailAddress `json:"email"`
	Number      string             `json:"number"` // medlemsnummer
	Group       string             `json:"group"`
	Korps       types.CorpsSlug    `json:"korps"`
	Diet        string             `json:"diet"`
	TshirtSize  string             `json:"tshirtSize"`
	SectionSlug types.Slug         `json:"sectionSlug"`
	Additionals map[string]any     `json:"additionals"`
}

// tshirtSizeKey is where the crew form's t-shirt selection is persisted
// inside the crewmember additionals blob. crewmember has no dedicated
// column for it, and it only matters for deriving the order line.
const tshirtSizeKey = "tshirtSize"

// crewOwnerType / crewParticipationSKU are the order dimensions for a crew
// member. Crew is always a single-person owner keyed by the crew member's
// userId.
var (
	crewOwnerType        = types.TeamTypeCrew
	crewParticipationSKU = "participation.crew"
)

func (app *application) showCrewHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := types.UserID(app.ReadNamedParam(r, "id"))
	if userID == "" {
		app.NotFoundResponse(w, r)
		return
	}
	member, err := app.models.Crewmember.GetByID(ctx, userID)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.NotFoundResponse(w, r)
		default:
			app.ServerErrorResponse(w, r, err)
		}
		return
	}

	config := app.buildTeamConfig(ctx, crewParticipationSKU, 1, 1)
	sections := app.firstLevelSections(ctx)
	view := crewMemberToView(member)

	// Re-derive the open order's lines from the current crew member on
	// every GET so the Betalinger section is self-healing and rendered on
	// the first visit. Same pattern as the patrulje / klan / personnel
	// show handlers.
	openOrder, paidOrders := app.loadOrders(ctx, crewOwnerType, string(userID))
	desired := derivedLinesForCrew(userID, view.TshirtSize)
	if openOrder == nil && len(desired) > 0 {
		if o, err := app.commands.Order.EnsureOpenOrder(ctx, crewOwnerType, string(userID)); err == nil {
			openOrder = o
		}
	}
	if openOrder != nil && derivedLinesNeedSync(openOrder, desired) {
		if o, err := app.setDerivedLinesAfterCreate(ctx, openOrder.OrderID, desired); err == nil {
			openOrder = o
		} else {
			log.Printf("setDerivedLinesAfterCreate %s: %v", openOrder.OrderID, err)
		}
	}

	err = app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{
		"config":     config,
		"sections":   sections,
		"member":     view,
		"order":      openOrder,
		"paidOrders": paidOrders,
	}, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

func (app *application) updateCrewHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := types.UserID(app.ReadNamedParam(r, "id"))
	if userID == "" {
		app.NotFoundResponse(w, r)
		return
	}
	var input struct {
		Member crewMemberView `json:"member"`
	}
	if err := app.ReadJSON(w, r, &input); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}

	// Fold the t-shirt size into additionals so it survives on the
	// crewmember row (which has no dedicated t-shirt column).
	additionals := input.Member.Additionals
	if additionals == nil {
		additionals = map[string]any{}
	}
	if input.Member.TshirtSize != "" {
		additionals[tshirtSizeKey] = input.Member.TshirtSize
	} else {
		delete(additionals, tshirtSizeKey)
	}

	if err := app.commands.Crewmember.Update(ctx, app.config.year, userID, crewmember.UpdateFields{
		Name:        input.Member.Name,
		Phone:       input.Member.Phone,
		Email:       input.Member.Email,
		MedlemNr:    input.Member.Number,
		Group:       input.Member.Group,
		Corps:       input.Member.Korps,
		Diet:        input.Member.Diet,
		Additionals: additionals,
	}); err != nil {
		log.Printf("crewmember.Update %q", err)
		app.BadRequestResponse(w, r, err)
		return
	}

	// Section assignment travels on its own event; empty slug unassigns.
	if err := app.commands.Crewmember.AssignSection(ctx, app.config.year, userID, input.Member.SectionSlug); err != nil {
		log.Printf("crewmember.AssignSection %q", err)
		app.BadRequestResponse(w, r, err)
		return
	}

	// Recompute the order (participation + optional t-shirt).
	desired := derivedLinesForCrew(userID, input.Member.TshirtSize)
	o, err := app.commands.Order.EnsureOpenOrder(ctx, crewOwnerType, string(userID))
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}
	o, err = app.setDerivedLinesAfterCreate(ctx, o.OrderID, desired)
	if err != nil {
		log.Printf("crew setDerivedLines %q", err)
		app.ServerErrorResponse(w, r, err)
		return
	}

	paymentLink := ""
	if o.DueAmount > 0 {
		amount := mobilepay.Amount{Value: int64(o.DueAmount), Currency: mobilepay.Currency(types.CurrencyDKK)}
		teamURL := "https://tilmelding.nathejk.dk/crew/" + string(userID)
		paymentLink, _ = app.commands.Payment.Request(amount, "Nathejk crewtilmelding", input.Member.Phone, input.Member.Email, teamURL, o.OrderID, "order")
	}

	updated, err := app.models.Crewmember.GetByID(ctx, userID)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}
	err = app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{
		"member":      crewMemberToView(updated),
		"order":       o,
		"paymentLink": paymentLink,
	}, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// listSectionsHandler returns the first-level (top) sections for the active
// year, used to populate the crew form's function/section selector. Nested
// sub-sections are intentionally excluded.
func (app *application) listSectionsHandler(w http.ResponseWriter, r *http.Request) {
	sections := app.firstLevelSections(r.Context())
	if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"sections": sections}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// sectionOption is the compact {slug,label} shape returned to the frontend.
type sectionOption struct {
	Slug  types.Slug `json:"slug"`
	Label string     `json:"label"`
}

// firstLevelSections returns the top-level sections (parentSlug == "") for
// the active year, ordered by the section table's own sort order. Errors are
// logged and yield an empty list so the crew page still renders.
func (app *application) firstLevelSections(ctx context.Context) []sectionOption {
	all, err := app.models.Section.GetAll(ctx, section.Filter{YearSlug: app.config.year})
	if err != nil {
		log.Printf("section.GetAll %q", err)
		return []sectionOption{}
	}
	out := []sectionOption{}
	for _, s := range all {
		if s.ParentSlug != "" {
			continue
		}
		out = append(out, sectionOption{Slug: s.Slug, Label: s.Label})
	}
	return out
}

// derivedLinesForCrew builds the order lines a crew member should hold: one
// participation line plus a t-shirt line when a size is chosen. Keyed by the
// crew member's userId so SetDerivedLines diffs cleanly.
func derivedLinesForCrew(userID types.UserID, tshirtSize string) []order.DesiredLine {
	lines := []order.DesiredLine{{
		ProductSKU: crewParticipationSKU,
		MemberID:   string(userID),
		Quantity:   1,
	}}
	if tshirtSize != "" {
		lines = append(lines, order.DesiredLine{
			ProductSKU: "tshirt.adult",
			MemberID:   string(userID),
			Quantity:   1,
			Attributes: map[string]any{"size": tshirtSize},
		})
	}
	return lines
}

// crewMemberToView flattens a projection row into the wire shape, parsing the
// additionals JSON blob and lifting the t-shirt size out of it.
func crewMemberToView(m *crewmember.CrewMember) crewMemberView {
	additionals := map[string]any{}
	if m.Additionals != "" {
		_ = json.Unmarshal([]byte(m.Additionals), &additionals)
	}
	tshirt, _ := additionals[tshirtSizeKey].(string)
	return crewMemberView{
		UserID:      m.UserID,
		Name:        m.Name,
		Phone:       m.Phone,
		Email:       m.Email,
		Number:      m.MedlemNr,
		Group:       m.Group,
		Korps:       m.Corps,
		Diet:        m.Diet,
		TshirtSize:  tshirt,
		SectionSlug: m.SectionSlug,
		Additionals: additionals,
	}
}
