package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/nathejk/shared-go/types"
	jsonapi "nathejk.dk/cmd/api/app"
	"nathejk.dk/internal/data"
	"nathejk.dk/nathejk/table/signup"
)

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
		//PhoneNumber types.PhoneNumber `json:"phone"`
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
