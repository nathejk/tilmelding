package main

import (
	"errors"
	"log"
	"net/http"

	"github.com/davecgh/go-spew/spew"
	"github.com/nathejk/shared-go/types"
	jsonapi "nathejk.dk/cmd/api/app"
	"nathejk.dk/internal/data"
	"nathejk.dk/internal/payment/mobilepay"
	"nathejk.dk/nathejk/table/klan"
	"nathejk.dk/nathejk/table/order"
)

func (app *application) showKlanHandler(w http.ResponseWriter, r *http.Request) {
	teamID := types.TeamID(app.ReadNamedParam(r, "id"))
	if teamID == "" {
		app.NotFoundResponse(w, r)
		return
	}
	team, err := app.models.Teams.GetKlan(teamID)
	if err != nil {
		log.Printf("GetKlan %q", err)
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.NotFoundResponse(w, r)
		default:
			app.ServerErrorResponse(w, r, err)
		}
		return
	}

	members, _, err := app.models.Members.GetSeniore(data.Filters{TeamID: teamID})
	if err != nil {
		log.Printf("GetSenior %q", err)
	}

	config := app.buildTeamConfig(r.Context(), "participation.klan", 1, 4)
	//contact, _ := app.models.Teams.GetContact(teamId)

	// Re-derive the open order's lines from the current member projection
	// on every GET. This makes the page self-healing against any drift
	// between the order and the projection: orphan lines (member removed
	// after the line was created) are cleared, missing lines (member
	// added but never billed) are added, and t-shirt size changes are
	// reflected. SetDerivedLines is idempotent for unchanged input.
	openOrder, paidOrders := app.loadOrders(r.Context(), types.TeamTypeKlan, string(teamID))
	desired := derivedLinesForKlanSeniore(members)
	if openOrder == nil && len(desired) > 0 {
		if o, err := app.commands.Order.EnsureOpenOrder(r.Context(), types.TeamTypeKlan, string(teamID)); err == nil {
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

	err = app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"config": config, "team": team, "members": members, "order": openOrder, "paidOrders": paidOrders}, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}
func (app *application) requestSeatHandler(w http.ResponseWriter, r *http.Request) {
	teamID := types.TeamID(app.ReadNamedParam(r, "id"))
	var input struct {
		TeamName             string `json:"teamName"`
		TeamGroup            string `json:"teamGroup"`
		TeamCorps            string `json:"teamCorps"`
		RequestedMemberCount uint32 `json:"requestedMemberCount"`
	}
	if err := app.ReadJSON(w, r, &input); err != nil {
		log.Printf("ReadJSON %q", err)
		app.BadRequestResponse(w, r, err)
		return
	}
	log.Printf("before update")
	err := app.commands.Klan.Update(r.Context(), teamID, klan.UpdateCommand{
		Name:      &input.TeamName,
		GroupName: &input.TeamGroup,
		Korps:     &input.TeamCorps,
	})
	log.Printf("after update")
	if err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}

	log.Printf("before reqeust")
	reservedMemberCount, err := app.commands.Klan.RequestMemberCount(r.Context(), app.config.year, teamID, input.RequestedMemberCount)
	if err != nil {
		log.Printf("with error %#v", err)

		app.BadRequestResponse(w, r, err)
		return
	}
	log.Printf("after update")
	paymentLink := ""
	status := types.SignupStatusOnHold
	var orderEnvelope *order.Order
	if reservedMemberCount > 0 {
		status = types.SignupStatusPay
		signup, err := app.models.Signup.GetByID(r.Context(), teamID)
		if err != nil {
			app.BadRequestResponse(w, r, err)
			return
		}
		if (signup.Phone == nil) || (signup.Email == nil) {
			spew.Dump(signup)
			return
		}

		// Build derived lines for the reserved seats. We don't have member
		// IDs at this stage (members are filled in later via updateKlan)
		// so each line carries a synthetic "pending-N" MemberID that
		// satisfies the commander's required-MemberID rule. updateKlan
		// later replaces these with member-keyed lines (the snapshot
		// DELETE+INSERT in the projector cleanly swaps them).
		desired := make([]order.DesiredLine, 0, reservedMemberCount)
		for i := uint32(0); i < reservedMemberCount; i++ {
			placeholder := pendingMemberID(i + 1)
			desired = append(desired, order.DesiredLine{
				LineID:     reservationLineID(i),
				ProductSKU: "participation.klan",
				MemberID:   placeholder,
				Quantity:   1,
			})
		}

		o, err := app.commands.Order.EnsureOpenOrder(r.Context(), types.TeamTypeKlan, string(teamID))
		if err != nil {
			app.ServerErrorResponse(w, r, err)
			return
		}
		o, err = app.commands.Order.SetDerivedLines(r.Context(), o.OrderID, desired)
		if err != nil {
			app.BadRequestResponse(w, r, err)
			return
		}
		orderEnvelope = o

		if o.DueAmount > 0 {
			amount := mobilepay.Amount{Value: int64(o.DueAmount), Currency: mobilepay.Currency(types.CurrencyDKK)}
			teamUrl := "https://tilmelding.nathejk.dk/klan/" + string(teamID)
			paymentLink, _ = app.commands.Payment.Request(amount, "Nathejk tilmelding", *signup.Phone, *signup.Email, teamUrl, o.OrderID, "order")
		}
	}
	team, _ := app.models.Teams.GetKlan(teamID)
	err = app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"team": team, "status": status, "order": orderEnvelope, "paymentLink": paymentLink}, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

