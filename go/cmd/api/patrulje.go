package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/nathejk/shared-go/types"
	jsonapi "nathejk.dk/cmd/api/app"
	"nathejk.dk/internal/data"
	"nathejk.dk/internal/payment/mobilepay"
	"nathejk.dk/nathejk/table/order"
	"nathejk.dk/nathejk/table/patrulje"
)

type TeamConfig struct {
	MinMemberCount int               `json:"minMemberCount"`
	MaxMemberCount int               `json:"maxMemberCount"`
	MemberPrice    int               `json:"memberPrice"`
	TShirtPrice    int               `json:"tshirtPrice"`
	Korps          []types.SlugLabel `json:"korps"`
	TShirtSizes    []types.SlugLabel `json:"tshirtSizes"`
}

// tshirtSizeLabels maps the slugs the catalogue carries on tshirt.adult
// to their display labels. The catalogue is the source of truth for
// *which* sizes are offered; this map only translates a known slug into
// something a user wants to read. Adding a new slug here without adding
// it to a product Sizes list is harmless; doing the reverse silently
// shows the slug as the label.
var tshirtSizeLabels = map[string]string{
	"xs":  "X-Small",
	"s":   "Small",
	"m":   "Medium",
	"l":   "Large",
	"xl":  "X-Large",
	"xxl": "XX-Large",
	"3xl": "XXX-Large",
}

// tshirtSizesFor builds a SlugLabel list for the given size slugs, in the
// order the catalogue lists them, prefixed with the "Ingen" (no t-shirt)
// option. Slugs without a label fall back to their slug.
func tshirtSizesFor(slugs []string) []types.SlugLabel {
	out := make([]types.SlugLabel, 0, len(slugs)+1)
	out = append(out, types.SlugLabel{Slug: "", Label: "Ingen"})
	for _, s := range slugs {
		label, ok := tshirtSizeLabels[s]
		if !ok {
			label = s
		}
		out = append(out, types.SlugLabel{Slug: s, Label: label})
	}
	return out
}

// buildTeamConfig assembles a TeamConfig for the show endpoints by
// reading prices and t-shirt sizes from the product catalogue. min and
// max are the team-type-specific member-count bounds (still hard-coded
// because they're a UI concern, not a catalogue concern).
//
// participationSKU is the participation product appropriate to the
// owner type (e.g. "participation.patrulje"). The function also reads
// "tshirt.adult" for the t-shirt price and the available sizes.
//
// On any catalogue lookup error, the corresponding field is left at its
// zero value: handlers degrade to "price unknown" rather than failing
// the whole show request, which mirrors how the legacy code handled
// missing data.
func (app *application) buildTeamConfig(ctx context.Context, participationSKU string, min, max int) TeamConfig {
	cfg := TeamConfig{
		MinMemberCount: min,
		MaxMemberCount: max,
		Korps:          types.CorpsSlugs.AsObjects(),
		TShirtSizes:    []types.SlugLabel{{Slug: "", Label: "Ingen"}},
	}
	if p, err := app.models.Product.GetBySKU(ctx, app.config.year, participationSKU); err == nil && p != nil {
		cfg.MemberPrice = p.UnitPrice / 100
	}
	if t, err := app.models.Product.GetBySKU(ctx, app.config.year, "tshirt.adult"); err == nil && t != nil {
		cfg.TShirtPrice = t.UnitPrice / 100
		cfg.TShirtSizes = tshirtSizesFor(t.Sizes)
	}
	return cfg
}

