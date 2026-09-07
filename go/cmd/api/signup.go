package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/nathejk/shared-go/tables/signup"
	"github.com/nathejk/shared-go/types"
	jsonapi "nathejk.dk/cmd/api/app"
	"nathejk.dk/internal/data"
)

// Closing a signup type.
//
// A closed signup type is one nobody new may join: the gøgler signup once the
// gøgler places are filled. Existing signups are untouched — they keep their
// pages, their orders and their payments.
//
// This is enforced here rather than in the product catalogue, even though the
// catalogue can express "not for sale" (product.active) and already holds the klan
// seat cap (participation.klan.stock). Two reasons:
//
//   - A seat is claimed at *signup*, which happens before any order exists. The
//     catalogue is only consulted when order lines are built, i.e. after the person
//     has signed up, verified their email, verified their phone and reached their
//     page. Refusing them there would be several steps too late.
//   - Marking participation.gogler inactive would break the people who are already
//     signed up rather than the ones arriving: order.buildLines rejects any desired
//     line whose product is inactive, and every show and save re-derives a
//     participation line for them. See PRD 003 §8.5; task 036 is the prerequisite
//     for the catalogue being usable for this at all, and task 037 is where this
//     switch should move once it is.

// closedSignupTypesEnv is the environment variable naming the closed signup types,
// comma separated.
const closedSignupTypesEnv = "CLOSED_SIGNUP_TYPES"

// defaultClosedSignupTypes is what is closed when the environment says nothing.
//
// Defaults to closed rather than open: a deployment that forgets the variable must
// not quietly start accepting gøglers again. Re-opening is the explicit act.
var defaultClosedSignupTypes = []string{string(types.TeamTypeBadut)}

// signupTypeAliases maps the names this codebase uses for a team type onto the
// canonical types.TeamType value.
//
// The gøgler flow is called "badut" nearly everywhere a human looks — the route is
// /indskrivning/badut, the view is BadutView.vue, the Go constant is
// TeamTypeBadut — but the value on the wire and in the database is "gøgler".
// Without this, setting CLOSED_SIGNUP_TYPES=badut would parse cleanly, match
// nothing, and silently leave the signup open. It also means nobody has to type ø
// into a deployment config.
var signupTypeAliases = map[string]types.TeamType{
	"badut":  types.TeamTypeBadut,
	"gogler": types.TeamTypeBadut,
	"gøgler": types.TeamTypeBadut,
}

// closedSignupTypesFromEnv reads the closed-signup set from the environment.
//
// A variable that is *set but empty* means "nothing is closed", which is how a
// signup is re-opened without a code change. That is why this reads the
// environment directly instead of using getEnvAsSlice, which cannot tell an empty
// value from an absent one and would hand back the default in both cases.
func closedSignupTypesFromEnv() map[string]bool {
	raw, ok := os.LookupEnv(closedSignupTypesEnv)
	if !ok {
		return canonicalSignupTypes(defaultClosedSignupTypes)
	}
	return canonicalSignupTypes(strings.Split(raw, ","))
}

// canonicalSignupTypes resolves aliases and builds the lookup set.
func canonicalSignupTypes(values []string) map[string]bool {
	set := make(map[string]bool, len(values))
	for name := range stringSet(values) {
		if canonical, ok := signupTypeAliases[strings.ToLower(name)]; ok {
			name = string(canonical)
		}
		set[name] = true
	}
	return set
}

// Oversubscribed signup types.
//
// "Closed" and "oversubscribed" (overtegnet) are the same refusal with two
// different things to say, so they are two sets rather than one:
//
//   - closed is administrative — the gøgler places are filled, nothing more to
//     see. The refusal is the terse "tilmelding er lukket".
//   - oversubscribed is the event itself being full. Everyone who arrives, at any
//     point in the funnel, is told the same sentence in the same words:
//     signupOversubscribedMessage. It is also the only state that reaches past the
//     signup flow into an existing team's page, which is why it is a set the show
//     endpoints can ask about rather than a message baked into one handler.
//
// Being oversubscribed implies being closed — see signupClosed — so the front page
// button, the create endpoint and the team page all follow from this one set.
const oversubscribedSignupTypesEnv = "OVERSUBSCRIBED_SIGNUP_TYPES"

// defaultOversubscribedSignupTypes is what is full when the environment says
// nothing.
//
// Defaults to patrulje: the 2026 patrulje signup filled up at 185 teams. Same
// fail-closed reasoning as defaultClosedSignupTypes — a deployment that forgets
// the variable must not quietly start selling places that do not exist.
// OVERSUBSCRIBED_SIGNUP_TYPES="" re-opens, with no code change.
//
// This is deliberately a separate variable from CLOSED_SIGNUP_TYPES rather than an
// extra entry in it: a deployment that already sets CLOSED_SIGNUP_TYPES=badut
// explicitly would otherwise override this default and silently keep the patrulje
// signup open.
var defaultOversubscribedSignupTypes = []string{string(types.TeamTypePatrulje)}