func (app *application) updateKlanHandler(w http.ResponseWriter, r *http.Request) {
	teamID := types.TeamID(app.ReadNamedParam(r, "id"))
	var input struct {
		Team    klan.Team `json:"team"`
		Members []struct {
			MemberID   types.MemberID     `json:"memberId"`
			Deleted    bool               `json:"deleted"`
			Name       string             `json:"name"`
			Address    string             `json:"address"`
			PostalCode string             `json:"postalCode"`
			Email      types.EmailAddress `json:"email"`
			Phone      types.PhoneNumber  `json:"phone"`
			Birthday   types.Date         `json:"birthday"`
			Vegitarian bool               `json:"vegitarian"`
			TShirtSize string             `json:"tshirtSize"`
		} `json:"members"`
	}
	if err := app.ReadJSON(w, r, &input); err != nil {
		log.Printf("ReadJSON %q", err)
		app.BadRequestResponse(w, r, err)
		return
	}

	var seniors []klan.Senior
	for _, m := range input.Members {
		diet := ""
		if m.Vegitarian {
			diet = "vegetar"
		}
		seniors = append(seniors, klan.Senior{
			MemberID:   m.MemberID,
			Deleted:    m.Deleted,
			Name:       m.Name,
			Address:    m.Address,
			PostalCode: m.PostalCode,
			Email:      m.Email,
			Phone:      m.Phone,
			Birthday:   m.Birthday,
			TShirtSize: m.TShirtSize,
			Diet:       diet,
		})
	}
	_, err := app.models.Teams.GetKlan(teamID)
	if err != nil {
		log.Printf("Signup.GetByID  %q", err)
		app.BadRequestResponse(w, r, err)
		return
	}
	err = app.commands.Klan.UpdateMembers(r.Context(), teamID, input.Team, seniors)
	if err != nil {
		log.Printf("UpdateKlan  %q", err)
		app.BadRequestResponse(w, r, err)
		return
	}

	// Project members into derived order lines (one participation +
	// optional t-shirt per active senior, keyed by memberId). This
	// supersedes any reservation-only lines created earlier in
	// requestSeatHandler — same SetDerivedLines call replaces them.
	desired := derivedLinesForKlan(seniors)

	o, err := app.commands.Order.EnsureOpenOrder(r.Context(), types.TeamTypeKlan, string(teamID))
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
	log.Printf("klan order %s total=%d paid=%d due=%d", o.OrderID, o.TotalAmount, o.PaidAmount, o.DueAmount)

	paymentLink := ""
	if o.DueAmount > 0 {
		signup, _ := app.models.Signup.GetByID(r.Context(), teamID)

		phone := types.PhoneNumber("")
		if (signup != nil) && (signup.Phone != nil) {
			phone = *signup.Phone
		}

		email := types.EmailAddress("")
		if (signup != nil) && (signup.Email != nil) {
			email = *signup.Email
		}
		amount := mobilepay.Amount{Value: int64(o.DueAmount), Currency: mobilepay.Currency(types.CurrencyDKK)}
		teamUrl := "https://tilmelding.nathejk.dk/klan/" + string(teamID)

		paymentLink, _ = app.commands.Payment.Request(amount, "Nathejk tilmelding", phone, email, teamUrl, o.OrderID, "order")
	}
	team, _ := app.models.Teams.GetKlan(teamID)
	err = app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"team": team, "order": o, "paymentLink": paymentLink}, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// derivedLinesForKlan is the klan equivalent of derivedLinesForPatrulje:
