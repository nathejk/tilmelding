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

// signupClosed reports whether the given team type may still sign up.
//
// Both the create endpoint and the front page read it, so what the user is told and
// what the server enforces cannot drift apart — which is exactly what had happened:
// the front page said CLOSED while the endpoint happily accepted gøglers.
func (app *application) signupClosed(teamType types.TeamType) bool {
	return app.config.signup.closedTypes[string(teamType)]
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
// @Description  Creates a signup for the given team type and sends a verification email. Refuses a team type whose signup is closed (see CLOSED_SIGNUP_TYPES) with 422 and a Danish message, publishing nothing — this is the only place a place is claimed, since it happens before any order exists. `/api/home` reports the same closed set, so a disabled button on the front page and this refusal cannot disagree.
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
		app.FailedValidationResponse(w, r, map[string]string{"type": signupClosedMessage})
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