// oversubscribedSignupTypesFromEnv reads the oversubscribed set from the
// environment. Set-but-empty means "nothing is full"; see
// closedSignupTypesFromEnv for why this reads os.LookupEnv directly.
func oversubscribedSignupTypesFromEnv() map[string]bool {
	raw, ok := os.LookupEnv(oversubscribedSignupTypesEnv)
	if !ok {
		return canonicalSignupTypes(defaultOversubscribedSignupTypes)
	}
	return canonicalSignupTypes(strings.Split(raw, ","))
}

// signupOversubscribed reports whether the given team type is full for the year.
func (app *application) signupOversubscribed(teamType types.TeamType) bool {
	return app.config.signup.oversubscribedTypes[string(teamType)]
}

// signupClosed reports whether the given team type may still sign up.
//
// Both the create endpoint and the front page read it, so what the user is told and
// what the server enforces cannot drift apart — which is exactly what had happened:
// the front page said CLOSED while the endpoint happily accepted gøglers.
func (app *application) signupClosed(teamType types.TeamType) bool {
	return app.config.signup.closedTypes[string(teamType)] || app.signupOversubscribed(teamType)
}

// signupStatus is the front page's label for a signup type. Klan has a third state
// (WAITINGLIST) that it computes for itself; see homeHandler.
func (app *application) signupStatus(teamType types.TeamType) string {
	if app.signupClosed(teamType) {
		return "CLOSED"
	}
	return "OPEN"
}

// signupClosedMessage is what a refused signup is told.
const signupClosedMessage = "tilmelding er lukket"

// signupOversubscribedMessage is what a refused signup is told when the event
// itself is full. Kept as one constant because it is shown in three places — the
// create endpoint's 422, the team page banner, and anything added later — and they
// must not drift into three slightly different sentences.
const signupOversubscribedMessage = "Nathejk er overtegnet for i år, tak for interessen"

// signupRefusalMessage is what to tell a team type that may not sign up.
func (app *application) signupRefusalMessage(teamType types.TeamType) string {
	if app.signupOversubscribed(teamType) {
		return signupOversubscribedMessage
	}
	return signupClosedMessage
}

// The cut-off: which *teams* the closure applies to.
//
// Closing the signup type shuts the front door, but it says nothing about the teams
// already inside — the 185 patruljer that signed up before the close. They must keep
// their page in full: editing the team, adding and removing members, and paying.
// Locking every patrulje page would have taken their roster and their payment button
// away, which is the opposite of what closing is for.
//
// So the team page asks a narrower question than the front page does: not "is this
// type full" but "is *this team* one of the ones that did not make it". The answer is
// when its signup was created, from signup.createdAt, compared against the moment the
// signup closed. Anything created at or after the cut-off is beyond capacity and gets
// the overtegnet page; everything before it is untouched.
//
// signup.createdAt is the only timestamp available: the patrulje projection has no
// created column, and status, team number and roster size are all things an admitted
// team can legitimately lack — using any of them would have locked out a team that
// had simply not added its members yet.
const oversubscribedSinceEnv = "OVERSUBSCRIBED_SINCE"

// defaultOversubscribedSince is the moment the patrulje signup filled up, in
// RFC 3339. Signups created before this are the teams that are in.
//
// Unlike the type set, this fails *open*: an empty or unparseable value means no team
// page is locked. Wrongly locking an admitted team costs it its roster and its
// payment; wrongly leaving a page open costs one page too many, and the front door is
// shut regardless. Given that asymmetry the fallback is to leave pages alone.
const defaultOversubscribedSince = "2026-09-07T00:00:00+02:00"

// oversubscribedSinceFromEnv reads the cut-off. The zero time means "no cut-off",
// i.e. lock nothing.
func oversubscribedSinceFromEnv() time.Time {
	raw, ok := os.LookupEnv(oversubscribedSinceEnv)
	if !ok {
		raw = defaultOversubscribedSince
	}
	if strings.TrimSpace(raw) == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, strings.TrimSpace(raw))
	if err != nil {
		log.Printf("%s=%q is not RFC 3339 (%v); no team page will be locked", oversubscribedSinceEnv, raw, err)
		return time.Time{}
	}
	return t
}

// signupCreatedAtLayouts are the formats signup.createdAt is known to hold.
//
// It is a VARCHAR written by the projection as `createdAt=%q` from the message time,
// so what lands in the column is time.Time's own String() format, not RFC 3339. Rows
// written by other paths, and any future switch to RFC 3339, are accepted too rather
// than being silently treated as unparseable.
var signupCreatedAtLayouts = []string{
	"2006-01-02 15:04:05.999999999 -0700 MST",
	"2006-01-02 15:04:05 -0700 MST",
	time.RFC3339Nano,
	"2006-01-02 15:04:05",
}

