package main

import (
	"context"
	"errors"
	"fmt"
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

// Klan team-size bounds. min is the number of members required before a team
// may pay; max is the largest roster allowed. Also fed into the show
// endpoint's TeamConfig so the UI and the server agree.
const (
	klanMinMembers = 1
	klanMaxMembers = 4
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

	config := app.buildTeamConfig(r.Context(), "participation.klan", klanMinMembers, klanMaxMembers)
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
	if openOrder != nil && app.derivedLinesNeedSync(r.Context(), openOrder, desired) {
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
		Team klan.Team `json:"team"`
	}
	if err := app.ReadJSON(w, r, &input); err != nil {
		log.Printf("ReadJSON %q", err)
		app.BadRequestResponse(w, r, err)
		return
	}
	_, err := app.models.Teams.GetKlan(teamID)
	if err != nil {
		log.Printf("Signup.GetByID  %q", err)
		app.BadRequestResponse(w, r, err)
		return
	}
	// Team + status only. Seniors are managed through the dedicated member
	// endpoints, so this save can never create or delete a senior identity.
	err = app.commands.Klan.UpdateTeam(r.Context(), teamID, input.Team)
	if err != nil {
		log.Printf("UpdateKlan  %q", err)
		app.BadRequestResponse(w, r, err)
		return
	}

	// Re-derive the open order from the current senior projection (self-heal).
	members, _, err := app.models.Members.GetSeniore(data.Filters{TeamID: teamID})
	if err != nil {
		log.Printf("GetSeniore %q", err)
	}
	desired := derivedLinesForKlanSeniore(members)
	openOrder, _ := app.loadOrders(r.Context(), types.TeamTypeKlan, string(teamID))
	if openOrder == nil && len(desired) > 0 {
		if o, err := app.commands.Order.EnsureOpenOrder(r.Context(), types.TeamTypeKlan, string(teamID)); err == nil {
			openOrder = o
		}
	}
	if openOrder != nil && app.derivedLinesNeedSync(r.Context(), openOrder, desired) {
		if o, err := app.setDerivedLinesAfterCreate(r.Context(), openOrder.OrderID, desired); err == nil {
			openOrder = o
		} else {
			log.Printf("setDerivedLinesAfterCreate %s: %v", openOrder.OrderID, err)
		}
	}

	due := 0
	orderID := ""
	if openOrder != nil {
		due = openOrder.DueAmount
		orderID = openOrder.OrderID
	}

	paymentLink := ""
	paymentError := ""
	switch {
	case due <= 0:
		// nothing to pay
	case len(members) < klanMinMembers:
		paymentError = fmt.Sprintf("en klan skal have mindst %d seniorer for at kunne betale", klanMinMembers)
	default:
		signup, _ := app.models.Signup.GetByID(r.Context(), teamID)

		phone := types.PhoneNumber("")
		if (signup != nil) && (signup.Phone != nil) {
			phone = *signup.Phone
		}

		email := types.EmailAddress("")
		if (signup != nil) && (signup.Email != nil) {
			email = *signup.Email
		}
		amount := mobilepay.Amount{Value: int64(due), Currency: mobilepay.Currency(types.CurrencyDKK)}
		teamUrl := "https://tilmelding.nathejk.dk/klan/" + string(teamID)

		paymentLink, _ = app.commands.Payment.Request(amount, "Nathejk tilmelding", phone, email, teamUrl, orderID, "order")
	}
	team, _ := app.models.Teams.GetKlan(teamID)
	err = app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"team": team, "order": openOrder, "paymentLink": paymentLink, "paymentError": paymentError}, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// rederiveKlanOrder recomputes the open order's derived lines from the current
// senior projection, with the given member's lines replaced (add/update) or
// removed (delete via a nil replacement). Mirrors rederivePatruljeOrder.
func (app *application) rederiveKlanOrder(ctx context.Context, teamID types.TeamID, changedMemberID string, replacement []order.DesiredLine) (*order.Order, error) {
	members, _, err := app.models.Members.GetSeniore(data.Filters{TeamID: teamID})
	if err != nil {
		log.Printf("GetSeniore %q", err)
	}
	desired := replaceMemberLines(derivedLinesForKlanSeniore(members), changedMemberID, replacement)
	o, err := app.commands.Order.EnsureOpenOrder(ctx, types.TeamTypeKlan, string(teamID))
	if err != nil {
		return nil, err
	}
	return app.setDerivedLinesAfterCreate(ctx, o.OrderID, desired)
}

// addKlanMemberHandler adds a single member (senior) to a klan team.
//
// @Summary      Add a member to a klan team
// @Description  Issues a server-side memberId, persists the member (one create event), recomputes the open order, and returns the created member (with its memberId) plus the order.
// @Tags         klan
// @Accept       json
// @Produce      json
// @Param        id     path  string                      true  "Team ID"
// @Param        body   body  object{member=klan.Senior}    true  "New member"
// @Success      200    {object}  object{member=klan.Senior,order=order.Order}
// @Failure      400    {object}  object{error=string}
// @Failure      404    {object}  object{error=string}
// @Router       /api/klan/{id}/member [post]
func (app *application) addKlanMemberHandler(w http.ResponseWriter, r *http.Request) {
	teamID := types.TeamID(app.ReadNamedParam(r, "id"))
	if teamID == "" {
		app.NotFoundResponse(w, r)
		return
	}
	if _, err := app.models.Teams.GetKlan(teamID); err != nil {
		if errors.Is(err, data.ErrRecordNotFound) {
			app.NotFoundResponse(w, r)
		} else {
			app.ServerErrorResponse(w, r, err)
		}
		return
	}
	var input struct {
		Member klan.Senior `json:"member"`
	}
	if err := app.ReadJSON(w, r, &input); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}

	// Enforce the team maximum server-side.
	if members, _, err := app.models.Members.GetSeniore(data.Filters{TeamID: teamID}); err == nil && len(members) >= klanMaxMembers {
		app.FailedValidationResponse(w, r, map[string]string{
			"members": fmt.Sprintf("en klan kan højst have %d seniorer", klanMaxMembers),
		})
		return
	}

	memberID, err := app.commands.Klan.AddMember(r.Context(), teamID, input.Member)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}
	input.Member.MemberID = memberID

	o, err := app.rederiveKlanOrder(r.Context(), teamID, string(memberID), derivedLinesForKlan([]klan.Senior{input.Member}))
	if err != nil {
		log.Printf("rederiveKlanOrder %q", err)
	}
	if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"member": input.Member, "order": o}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// updateKlanMemberHandler updates a single existing member (senior).
