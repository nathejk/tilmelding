package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/davecgh/go-spew/spew"
	"github.com/nathejk/shared-go/tables"
	"github.com/nathejk/shared-go/tables/klan"
	"github.com/nathejk/shared-go/tables/order"
	payments "github.com/nathejk/shared-go/tables/payment"
	"github.com/nathejk/shared-go/tables/senior"
	"github.com/nathejk/shared-go/types"
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
	PaidOrders   []orderResponse   `json:"paidOrders"`
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

func newKlanTeamResponse(k *klan.Klan) *klanTeamResponse {
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
func newKlanRosterMemberResponses(in []*senior.Senior) []klanRosterMemberResponse {
	out := make([]klanRosterMemberResponse, 0, len(in))
	for _, s := range in {
		if s == nil {
			continue
		}
		out = append(out, klanRosterMemberResponse{
			// The projection has one member key, `memberId`; the previous
			// query copied it into both `id` and `memberId` and the page
			// reads `memberId`, so both keep coming from the same column.
			ID:         string(s.MemberID),
			MemberID:   string(s.MemberID),
			TeamID:     string(s.TeamID),
			Name:       s.Name,
			Address:    s.Address,
			PostalCode: s.PostalCode,
			City:       s.City,
			Email:      string(s.Email),
			Phone:      string(s.Phone),
			Birthday:   s.Birthday,
			Diet:       s.Diet,
			TShirtSize: s.TshirtSize,
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
// @Description  Returns the server-side config (member bounds, prices, corps, t-shirt options and the SKUs closed for sale), the team, its senior roster, the open order and any paid orders. Re-derives the open order from the senior projection on every call, so the page is self-healing against drift. Order line quantities and lineTotals may be negative: a free t-shirt size change is recorded as a zero-sum pair of lines (one negative for the size handed back, one positive for the size now wanted), so clients must sum them rather than assume positive values. `config.closedProducts` names products that may no longer be bought — clients must offer no way to buy or re-size one, and their price and sizes remain in the config only so what was already bought can be rendered. Unpaid units of a closed product are dropped from the open order, so the amount due falls accordingly; an order with a payment already in flight is left exactly as its payer saw it.
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
	team, err := app.models.Klan.GetByID(r.Context(), teamID)
	if err != nil {
		log.Printf("GetKlan %q", err)
		switch {
		case errors.Is(err, tables.ErrRecordNotFound):
			app.NotFoundResponse(w, r)
		default:
			app.ServerErrorResponse(w, r, err)
		}
		return
	}

	members, err := app.models.Senior.GetAll(r.Context(), senior.Filter{TeamIDs: []types.TeamID{teamID}})
	if err != nil {
		log.Printf("GetSenior %q", err)
	}

	config := app.buildTeamConfig(r.Context(), "participation.klan", klanMinMembers, klanMaxMembers)

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
	if openOrder != nil && app.syncNeeded(r.Context(), openOrder, desired) {
		if o, err := app.setDerivedLinesAfterCreate(r.Context(), openOrder, desired); err == nil {
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
		// SetDerivedLines directly, and deliberately not through
		// setDerivedLinesAfterCreate: these are participation seats only, so
		// there is nothing here the sellable filter could ever drop, and the
		// seat reservation must fail loudly rather than retry if the order is
		// not there. Any merchandise added to this path would need the filter.
		o, err = app.commands.Order.SetDerivedLines(r.Context(), o.OrderID, desired)
		if err != nil {
			app.BadRequestResponse(w, r, err)
			return
		}
		orderEnvelope = o

		if o.DueAmount > 0 {
			// Gated like every other Payment.Request call site, so the rule has no
			// exceptions to remember. These are participation seats only, so it
			// cannot refuse in practice — but a merchandise line reaching this
			// path must not become a payment request.
			if ok, _ := app.chargeable(o); ok {
				amount := payments.Amount{Value: int64(o.DueAmount), Currency: types.CurrencyDKK}
				teamUrl := app.config.baseurl + "/klan/" + string(teamID)
				paymentLink, _ = app.commands.Payment.Request(payments.Charge{
					Amount:      amount,
					Description: "Nathejk tilmelding",
					Phone:       *signup.Phone,
					Email:       *signup.Email,
					ReturnUrl:   teamUrl,
					OrderID:     o.OrderID,
					// The reserved seats, so the wallet receipt reads
					// "Senior-deltagelse ×N" rather than a bare total.
					Lines: paymentLinesFromOrder(o),
				})
			}
		}
	}
	team, _ := app.models.Klan.GetByID(r.Context(), teamID)
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
// @Description  Saves the team-level fields only — seniors are managed through the dedicated member endpoints, so this can never create or delete a senior. Re-derives the open order from the senior projection and returns a payment link when there is something due and the team meets the minimum size. No link is issued while the open order still holds a line for a product closed for sale (only possible when a payment is already in flight against it): `paymentLink` is empty and `paymentError` says so. When settle=true and the resulting order costs nothing but is not empty, the order is frozen into the paid history and `order` comes back null with `paidOrders` refreshed.
// @Tags         klan
// @Accept       json
// @Produce      json
// @Param        id    path  string                    true  "Team ID"
// @Param        body  body  object{team=klan.Team,settle=bool}    true  "Team fields. Set settle=true on an explicit user save: it allows an order that costs nothing (a free t-shirt size change, recorded as a zero-sum pair of lines) to be frozen into the paid history. Omit it for background recomputes."
// @Success      200   {object}  updateKlanResponse
// @Failure      400   {object}  object{error=string}
// @Failure      500   {object}  object{error=string}
// @Router       /api/klan/{id} [put]
func (app *application) updateKlanHandler(w http.ResponseWriter, r *http.Request) {
	teamID := types.TeamID(app.ReadNamedParam(r, "id"))
	var input struct {
		Team klan.Team `json:"team"`
		// Settle marks this PUT as the user's explicit save, which is what
		// allows a free order to be frozen. See updatePersonnelHandler.
		Settle bool `json:"settle"`
	}
	if err := app.ReadJSON(w, r, &input); err != nil {
		log.Printf("ReadJSON %q", err)
		app.BadRequestResponse(w, r, err)
		return
	}
	_, err := app.models.Klan.GetByID(r.Context(), teamID)
	if err != nil {
		log.Printf("Signup.GetByID  %q", err)
		app.BadRequestResponse(w, r, err)
		return
	}
	// Team + status only. Seniors are managed through the dedicated member
	// endpoints, so this save can never create or delete a senior identity.
	// It also carries no t-shirt size — the size lock lives on the member
	// endpoints, and the derived lines below read sizes from the projection
	// rather than from this body.
	err = app.commands.Klan.UpdateTeam(r.Context(), teamID, input.Team)
	if err != nil {
		log.Printf("UpdateKlan  %q", err)
		app.BadRequestResponse(w, r, err)
		return
	}

	// Re-derive the open order from the current senior projection (self-heal).
	members, err := app.models.Senior.GetAll(r.Context(), senior.Filter{TeamIDs: []types.TeamID{teamID}})
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
	if openOrder != nil && app.syncNeeded(r.Context(), openOrder, desired) {
		if o, err := app.setDerivedLinesAfterCreate(r.Context(), openOrder, desired); err == nil {
			openOrder = o
		} else {
			log.Printf("setDerivedLinesAfterCreate %s: %v", openOrder.OrderID, err)
		}
	}

	if input.Settle {
		openOrder = app.settleIfFree(r.Context(), openOrder)
	}

	due := 0
	orderID := ""
	if openOrder != nil {
		due = openOrder.DueAmount
		orderID = openOrder.OrderID
	}

	paymentLink := ""
	paymentError := ""
	chargeable, chargeRefusal := app.chargeable(openOrder)
	switch {
	case due <= 0:
		// nothing to pay
	case !chargeable:
		// The order still holds a line for a product that is closed for sale, so
		// no link may be issued: it would sell one. See app.chargeable.
		paymentError = chargeRefusal
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

		paymentLink, _ = app.commands.Payment.Request(payments.Charge{
			Amount:      amount,
			Description: "Nathejk tilmelding",
			Phone:       phone,
			Email:       email,
			ReturnUrl:   teamUrl,
			OrderID:     orderID,
			Lines:       paymentLinesFromOrder(openOrder),
		})
	}
	team, _ := app.models.Klan.GetByID(r.Context(), teamID)
	currentOrder, paidOrders := app.ordersForResponse(r.Context(), openOrder, types.TeamTypeKlan, string(teamID))
	err = app.WriteJSON(w, http.StatusOK, updateKlanResponse{
		Team:         newKlanTeamResponse(team),
		Order:        newOrderResponse(currentOrder),
		PaidOrders:   newOrderResponses(paidOrders),
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
	members, err := app.models.Senior.GetAll(ctx, senior.Filter{TeamIDs: []types.TeamID{teamID}})
	if err != nil {
		log.Printf("GetSeniore %q", err)
	}
	desired := replaceMemberLines(derivedLinesForKlanSeniore(members), changedMemberID, replacement)
	o, err := app.commands.Order.EnsureOpenOrder(ctx, types.TeamTypeKlan, string(teamID))
	if err != nil {
		return nil, err
	}
	return app.setDerivedLinesAfterCreate(ctx, o, desired)
}

// addKlanMemberHandler adds a single member (senior) to a klan team.
//
// @Summary      Add a member to a klan team
// @Description  Issues a server-side memberId, persists the member (one create event), recomputes the open order, and returns the created member (with its memberId) plus the order. `tshirtSize` is ignored while the year t-shirt is closed for sale: a member added then gets no shirt.
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
	if _, err := app.models.Klan.GetByID(r.Context(), teamID); err != nil {
		if errors.Is(err, tables.ErrRecordNotFound) {
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
	if members, err := app.models.Senior.GetAll(r.Context(), senior.Filter{TeamIDs: []types.TeamID{teamID}}); err == nil && len(members) >= klanMaxMembers {
		app.FailedValidationResponse(w, r, map[string]string{
			"members": fmt.Sprintf("en klan kan højst have %d seniorer", klanMaxMembers),
		})
		return
	}

	// A senior added while the shirt is closed for sale gets no shirt: there is
	// nothing stored to keep, and the sale is shut.
	app.lockKlanMemberSize(r.Context(), teamID, &input.Member)

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

// storedKlanSize is the t-shirt size the projection currently holds for a senior,
// or "" when the senior or their size is not there.
//
// Read through the roster because the senior read API exposes no by-id lookup.
// Only called while sizes are locked.
func (app *application) storedKlanSize(ctx context.Context, teamID types.TeamID, memberID types.MemberID) string {
	members, err := app.models.Senior.GetAll(ctx, senior.Filter{TeamIDs: []types.TeamID{teamID}})
	if err != nil {
		// Fail closed: an unreadable roster must not become licence to write the
		// requested size.
		log.Printf("storedKlanSize %q", err)
		return ""
	}
	for _, m := range members {
		if m != nil && m.MemberID == memberID {
			return m.TshirtSize
		}
	}
	return ""
}

// lockKlanMemberSize replaces the requested t-shirt size with the stored one while
// the shirt is closed for sale. A new senior (no stored size) gets no shirt.
func (app *application) lockKlanMemberSize(ctx context.Context, teamID types.TeamID, member *klan.Senior) {
	if !app.tshirtLocked() {
		return
	}
	stored := ""
	if member.MemberID != "" {
		stored = app.storedKlanSize(ctx, teamID, member.MemberID)
	}
	member.TShirtSize = app.lockedSize(tshirtSKU, stored, member.TShirtSize)
}

// updateKlanMemberHandler updates a single existing member (senior).
//
// @Summary      Update a klan member
// @Description  Publishes one update event for the member (never creates an identity) and recomputes the open order. `tshirtSize` is ignored while the year t-shirt is closed for sale: the stored size is persisted and returned instead, so the size of a shirt already ordered cannot change.
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

	// The stored size wins while the shirt is closed for sale, so a stale page or
	// a hand-made request cannot re-size a shirt that is already being printed.
	// Applied before the update is published, and the response carries the value
	// that was actually persisted.
	app.lockKlanMemberSize(r.Context(), teamID, &input.Member)

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
// It works with the []*senior.Senior slice returned by the senior querier (the
// show handler) rather than the []klan.Senior used by the update handler.
func derivedLinesForKlanSeniore(members []*senior.Senior) []order.DesiredLine {
	lines := make([]order.DesiredLine, 0, len(members)*2)
	for _, s := range members {
		lines = append(lines, order.DesiredLine{
			ProductSKU: "participation.klan",
			MemberID:   string(s.MemberID),
			Quantity:   1,
		})
		if s.TshirtSize != "" {
			lines = append(lines, order.DesiredLine{
				ProductSKU: "tshirt.adult",
				MemberID:   string(s.MemberID),
				Quantity:   1,
				Attributes: map[string]any{"size": s.TshirtSize},
			})
		}
	}
	return lines
}

// klanLinesNeedSync was the per-handler diff helper; the comparison now
// lives in shared-go as order.Commands.SyncNeeded, reached through
// app.syncNeeded in orders.go and shared with the patrulje, crew and
// personnel handlers.

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