// parseSignupCreatedAt parses a signup's createdAt column. The bool reports whether
// the value could be read at all; callers must treat false as "unknown", never as old
// or new.
func parseSignupCreatedAt(value string) (time.Time, bool) {
	value = strings.TrimSpace(value)
	// A time.Time that carries a monotonic reading stringifies with an " m=+0.0"
	// suffix that no layout matches.
	if i := strings.Index(value, " m="); i > 0 {
		value = value[:i]
	}
	if value == "" {
		return time.Time{}, false
	}
	for _, layout := range signupCreatedAtLayouts {
		if t, err := time.Parse(layout, value); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// teamOversubscribed reports whether this specific team is one the closure applies
// to: its type is full *and* it signed up at or after the cut-off.
//
// Fails open at every step where the answer is not certain — no cut-off configured,
// no signup row, an unreadable timestamp — because the cost of a false positive is an
// admitted team locked out of its own roster and payment.
func (app *application) teamOversubscribed(ctx context.Context, teamType types.TeamType, teamID types.TeamID) bool {
	if !app.signupOversubscribed(teamType) {
		return false
	}
	cutoff := app.config.signup.oversubscribedSince
	if cutoff.IsZero() {
		return false
	}
	signup, err := app.models.Signup.GetByID(ctx, teamID)
	if err != nil || signup == nil {
		log.Printf("teamOversubscribed %q: no signup row (%v); leaving the page open", teamID, err)
		return false
	}
	created, ok := parseSignupCreatedAt(signup.CreatedAt)
	if !ok {
		log.Printf("teamOversubscribed %q: unreadable createdAt %q; leaving the page open", teamID, signup.CreatedAt)
		return false
	}
	return !created.Before(cutoff)
}

func (app *application) showSignupHandler(w http.ResponseWriter, r *http.Request) {
	id := types.TeamID(app.ReadNamedParam(r, "id"))
	if id == "" {
		log.Print("Not IDea")
		app.NotFoundResponse(w, r)
		return
	}
	team, err := app.models.Signup.GetByID(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			log.Printf("Not Found %q", id)
			app.NotFoundResponse(w, r)
		default:
			app.ServerErrorResponse(w, r, err)
		}
		return
	}
	err = app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"signup": team}, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

func (app *application) validatePhoneNumberHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		TeamID  types.TeamID `json:"teamId"`
		Pincode string       `json:"pincode"`
	}
	if err := app.ReadJSON(w, r, &input); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}
	payload := jsonapi.Envelope{"ok": true}
	err := app.commands.Signup.VerifyPhone(r.Context(), input.TeamID, input.Pincode)
	if err != nil {
		payload["ok"] = false
		payload["error"] = err.Error()
	}
	err = app.WriteJSON(w, http.StatusCreated, payload, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

func (app *application) validateEmailCallbackHandler(w http.ResponseWriter, r *http.Request) {
	teamID := types.TeamID(app.ReadNamedParam(r, "id"))
	secret := app.ReadNamedParam(r, "secret")
	err := app.commands.Signup.VerifyEmail(r.Context(), teamID, secret)
	if err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}
	err = app.commands.Signup.SendVerificationSms(r.Context(), teamID)
	if err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/indskrivning/%s", teamID), http.StatusSeeOther)
}

// createSignupHandler starts a signup: it publishes the signedup event and sends
// the verification email.
//
// @Summary      Start a signup
// @Description  Creates a signup for the given team type and sends a verification email. Refuses a team type whose signup is closed (see CLOSED_SIGNUP_TYPES) or full (see OVERSUBSCRIBED_SIGNUP_TYPES, whose 422 message is the "overtegnet" wording also shown on the team page) with 422 and a Danish message, publishing nothing — this is the only place a place is claimed, since it happens before any order exists. `/api/home` reports the same closed set, so a disabled button on the front page and this refusal cannot disagree.
// @Tags         signup
// @Accept       json
// @Produce      json
// @Param        body  body  object{type=string,name=string,emailPending=string,phonePending=string}  true  "Signup details. `type` is a team type: patrulje, klan, crew or gøgler."
// @Success      201   {object}  object{ok=bool}
// @Failure      400   {object}  object{error=string}
// @Failure      422   {object}  object{error=object}
// @Router       /api/signup [post]
func (app *application) createSignupHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		TeamType     types.TeamType     `json:"type"`
		Name         string             `json:"name"`
		EmailPending types.EmailAddress `json:"emailPending"`
		PhonePending types.PhoneNumber  `json:"phonePending"`
	}
	if err := app.ReadJSON(w, r, &input); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}
	// Refuse a closed signup type before anything is published. The front page
	// disables the button, but that is an affordance: this is where a place is
	// actually claimed, so it is where the refusal has to live. Reachable from a
	// stale page or a hand-made request.
	if app.signupClosed(input.TeamType) {
		app.FailedValidationResponse(w, r, map[string]string{"type": app.signupRefusalMessage(input.TeamType)})
		return
	}
	year := types.YearSlug(fmt.Sprintf("%d", time.Now().Year()))
	teamID, err := app.commands.Signup.Signup(r.Context(), year, signup.SignupCommand{
		TeamType: input.TeamType,
		Name:     input.Name,
		Phone:    input.PhonePending,
		Email:    input.EmailPending,
	})
	if err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}
	app.Background(func() {
		time.Sleep(3 * time.Second)
		err := app.commands.Signup.SendVerificationEmail(context.Background(), teamID)
		if err != nil {
			log.Printf("mail send failed %v", err)
		}
	})
	err = app.WriteJSON(w, http.StatusCreated, jsonapi.Envelope{"ok": true}, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}
