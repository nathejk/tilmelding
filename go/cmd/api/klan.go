package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/davecgh/go-spew/spew"
	"github.com/nathejk/shared-go/tables/klan"
	"github.com/nathejk/shared-go/tables/order"
	payments "github.com/nathejk/shared-go/tables/payment"
	"github.com/nathejk/shared-go/types"
	"nathejk.dk/internal/data"
)

// Klan team-size bounds. min is the number of members required before a team
// may pay; max is the largest roster allowed. Also fed into the show
// endpoint's TeamConfig so the UI and the server agree.
const (
	klanMinMembers = 1
	klanMaxMembers = 4
)

// Response DTOs — the klan half of the frontend/backend contract. See the file
// comment in response.go for why handlers must not marshal domain types.

// klanTeamResponse is the team as the klan page needs it.
//
// Note the `id` tag (not `teamId`): the page echoes this object straight back
// as the `team` field of PUT /api/klan/{id}, where it is read as a klan.Team —
// which keys on `teamId` and so never sees the id. That is harmless (the path
// parameter is authoritative for the team identity) and is kept as-is because
// renaming the tag would silently start filling in klan.Team.TeamID.
//
// `reservedMemberCount` is deliberately NOT part of the contract even though
// the previous envelope carried it: its only consumer — the loop padding the
// roster up to the reserved seat count — is commented out in KlanView.vue, and
// the klan entity's GetByID does not select the column. Re-add it here (and to
// the query) if the padding ever comes back.
type klanTeamResponse struct {
	ID          string `json:"id"`
	Status      string `json:"status"`
	Name        string `json:"name"`
	Group       string `json:"group"`
	Korps       string `json:"korps"`
	MemberCount int    `json:"memberCount"`
}

// klanRosterMemberResponse is a senior as read back from the projection for the
// roster. Distinct from klanMemberResponse below: the roster carries
// projection-only fields (id, teamId, city) that a write echo does not have.
//
// The projection's year, armNumber and timestamps are deliberately omitted —
// the page has no use for them.
type klanRosterMemberResponse struct {
	ID         string `json:"id"`
	MemberID   string `json:"memberId"`
	TeamID     string `json:"teamId"`
	Name       string `json:"name"`
	Address    string `json:"address"`
	PostalCode string `json:"postalCode"`
	City       string `json:"city"`
	Email      string `json:"email"`
	Phone      string `json:"phone"`
	Birthday   string `json:"birthday"`
	Diet       string `json:"diet"`
	TShirtSize string `json:"tshirtSize"`
}

// klanMemberResponse echoes a single senior back after a write. The add
// endpoint is the only place the client learns the server-issued memberId.
type klanMemberResponse struct {
	MemberID   string `json:"memberId"`
	Deleted    bool   `json:"deleted"`
	Name       string `json:"name"`
	Address    string `json:"address"`
	PostalCode string `json:"postalCode"`
	Email      string `json:"email"`
	Phone      string `json:"phone"`
	Birthday   string `json:"birthday"`
	Diet       string `json:"diet"`
	TShirtSize string `json:"tshirtSize"`
}

// showKlanResponse is the body of GET /api/klan/{id}.
type showKlanResponse struct {
	Config     teamConfigResponse         `json:"config"`
	Team       *klanTeamResponse          `json:"team"`
	Members    []klanRosterMemberResponse `json:"members"`
	Order      *orderResponse             `json:"order"`
	PaidOrders []orderResponse            `json:"paidOrders"`
}

// requestSeatResponse is the body of PUT /api/klan/{id}/request.
//
// Status is the signup status the request resulted in: PAY when seats were
// reserved, HOLD when the klan went on the waiting list. PaymentLink is empty
// unless there is something to pay right away.
type requestSeatResponse struct {
	Team        *klanTeamResponse `json:"team"`
	Status      string            `json:"status"`
	Order       *orderResponse    `json:"order"`
	PaymentLink string            `json:"paymentLink"`
}