func (app *application) showPatruljeHandler(w http.ResponseWriter, r *http.Request) {
	teamID := types.TeamID(app.ReadNamedParam(r, "id"))
	if teamID == "" {
		app.NotFoundResponse(w, r)
		return
	}
	team, err := app.models.Teams.GetPatrulje(teamID)
	if err != nil {
		log.Printf("GetPatrulje %q", err)
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.NotFoundResponse(w, r)
		default:
			app.ServerErrorResponse(w, r, err)
		}
		return
	}

	members, _, err := app.models.Members.GetSpejdere(data.Filters{TeamID: teamID})
	if err != nil {
		log.Printf("GetSpejdere %q", err)
	}

	config := app.buildTeamConfig(r.Context(), "participation.patrulje", 3, 7)
	contact, _ := app.models.Teams.GetContact(teamID)

	// Re-derive the open order's lines from the current member projection
	// on every GET so the page is self-healing against any drift between
	// the order and the projection. SetDerivedLines is idempotent for
	// unchanged input — see klanLinesNeedSync for the same pattern.
	openOrder, paidOrders := app.loadOrders(r.Context(), types.TeamTypePatrulje, string(teamID))
	desired := derivedLinesForPatruljeSpejdere(members)
	if openOrder == nil && len(desired) > 0 {
		if o, err := app.commands.Order.EnsureOpenOrder(r.Context(), types.TeamTypePatrulje, string(teamID)); err == nil {
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

	err = app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"config": config, "team": team, "contact": contact, "members": members, "order": openOrder, "paidOrders": paidOrders}, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

func (app *application) assignNumberHandler(w http.ResponseWriter, r *http.Request) {
	teams, _ := app.models.Patrulje.GetAll(r.Context(), patrulje.Filter{YearSlug: app.config.year})
	log.Printf("Assigning numbers to %d teams", len(teams))
	for _, team := range teams {
		if team.TeamNumber != "" {
			log.Printf("%s already got number %q", team.TeamID, team.TeamNumber)
			continue
		}

		amountPaid := app.models.Payment.AmountPaidByTeamID(team.TeamID)
		if amountPaid == 0 {
			log.Printf("%s have no registered payments", team.TeamID)
			continue
		}
		log.Printf("%s ASSIGNING NUMBER", team.TeamID)
		_ = app.commands.Patrulje.AssignNumber(r.Context(), team.TeamID)
		time.Sleep(time.Second)
		p, _ := app.models.Teams.GetPatrulje(team.TeamID)
		log.Printf("%s Got number %q", team.TeamID, p.Number)
	}
}

func (app *application) updatePatruljeHandler(w http.ResponseWriter, r *http.Request) {
	teamID := types.TeamID(app.ReadNamedParam(r, "id"))
	var input struct {
		Team    patrulje.Team      `json:"team"`
		Contact patrulje.Contact   `json:"contact"`
		Members []patrulje.Spejder `json:"members"`
	}
	if err := app.ReadJSON(w, r, &input); err != nil {
		log.Printf("ReadJSON %q", err)
		app.BadRequestResponse(w, r, err)
		return
	}
	_, err := app.models.Teams.GetPatrulje(teamID)
	if err != nil {
		log.Printf("Signup.GetByID  %q", err)
		app.BadRequestResponse(w, r, err)
		return
	}
	err = app.commands.Patrulje.Update(r.Context(), teamID, input.Team, input.Contact, input.Members)
	if err != nil {
		log.Printf("UpdatePatrulje  %q", err)
		app.BadRequestResponse(w, r, err)
		return
	}

	// Project the team form into derived order lines: one participation line
	// per active member, plus a t-shirt line per member who picked a size.
	// Lines are keyed by memberId so removing a member from the form
	// naturally removes their lines on the next SetDerivedLines.
	desired := derivedLinesForPatrulje(input.Members)

	o, err := app.commands.Order.EnsureOpenOrder(r.Context(), types.TeamTypePatrulje, string(teamID))
	if err != nil {
		log.Printf("EnsureOpenOrder %q", err)
		app.ServerErrorResponse(w, r, err)
		return
	}
	o, err = app.commands.Order.SetDerivedLines(r.Context(), o.OrderID, desired)
	if err != nil {
		log.Printf("SetDerivedLines %q", err)
		app.BadRequestResponse(w, r, err)
		return
	}
	log.Printf("order %s total=%d paid=%d due=%d", o.OrderID, o.TotalAmount, o.PaidAmount, o.DueAmount)

	paymentLink := ""
	if o.DueAmount > 0 {
		signup, _ := app.models.Signup.GetByID(r.Context(), teamID)
		if (input.Contact.Phone == "") && (signup != nil) && (signup.Phone != nil) {
			input.Contact.Phone = *signup.Phone
		}
		if (input.Contact.Email == "") && (signup != nil) && (signup.Email != nil) {
			input.Contact.Email = *signup.Email
		}
		// Order.DueAmount is already in minor units (øre), unlike the
		// legacy DKK arithmetic that needed *100.
		amount := mobilepay.Amount{Value: int64(o.DueAmount), Currency: mobilepay.Currency(types.CurrencyDKK)}
		teamUrl := "https://tilmelding.nathejk.dk/patrulje/" + string(teamID)

		paymentLink, _ = app.commands.Payment.Request(amount, "Nathejk tilmelding", input.Contact.Phone, input.Contact.Email, teamUrl, o.OrderID, "order")
	}
	team, _ := app.models.Teams.GetPatrulje(teamID)
	err = app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"team": team, "order": o, "paymentLink": paymentLink}, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// replaceMemberLines returns base with any lines belonging to memberID removed,
// then the replacement lines appended. Used to re-derive a patrulje order after
// a single-member add/update/delete without waiting for the member projection
// to catch up: the change is applied on top of the (possibly stale) projection
// so the recomputed order is correct immediately.
//
//   - add / update: replacement = the member's derived lines (participation +
//     optional t-shirt); any stale lines for that member are dropped first.
//   - delete: replacement = nil, so the member's lines are simply removed.
func replaceMemberLines(base []order.DesiredLine, memberID string, replacement []order.DesiredLine) []order.DesiredLine {
	out := make([]order.DesiredLine, 0, len(base)+len(replacement))
	for _, l := range base {
		if l.MemberID == memberID {
			continue
		}
		out = append(out, l)
	}
	return append(out, replacement...)
}

// rederivePatruljeOrder recomputes the open order's derived lines from the
// current member projection, with the given member's lines replaced (add/update)
// or removed (delete via a nil replacement). Ensures an open order exists first.
func (app *application) rederivePatruljeOrder(ctx context.Context, teamID types.TeamID, changedMemberID string, replacement []order.DesiredLine) (*order.Order, error) {
	members, _, err := app.models.Members.GetSpejdere(data.Filters{TeamID: teamID})
	if err != nil {
		log.Printf("GetSpejdere %q", err)
	}
	desired := replaceMemberLines(derivedLinesForPatruljeSpejdere(members), changedMemberID, replacement)
	o, err := app.commands.Order.EnsureOpenOrder(ctx, types.TeamTypePatrulje, string(teamID))
	if err != nil {
		return nil, err
	}
	return app.setDerivedLinesAfterCreate(ctx, o.OrderID, desired)
}

// addPatruljeMemberHandler adds a single member to a patrulje team.
//
// @Summary      Add a member to a patrulje team
// @Description  Issues a server-side memberId, persists the member (one create event), recomputes the open order, and returns the created member (with its memberId) plus the order.
// @Tags         patrulje
// @Accept       json
// @Produce      json
// @Param        id     path  string                         true  "Team ID"
// @Param        body   body  object{member=patrulje.Spejder}  true  "New member"
// @Success      200    {object}  object{member=patrulje.Spejder,order=order.Order}
// @Failure      400    {object}  object{error=string}
// @Failure      404    {object}  object{error=string}
// @Router       /api/patrulje/{id}/member [post]
func (app *application) addPatruljeMemberHandler(w http.ResponseWriter, r *http.Request) {
	teamID := types.TeamID(app.ReadNamedParam(r, "id"))
	if teamID == "" {
		app.NotFoundResponse(w, r)
		return
	}
	if _, err := app.models.Teams.GetPatrulje(teamID); err != nil {
		if errors.Is(err, data.ErrRecordNotFound) {
			app.NotFoundResponse(w, r)
		} else {
			app.ServerErrorResponse(w, r, err)
		}
		return
	}
	var input struct {
		Member patrulje.Spejder `json:"member"`
	}
	if err := app.ReadJSON(w, r, &input); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}

	memberID, err := app.commands.Patrulje.AddMember(r.Context(), teamID, input.Member)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}
	input.Member.MemberID = memberID

	o, err := app.rederivePatruljeOrder(r.Context(), teamID, string(memberID), derivedLinesForPatrulje([]patrulje.Spejder{input.Member}))
	if err != nil {
		log.Printf("rederivePatruljeOrder %q", err)
	}
	if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"member": input.Member, "order": o}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// updatePatruljeMemberHandler updates a single existing member.
