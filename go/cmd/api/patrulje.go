package main

import (
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/nathejk/shared-go/types"
	jsonapi "nathejk.dk/cmd/api/app"
	"nathejk.dk/internal/data"
	"nathejk.dk/internal/payment/mobilepay"
	"nathejk.dk/nathejk/commands"
	"nathejk.dk/nathejk/table/patrulje"
)

type SlugLabel struct {
	Slug  string `json:"slug"`
	Label string `json:"label"`
}
type TeamConfig struct {
	MinMemberCount int         `json:"minMemberCount"`
	MaxMemberCount int         `json:"maxMemberCount"`
	MemberPrice    int         `json:"memberPrice"`
	TShirtPrice    int         `json:"tshirtPrice"`
	Korps          []SlugLabel `json:"korps"`
	TShirtSizes    []SlugLabel `json:"tshirtSizes"`
}

func Korps() []SlugLabel {
	return []SlugLabel{
		{Slug: "dds", Label: "Det Danske Spejderkorps"},
		{Slug: "kfum", Label: "KFUM-Spejderne"},
		{Slug: "kfuk", Label: "De grønne pigespejdere"},
		{Slug: "dbs", Label: "Danske Baptisters Spejderkorps"},
		{Slug: "dgs", Label: "De Gule Spejdere"},
		{Slug: "dss", Label: "Dansk Spejderkorps Sydslesvig"},
		{Slug: "fdf", Label: "FDF / FPF"},
		{Slug: "andet", Label: "Andet"},
	}
}
func TShirtSizes() []SlugLabel {
	return []SlugLabel{
		{Slug: "", Label: "Ingen"},
		{Slug: "xs", Label: "X-Small"},
		{Slug: "s", Label: "Small"},
		{Slug: "m", Label: "Medium"},
		{Slug: "l", Label: "Large"},
		{Slug: "xl", Label: "X-Large"},
		{Slug: "xxl", Label: "XX-Large"},
	}
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

	config := TeamConfig{
		MinMemberCount: 3,
		MaxMemberCount: 7,
		MemberPrice:    250,
		TShirtPrice:    175,
		Korps:          Korps(),
		TShirtSizes:    TShirtSizes(),
	}
	contact, _ := app.models.Teams.GetContact(teamID)

	payments, _, err := app.models.Payment.GetAll(teamID)
	if err != nil {
		log.Printf("Payment.GetAll %q", err)
	}
	err = app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"config": config, "team": team, "contact": contact, "members": members, "payments": payments}, nil)
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
		_ = app.commands.Team.AssignNumber(team.TeamID)
		time.Sleep(time.Second)
		p, _ := app.models.Teams.GetPatrulje(team.TeamID)
		log.Printf("%s Got number %q", team.TeamID, p.Number)
	}
}

func (app *application) updatePatruljeHandler(w http.ResponseWriter, r *http.Request) {
	teamID := types.TeamID(app.ReadNamedParam(r, "id"))
	var input struct {
		Team    commands.Patrulje  `json:"team"`
		Contact commands.Contact   `json:"contact"`
		Members []commands.Spejder `json:"members"`
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
	err = app.commands.Team.UpdatePatrulje(teamID, input.Team, input.Contact, input.Members)
	if err != nil {
		log.Printf("UpdatePatrulje  %q", err)
		app.BadRequestResponse(w, r, err)
		return
	}
	var tshirtCount = 0
	var activeMemberCount = 0
	for _, member := range input.Members {
		if member.Deleted {
			continue
		}
		if len(member.TShirtSize) > 0 {
			tshirtCount++
		}
		activeMemberCount++
	}
	totalAmount := tshirtCount*175 + activeMemberCount*250
	paidAmount := app.models.Payment.AmountPaidByTeamID(teamID)
	dueAmount := totalAmount - paidAmount
	log.Printf("total=%d paid=%d due=%d\n", totalAmount, paidAmount, dueAmount)
	paymentLink := ""
	if dueAmount > 0 {
		signup, _ := app.models.Signup.GetByID(r.Context(), teamID)
		if (input.Contact.Phone == "") && (signup != nil) && (signup.Phone != nil) {
			input.Contact.Phone = *signup.Phone
		}
		if (input.Contact.Email == "") && (signup != nil) && (signup.Email != nil) {
			input.Contact.Email = *signup.Email
		}
		amount := mobilepay.Amount{Value: int64(dueAmount) * 100, Currency: mobilepay.Currency(types.CurrencyDKK)}
		teamUrl := "https://tilmelding.nathejk.dk/patrulje/" + string(teamID)

		paymentLink, _ = app.commands.Payment.Request(amount, "Nathejk tilmelding", input.Contact.Phone, input.Contact.Email, teamUrl, string(teamID), string(types.TeamTypePatrulje))
	}
	team, _ := app.models.Teams.GetPatrulje(teamID)
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