// updateKlanResponse is the body of PUT /api/klan/{id}.
//
// PaymentLink is empty when nothing is due; PaymentError carries the
// human-readable reason payment was refused (e.g. team below the minimum size)
// rather than failing the request, because the save itself succeeded.
type updateKlanResponse struct {
	Team         *klanTeamResponse `json:"team"`
	Order        *orderResponse    `json:"order"`
	PaymentLink  string            `json:"paymentLink"`
	PaymentError string            `json:"paymentError"`
}

// klanMemberMutationResponse is the body of the member add and update
// endpoints: the member as stored, plus the recomputed open order.
type klanMemberMutationResponse struct {
	Member klanMemberResponse `json:"member"`
	Order  *orderResponse     `json:"order"`
}

// deleteKlanMemberResponse is the body of the member delete endpoint. Only the
// recomputed order is returned; the member is gone.
type deleteKlanMemberResponse struct {
	Order *orderResponse `json:"order"`
}

func newKlanTeamResponse(k *data.Klan) *klanTeamResponse {
	if k == nil {
		return nil
	}
	return &klanTeamResponse{
		ID:          string(k.ID),
		Status:      string(k.Status),
		Name:        k.Name,
		Group:       k.Group,
		Korps:       k.Korps,
		MemberCount: k.MemberCount,
	}
}

// newKlanRosterMemberResponses always returns a non-nil slice so `members`
// stays `[]` rather than `null`. The show handler ignores the roster read error
// and the page then does `data.members.map(...)` unguarded, so `null` would
// break it; `[]` renders an empty roster.
func newKlanRosterMemberResponses(in []*data.Senior) []klanRosterMemberResponse {
	out := make([]klanRosterMemberResponse, 0, len(in))
	for _, s := range in {
		if s == nil {
			continue
		}
		out = append(out, klanRosterMemberResponse{
			ID:         string(s.ID),
			MemberID:   string(s.MemberID),
			TeamID:     string(s.TeamID),
			Name:       s.Name,
			Address:    s.Address,
			PostalCode: s.PostalCode,
			City:       s.City,
			Email:      s.Email,
			Phone:      s.Phone,
			Birthday:   string(s.Birthday),
			Diet:       s.Diet,
			TShirtSize: s.TShirtSize,
		})
	}
	return out
}

func newKlanMemberResponse(s klan.Senior) klanMemberResponse {
	return klanMemberResponse{
		MemberID:   string(s.MemberID),
		Deleted:    s.Deleted,
		Name:       s.Name,
		Address:    s.Address,
		PostalCode: s.PostalCode,
		Email:      string(s.Email),
		Phone:      string(s.Phone),
		Birthday:   string(s.Birthday),
		Diet:       s.Diet,
		TShirtSize: s.TShirtSize,
	}
}