//
// @Summary      Update a klan member
// @Description  Publishes one update event for the member (never creates an identity) and recomputes the open order.
// @Tags         klan
// @Accept       json
// @Produce      json
// @Param        id        path  string                      true  "Team ID"
// @Param        memberId  path  string                      true  "Member ID"
// @Param        body      body  object{member=klan.Senior}    true  "Member fields"
// @Success      200       {object}  object{member=klan.Senior,order=order.Order}
// @Failure      400       {object}  object{error=string}
// @Failure      404       {object}  object{error=string}
// @Router       /api/klan/{id}/member/{memberId} [put]
func (app *application) updateKlanMemberHandler(w http.ResponseWriter, r *http.Request) {
	teamID := types.TeamID(app.ReadNamedParam(r, "id"))
	memberID := types.MemberID(app.ReadNamedParam(r, "memberId"))
	if teamID == "" || memberID == "" {
		app.NotFoundResponse(w, r)
		return
	}
	var input struct {
		Member klan.Senior `json:"member"`
	}
	if err := app.ReadJSON(w, r, &input); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}
	input.Member.MemberID = memberID // path is authoritative

	if err := app.commands.Klan.UpdateMember(r.Context(), teamID, input.Member); err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}
	o, err := app.rederiveKlanOrder(r.Context(), teamID, string(memberID), derivedLinesForKlan([]klan.Senior{input.Member}))
	if err != nil {
		log.Printf("rederiveKlanOrder %q", err)
	}
	if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"member": input.Member, "order": o}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// deleteKlanMemberHandler removes a single member (senior).
//
// @Summary      Delete a klan member
// @Description  Publishes one delete event, vacating the member's seat, and recomputes the open order.
// @Tags         klan
// @Produce      json
// @Param        id        path  string  true  "Team ID"
// @Param        memberId  path  string  true  "Member ID"
// @Success      200       {object}  object{order=order.Order}
// @Failure      404       {object}  object{error=string}
// @Router       /api/klan/{id}/member/{memberId} [delete]
func (app *application) deleteKlanMemberHandler(w http.ResponseWriter, r *http.Request) {
	teamID := types.TeamID(app.ReadNamedParam(r, "id"))
	memberID := types.MemberID(app.ReadNamedParam(r, "memberId"))
	if teamID == "" || memberID == "" {
		app.NotFoundResponse(w, r)
		return
	}
	if err := app.commands.Klan.DeleteMember(r.Context(), teamID, memberID); err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}
	o, err := app.rederiveKlanOrder(r.Context(), teamID, string(memberID), nil)
	if err != nil {
		log.Printf("rederiveKlanOrder %q", err)
	}
	if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"order": o}, nil); err != nil {
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