//
// @Summary      Update a patrulje member
// @Description  Publishes one update event for the member (never creates an identity) and recomputes the open order.
// @Tags         patrulje
// @Accept       json
// @Produce      json
// @Param        id        path  string                         true  "Team ID"
// @Param        memberId  path  string                         true  "Member ID"
// @Param        body      body  object{member=patrulje.Spejder}  true  "Member fields"
// @Success      200       {object}  object{member=patrulje.Spejder,order=order.Order}
// @Failure      400       {object}  object{error=string}
// @Failure      404       {object}  object{error=string}
// @Router       /api/patrulje/{id}/member/{memberId} [put]
func (app *application) updatePatruljeMemberHandler(w http.ResponseWriter, r *http.Request) {
	teamID := types.TeamID(app.ReadNamedParam(r, "id"))
	memberID := types.MemberID(app.ReadNamedParam(r, "memberId"))
	if teamID == "" || memberID == "" {
		app.NotFoundResponse(w, r)
		return
	}
	var input struct {
		Member patrulje.Spejder `json:"member"`
	}
	if err := app.ReadJSON(w, r, &input); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}
	input.Member.MemberID = memberID // path is authoritative

	if err := app.commands.Patrulje.UpdateMember(r.Context(), teamID, input.Member); err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}
	o, err := app.rederivePatruljeOrder(r.Context(), teamID, string(memberID), derivedLinesForPatrulje([]patrulje.Spejder{input.Member}))
	if err != nil {
		log.Printf("rederivePatruljeOrder %q", err)
	}
	if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"member": input.Member, "order": o}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// deletePatruljeMemberHandler removes a single member.
