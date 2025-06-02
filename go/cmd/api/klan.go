package main

import (
	"errors"
	"log"
	"net/http"

	"github.com/nathejk/shared-go/types"
	jsonapi "nathejk.dk/cmd/api/app"
	"nathejk.dk/internal/data"
	"nathejk.dk/internal/payment/mobilepay"
	"nathejk.dk/nathejk/commands"
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

	config := TeamConfig{
		MinMemberCount: 1,
		MaxMemberCount: 4,
		MemberPrice:    250,
		TShirtPrice:    175,
		Korps:          Korps(),
		TShirtSizes:    TShirtSizes(),
	}
	//contact, _ := app.models.Teams.GetContact(teamId)

	payments, _, err := app.models.Payment.GetAll(teamID)
	if err != nil {
		log.Printf("Payment.GetAll %q", err)
	}
	err = app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"config": config, "team": team, "members": members, "payments": payments}, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}
func (app *application) updateKlanHandler(w http.ResponseWriter, r *http.Request) {
	teamID := types.TeamID(app.ReadNamedParam(r, "id"))
	var input struct {
		Team    commands.Klan     `json:"team"`
		Members []commands.Senior `json:"members"`
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
	err = app.commands.Team.UpdateKlan(teamID, input.Team, input.Members)
	if err != nil {
		log.Printf("UpdateKlan  %q", err)
		app.BadRequestResponse(w, r, err)
		return
	}
	var tshirtCount = 0
	for _, member := range input.Members {
		if len(member.TShirtSize) > 0 {
			tshirtCount++
		}
	}
	paymentLink := ""
	totalAmount := tshirtCount*175 + len(input.Members)*250
	paidAmount := app.models.Payment.AmountPaidByTeamID(teamID)
	dueAmount := totalAmount - paidAmount
	if dueAmount > 0 {
		signup, _ := app.models.Signup.GetByID(teamID)

		phone := types.PhoneNumber("")
		if (signup != nil) && (signup.Phone != nil) {
			phone = *signup.Phone
		}

		email := types.EmailAddress("")
		if (signup != nil) && (signup.Email != nil) {
			email = *signup.Email
		}
		amount := mobilepay.Amount{Value: int64(dueAmount) * 100, Currency: mobilepay.Currency(types.CurrencyDKK)}
		teamUrl := "https://tilmelding.nathejk.dk/klan/" + string(teamID)

		paymentLink, _ = app.commands.Payment.Request(amount, "Nathejk tilmelding", phone, email, teamUrl, string(teamID), string(types.TeamTypeKlan))
	}
	team, _ := app.models.Teams.GetKlan(teamID)
	/*
		page := fmt.Sprintf("/patrulje/%s", input.TeamID)
		err = app.WriteJSON(w, http.StatusCreated, jsonapi.Envelope{"team": map[string]string{"teamPage": page}}, nil)
		if err != nil {
			app.ServerErrorResponse(w, r, err)
		}*/
	err = app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"team": team, "paymentLink": paymentLink}, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}
