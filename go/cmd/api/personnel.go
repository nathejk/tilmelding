package main

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/nathejk/shared-go/types"
	jsonapi "nathejk.dk/cmd/api/app"
	"nathejk.dk/internal/data"
	"nathejk.dk/internal/payment/mobilepay"
	"nathejk.dk/nathejk/table/order"
	"nathejk.dk/nathejk/table/personnel"
)

func (app *application) showPersonnelHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	userID := types.UserID(app.ReadNamedParam(r, "id"))
	if userID == "" {
		app.NotFoundResponse(w, r)
		return
	}
	person, err := app.models.Personnel.GetByID(ctx, userID)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.NotFoundResponse(w, r)
		default:
			app.ServerErrorResponse(w, r, err)
		}
		return
	}
	config := app.buildTeamConfig(r.Context(), participationSKUForPerson(person), 1, 1)
	//contact, _ := app.models.Teams.GetContact(teamId)

	// Re-derive the open order's lines from the current person record on
	// every GET so the page is self-healing against any drift between the
	// order and the projection (e.g. a t-shirt size edited out of band).
	// Also creates the order on the first visit for users who signed up
	// before the order system was introduced. SetDerivedLines is
	// idempotent for unchanged input — see derivedLinesNeedSync.
	openOrder, paidOrders := app.loadOrders(r.Context(), personnelOrderOwnerType(person), string(userID))
	desired := derivedLinesForPersonnel(person, personnel.Person{TshirtSize: person.TshirtSize})
	if openOrder == nil && len(desired) > 0 {
		if o, err := app.commands.Order.EnsureOpenOrder(r.Context(), personnelOrderOwnerType(person), string(userID)); err == nil {
			openOrder = o
		}
	}
	if openOrder != nil && derivedLinesNeedSync(openOrder, desired) {
		if o, err := app.setDerivedLinesAfterCreate(r.Context(), openOrder.OrderID, desired); err == nil {
			openOrder = o
		} else {
			log.Printf("setDerivedLinesAfterCreate %s: %v", openOrder.OrderID, err)
		}
	}

	err = app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"config": config, "person": person, "order": openOrder, "paidOrders": paidOrders}, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}
func (app *application) updatePersonnelHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	userID := types.UserID(app.ReadNamedParam(r, "id"))
	var input struct {
		Person personnel.Person `json:"person"`
	}
	if err := app.ReadJSON(w, r, &input); err != nil {
		log.Printf("ReadJSON %q", err)
		app.BadRequestResponse(w, r, err)
		return
	}

	person, err := app.models.Personnel.GetByID(ctx, userID)
	if err != nil {
		log.Printf("Signup.GetByID  %q", err)
		app.BadRequestResponse(w, r, err)
		return
	}
	err = app.commands.Personnel.Update(ctx, userID, person.Type, input.Person)
	if err != nil {
		log.Printf("UpdatePerson  %q", err)
		app.BadRequestResponse(w, r, err)
		return
	}

	// Map (personnel.Type, personnel.Type==friend, tshirtSize) into the
	// derived order lines. There's exactly one "member" — the person
	// themselves — so each line uses the userID as the memberId attribute.
	desired := derivedLinesForPersonnel(person, input.Person)

	o, err := app.commands.Order.EnsureOpenOrder(r.Context(), personnelOrderOwnerType(person), string(userID))
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}
	o, err = app.commands.Order.SetDerivedLines(r.Context(), o.OrderID, desired)
	if err != nil {
		log.Printf("SetDerivedLines %q", err)
		app.BadRequestResponse(w, r, err)
		return
	}
	log.Printf("personnel order %s total=%d paid=%d due=%d", o.OrderID, o.TotalAmount, o.PaidAmount, o.DueAmount)

	paymentLink := ""
	if o.DueAmount > 0 {
		signup, _ := app.models.Signup.GetByID(r.Context(), types.TeamID(userID))

		phone := types.PhoneNumber("")
		if (signup != nil) && (signup.Phone != nil) {
			phone = *signup.Phone
		}

		email := types.EmailAddress("")
		if (signup != nil) && (signup.Email != nil) {
			email = *signup.Email
		}
		amount := mobilepay.Amount{Value: int64(o.DueAmount), Currency: mobilepay.Currency(types.CurrencyDKK)}
		teamUrl := "https://tilmelding.nathejk.dk/badut/" + string(userID)

		paymentLink, _ = app.commands.Payment.Request(amount, "Nathejk gøglertilmelding", phone, email, teamUrl, o.OrderID, "order")
	}
	updated, _ := app.models.Personnel.GetByID(ctx, userID)
	err = app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"person": updated, "order": o, "paymentLink": paymentLink}, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// derivedLinesForPersonnel maps a single crew/gøgler person into the
// derived order lines they should hold:
//
//   - One participation line. Which SKU is used depends on the staff
//     record's Type:
//   - TeamTypeBadut → participation.gogler
//   - TeamTypeCrew → participation.crew
//   - One t-shirt line if the person picked a size.
//
// The legacy handler hard-coded these prices inline; here they live in
// the catalogue and are picked by SKU.
func derivedLinesForPersonnel(staff *personnel.Staff, edited personnel.Person) []order.DesiredLine {
	participationSKU := participationSKUForPerson(staff)

	lines := []order.DesiredLine{{
		ProductSKU: participationSKU,
		MemberID:   string(staff.ID),
		Quantity:   1,
	}}

	size := edited.TshirtSize
	if size != "" {
		lines = append(lines, order.DesiredLine{
			ProductSKU: "tshirt.adult",
			MemberID:   string(staff.ID),
			Quantity:   1,
			Attributes: map[string]any{"size": size},
		})
	}
	return lines
}

// participationSKUForPerson maps a personnel record to its participation
// SKU. Two personnel flavours, two SKUs:
//
//   - badut (gøgler) → participation.gogler
//   - crew (everything else) → participation.crew
//
// Pre-rename the codebase carried three flavours (staff, friend,
// staff.friend); they were collapsed into a single "crew" SKU. Legacy
// personnel rows with Type=="staff" or Type=="friend" are migrated to
// Type=="crew" by the projection (see personnel/consumer.go) so this
// function only needs to handle the post-rename world. As a defensive
// fallback any unknown Type is treated as crew.
func participationSKUForPerson(s *personnel.Staff) string {
	if s.Type == types.TeamTypeBadut {
		return "participation.gogler"
	}
	return "participation.crew"
}

// personnelOrderOwnerType returns the order ownerType to use for a
// personnel record. The order entity recognises four owner types
// (patrulje, klan, crew, badut). Personnel rows are either crew or
// badut; anything unexpected is treated as crew so we don't end up with
// orphan ownerTypes the runtime can't look up.
func personnelOrderOwnerType(s *personnel.Staff) types.TeamType {
	if s.Type == types.TeamTypeBadut {
		return types.TeamTypeBadut
	}
	return types.TeamTypeCrew
}
