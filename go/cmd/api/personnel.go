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
	"nathejk.dk/nathejk/commands"
	"nathejk.dk/nathejk/table/payment"
)

func (app *application) showPersonnelHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	userID := types.UserID(app.ReadNamedParam(r, "id"))
	if userID == "" {
		app.NotFoundResponse(w, r)
		return
	}
	personnel, err := app.models.Personnel.GetByID(ctx, userID)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.NotFoundResponse(w, r)
		default:
			app.ServerErrorResponse(w, r, err)
		}
		return
	}
	config := TeamConfig{
		MinMemberCount: 1,
		MaxMemberCount: 1,
		MemberPrice:    100,
		TShirtPrice:    175,
		Korps:          Korps(),
		TShirtSizes:    TShirtSizes(),
	}
	if personnel.Type == "friend" {
		config.MemberPrice = 0
	}
	//contact, _ := app.models.Teams.GetContact(teamId)

	payments, _, err := app.models.Payment.GetAll(types.TeamID(userID))
	if err != nil {
		log.Printf("Payment.GetAll %q", err)
	}
	if payments == nil {
		payments = []payment.Payment{}
	}
	err = app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"config": config, "person": personnel, "payments": payments}, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}
func (app *application) updatePersonnelHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	userID := types.UserID(app.ReadNamedParam(r, "id"))
	var input struct {
		Person commands.Person `json:"person"`
	}
	if err := app.ReadJSON(w, r, &input); err != nil {
		log.Printf("ReadJSON %q", err)
		app.BadRequestResponse(w, r, err)
		return
	}

	personnel, err := app.models.Personnel.GetByID(ctx, userID)
	if err != nil {
		log.Printf("Signup.GetByID  %q", err)
		app.BadRequestResponse(w, r, err)
		return
	}
	err = app.commands.Personnel.UpdatePerson(userID, personnel.Type, input.Person)
	if err != nil {
		log.Printf("UpdatePerson  %q", err)
		app.BadRequestResponse(w, r, err)
		return
	}
	var tshirtCount = 0
	if input.Person.TshirtSize != "" {
		tshirtCount++
	}
	paymentLink := ""
	totalAmount := tshirtCount * 175
	if personnel.Type != "friend" {
		totalAmount = totalAmount + 100
	}
	paidAmount := app.models.Payment.AmountPaidByTeamID(types.TeamID(userID))
	dueAmount := totalAmount - paidAmount
	if dueAmount > 0 {
		signup, _ := app.models.Signup.GetByID(types.TeamID(userID))

		phone := types.PhoneNumber("")
		if (signup != nil) && (signup.Phone != nil) {
			phone = *signup.Phone
		}

		email := types.EmailAddress("")
		if (signup != nil) && (signup.Email != nil) {
			email = *signup.Email
		}
		amount := mobilepay.Amount{Value: int64(dueAmount) * 100, Currency: mobilepay.Currency(types.CurrencyDKK)}
		teamUrl := "https://tilmelding.nathejk.dk/badut/" + string(userID)

		paymentLink, _ = app.commands.Payment.Request(amount, "Nathejk gøglertilmelding", phone, email, teamUrl, string(userID), string("gøgler"))
	}
	person, _ := app.models.Personnel.GetByID(ctx, userID)
	/*
		page := fmt.Sprintf("/patrulje/%s", input.TeamID)
		err = app.WriteJSON(w, http.StatusCreated, jsonapi.Envelope{"team": map[string]string{"teamPage": page}}, nil)
		if err != nil {
			app.ServerErrorResponse(w, r, err)
		}*/
	err = app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"person": person, "paymentLink": paymentLink}, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}