//
// @Summary      Delete a patrulje member
// @Description  Publishes one delete event, vacating the member's seat, and recomputes the open order.
// @Tags         patrulje
// @Produce      json
// @Param        id        path  string  true  "Team ID"
// @Param        memberId  path  string  true  "Member ID"
// @Success      200       {object}  object{order=order.Order}
// @Failure      404       {object}  object{error=string}
// @Router       /api/patrulje/{id}/member/{memberId} [delete]
func (app *application) deletePatruljeMemberHandler(w http.ResponseWriter, r *http.Request) {
	teamID := types.TeamID(app.ReadNamedParam(r, "id"))
	memberID := types.MemberID(app.ReadNamedParam(r, "memberId"))
	if teamID == "" || memberID == "" {
		app.NotFoundResponse(w, r)
		return
	}
	if err := app.commands.Patrulje.DeleteMember(r.Context(), teamID, memberID); err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}
	o, err := app.rederivePatruljeOrder(r.Context(), teamID, string(memberID), nil)
	if err != nil {
		log.Printf("rederivePatruljeOrder %q", err)
	}
	if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"order": o}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// derivedLinesForPatrulje translates the team-form member list into the
// derived order lines the patrulje order should hold. Two lines per active
// member at most:
//
//   - participation.patrulje (always, qty 1)
//   - tshirt.adult (only if the member picked a size; size goes on the line
//     attributes for receipt rendering)
//
// Both lines are keyed by memberId so the next SetDerivedLines from a form
// where the member is gone produces a snapshot without that line — the
// projector then drops it.
func derivedLinesForPatrulje(members []patrulje.Spejder) []order.DesiredLine {
	lines := make([]order.DesiredLine, 0, len(members)*2)
	for _, m := range members {
		if m.Deleted {
			continue
		}
		lines = append(lines, order.DesiredLine{
			ProductSKU: "participation.patrulje",
			MemberID:   string(m.MemberID),
			Quantity:   1,
		})
		if m.TShirtSize != "" {
			lines = append(lines, order.DesiredLine{
				ProductSKU: "tshirt.adult",
				MemberID:   string(m.MemberID),
				Quantity:   1,
				Attributes: map[string]any{"size": m.TShirtSize},
			})
		}
	}
	return lines
}

// derivedLinesForPatruljeSpejdere is the read-path variant of
// derivedLinesForPatrulje. It works with the []*data.Spejder slice returned
// by GetSpejdere (the show handler) rather than the []patrulje.Spejder used
// by the update handler.
func derivedLinesForPatruljeSpejdere(members []*data.Spejder) []order.DesiredLine {
	lines := make([]order.DesiredLine, 0, len(members)*2)
	for _, m := range members {
		lines = append(lines, order.DesiredLine{
			ProductSKU: "participation.patrulje",
			MemberID:   string(m.MemberID),
			Quantity:   1,
		})
		if m.TShirtSize != "" {
			lines = append(lines, order.DesiredLine{
				ProductSKU: "tshirt.adult",
				MemberID:   string(m.MemberID),
				Quantity:   1,
				Attributes: map[string]any{"size": m.TShirtSize},
			})
		}
	}
	return lines
}

// patruljeLinesNeedSync was the per-handler diff helper; it has moved to
// orders.go as derivedLinesNeedSync, shared with the klan and personnel
// show handlers.