// showKlanHandler returns everything the klan page needs in one call.
//
// @Summary      Show a klan team
// @Description  Returns the server-side config (member bounds, prices, corps and t-shirt options), the team, its senior roster, the open order and any paid orders. Re-derives the open order from the senior projection on every call, so the page is self-healing against drift.
// @Tags         klan
// @Produce      json
// @Param        id   path      string  true  "Team ID"
// @Success      200  {object}  showKlanResponse
// @Failure      404  {object}  object{error=string}
// @Failure      500  {object}  object{error=string}
// @Router       /api/klan/{id} [get]
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

	err = app.WriteJSON(w, http.StatusOK, showKlanResponse{
		Config:     newTeamConfigResponse(config),
		Team:       newKlanTeamResponse(team),
		Members:    newKlanRosterMemberResponses(members),
		Order:      newOrderResponse(openOrder),
		PaidOrders: newOrderResponses(paidOrders),
	}, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// requestSeatHandler books the klan's seats, or puts it on the waiting list.
//
// @Summary      Request seats for a klan
// @Description  Saves the team details, then asks for the requested number of seats. If seats are available the klan moves to PAY and an open order with one participation line per reserved seat is created (keyed on synthetic "pending-N" member ids until the seniors are known), together with a payment link. If the global senior cap is reached the klan goes on HOLD and no order is created.
// @Tags         klan
// @Accept       json
// @Produce      json
// @Param        id    path  string  true  "Team ID"
// @Param        body  body  object{teamName=string,teamGroup=string,teamCorps=string,requestedMemberCount=int}  true  "Team details and the number of seats wanted"
// @Success      200   {object}  requestSeatResponse
// @Failure      400   {object}  object{error=string}
// @Failure      500   {object}  object{error=string}
// @Router       /api/klan/{id}/request [put]
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
			amount := payments.Amount{Value: int64(o.DueAmount), Currency: types.CurrencyDKK}
			teamUrl := app.config.baseurl + "/klan/" + string(teamID)
			paymentLink, _ = app.commands.Payment.Request(amount, "Nathejk tilmelding", *signup.Phone, *signup.Email, teamUrl, o.OrderID, "order")
		}
	}
	team, _ := app.models.Teams.GetKlan(teamID)
	err = app.WriteJSON(w, http.StatusOK, requestSeatResponse{
		Team:        newKlanTeamResponse(team),
		Status:      string(status),
		Order:       newOrderResponse(orderEnvelope),
		PaymentLink: paymentLink,
	}, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// updateKlanHandler saves the team details and re-prices the open order.
//
// @Summary      Update a klan team
// @Description  Saves the team-level fields only — seniors are managed through the dedicated member endpoints, so this can never create or delete a senior. Re-derives the open order from the senior projection and returns a payment link when there is something due and the team meets the minimum size.
// @Tags         klan
// @Accept       json
// @Produce      json
// @Param        id    path  string                    true  "Team ID"
// @Param        body  body  object{team=klan.Team}    true  "Team fields"
// @Success      200   {object}  updateKlanResponse
// @Failure      400   {object}  object{error=string}
// @Failure      500   {object}  object{error=string}
// @Router       /api/klan/{id} [put]
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
		amount := payments.Amount{Value: int64(due), Currency: types.CurrencyDKK}
		teamUrl := app.config.baseurl + "/klan/" + string(teamID)

		paymentLink, _ = app.commands.Payment.Request(amount, "Nathejk tilmelding", phone, email, teamUrl, orderID, "order")
	}
	team, _ := app.models.Teams.GetKlan(teamID)
	err = app.WriteJSON(w, http.StatusOK, updateKlanResponse{
		Team:         newKlanTeamResponse(team),
		Order:        newOrderResponse(openOrder),
		PaymentLink:  paymentLink,
		PaymentError: paymentError,
	}, nil)
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
// @Success      200    {object}  klanMemberMutationResponse
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
	if err := app.WriteJSON(w, http.StatusOK, klanMemberMutationResponse{
		Member: newKlanMemberResponse(input.Member),
		Order:  newOrderResponse(o),
	}, nil); err != nil {
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
// @Success      200       {object}  klanMemberMutationResponse
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
	if err := app.WriteJSON(w, http.StatusOK, klanMemberMutationResponse{
		Member: newKlanMemberResponse(input.Member),
		Order:  newOrderResponse(o),
	}, nil); err != nil {
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
// @Success      200       {object}  deleteKlanMemberResponse
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
	if err := app.WriteJSON(w, http.StatusOK, deleteKlanMemberResponse{Order: newOrderResponse(o)}, nil); err != nil {
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
//
// Reporting convention (task 009 decision): the placeholder approach is kept
// deliberately — deferring order creation until members are known would cost
// the reservation-time payment link, a worse UX. No report surfaces these IDs
// today. Any future "members per order" report must exclude them with
// `memberId NOT LIKE 'pending-%'`; the prefix exists precisely so it can.
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