// one participation + optional t-shirt per active senior, keyed on
// memberId so subsequent saves diff cleanly.
func derivedLinesForKlan(seniors []klan.Senior) []order.DesiredLine {
	lines := make([]order.DesiredLine, 0, len(seniors)*2)
	for _, s := range seniors {
		if s.Deleted {
			continue
		}
		lines = append(lines, order.DesiredLine{
			ProductSKU: "participation.klan",
			MemberID:   string(s.MemberID),
			Quantity:   1,
		})
		if s.TShirtSize != "" {
			lines = append(lines, order.DesiredLine{
				ProductSKU: "tshirt.adult",
				MemberID:   string(s.MemberID),
				Quantity:   1,
				Attributes: map[string]any{"size": s.TShirtSize},
			})
		}
	}
	return lines
}

// derivedLinesForKlanSeniore is the read-path variant of derivedLinesForKlan.
// It works with the []*data.Senior slice returned by GetSeniore (the show
// handler) rather than the []klan.Senior used by the update handler.
func derivedLinesForKlanSeniore(members []*data.Senior) []order.DesiredLine {
	lines := make([]order.DesiredLine, 0, len(members)*2)
	for _, s := range members {
		lines = append(lines, order.DesiredLine{
			ProductSKU: "participation.klan",
			MemberID:   string(s.MemberID),
			Quantity:   1,
		})
		if s.TShirtSize != "" {
			lines = append(lines, order.DesiredLine{
				ProductSKU: "tshirt.adult",
				MemberID:   string(s.MemberID),
				Quantity:   1,
				Attributes: map[string]any{"size": s.TShirtSize},
			})
		}
	}
	return lines
}

// klanLinesNeedSync was the per-handler diff helper; it has moved to
// orders.go as derivedLinesNeedSync, shared with the patrulje and
// personnel show handlers.

// reservationLineID is the deterministic LineID used for the placeholder
// klan participation lines created by requestSeatHandler before any
// senior identities are known. Using a separate prefix means that when
// updateKlanHandler later runs and emits memberId-keyed lines, the
// snapshot DELETE+INSERT in the projector cleanly replaces these with the
// real per-senior lines.
func reservationLineID(i uint32) string {
	return "derived:participation.klan:reservation-" + uintToStr(i)
}

// pendingMemberID is the synthetic MemberID stamped on klan reservation
// placeholder lines before any senior identities are known. Using a
// stable, recognisable prefix ("pending-") satisfies the commander's
// required-MemberID rule and makes the placeholder nature obvious in
// reports built off order_line.memberId. updateKlanHandler later
// supersedes these with real senior IDs via SetDerivedLines.
func pendingMemberID(i uint32) string {
	return "pending-" + uintToStr(i)
}

func uintToStr(i uint32) string {
	// Tiny helper so we don't pull strconv into klan.go just for this.
	if i == 0 {
		return "0"
	}
	var digits [10]byte
	pos := len(digits)
	for i > 0 {
		pos--
		digits[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(digits[pos:])
}
