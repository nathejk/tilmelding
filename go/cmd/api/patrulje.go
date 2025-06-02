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
		SlugLabel{Slug: "dds", Label: "Det Danske Spejderkorps"},
		SlugLabel{Slug: "kfum", Label: "KFUM-Spejderne"},
		SlugLabel{Slug: "kfuk", Label: "De grønne pigespejdere"},
		SlugLabel{Slug: "dbs", Label: "Danske Baptisters Spejderkorps"},
		SlugLabel{Slug: "dgs", Label: "De Gule Spejdere"},
		SlugLabel{Slug: "dss", Label: "Dansk Spejderkorps Sydslesvig"},
		SlugLabel{Slug: "fdf", Label: "FDF / FPF"},
		SlugLabel{Slug: "andet", Label: "Andet"},
	}
}
func TShirtSizes() []SlugLabel {
	return []SlugLabel{
		SlugLabel{Slug: "", Label: "Ingen"},
		SlugLabel{Slug: "xs", Label: "X-Small"},
		SlugLabel{Slug: "s", Label: "Small"},
		SlugLabel{Slug: "m", Label: "Medium"},
		SlugLabel{Slug: "l", Label: "Large"},
		SlugLabel{Slug: "xl", Label: "X-Large"},
		SlugLabel{Slug: "xxl", Label: "XX-Large"},
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
	for _, member := range input.Members {
		if len(member.TShirtSize) > 0 {
			tshirtCount++
		}
	}
	paymentLink := ""
	totalAmount := tshirtCount*175 + len(input.Members)*250
	paidAmount := app.models.Payment.AmountPaidByTeamID(teamID)
	dueAmount := totalAmount - paidAmount
	log.Printf("total=%d paid=%d due=%d\n", totalAmount, paidAmount, dueAmount)
	if dueAmount > 0 {
		signup, _ := app.models.Signup.GetByID(teamID)
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
