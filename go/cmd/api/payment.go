package main

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/nathejk/shared-go/messages"
	"github.com/nathejk/shared-go/types"
	jsonapi "nathejk.dk/cmd/api/app"
	"nathejk.dk/internal/data"
	"nathejk.dk/superfluids/streaminterface"
)

func (app *application) mobilepayCallbackHandler(w http.ResponseWriter, r *http.Request) {
	reference := app.ReadNamedParam(r, "ref")
	err := app.commands.Payment.Capture(reference)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}

	app.Background(func() {
		time.Sleep(2 * time.Second)
		payment, err := app.models.Payment.GetByReference(reference)
		if err != nil {
			app.logger.PrintError(err, nil)
			return
		}
		data := map[string]any{
			"payment": payment,
		}

		err = app.mailer.Send(string(payment.ReceiptEmail), "payment_received.tmpl", data)
		if err != nil {
			app.logger.PrintError(err, nil)
			return
		}
		msg := app.jetstream.MessageFunc()(streaminterface.SubjectFromStr(fmt.Sprintf("NATHEJK.%s.mail.%s.sent", "2025", types.PingTypePaymentReceived)))
		msg.SetBody(&messages.NathejkMailSent{
			PingType:  types.PingTypePaymentReceived,
			TeamID:    types.TeamID(""),
			Recipient: payment.ReceiptEmail,
			Subject:   "Betaling modtaget",
		})
		msg.SetMeta(&messages.Metadata{Producer: "tilmelding-api"})
		if err := app.jetstream.Publish(msg); err != nil {
			app.logger.PrintError(err, nil)
		}
	})
	http.Redirect(w, r, "/betaling/"+reference, http.StatusSeeOther)
}

func (app *application) showPaymentHandler(w http.ResponseWriter, r *http.Request) {
	reference := app.ReadNamedParam(r, "ref")
	payment, err := app.models.Payment.GetByReference(reference)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.NotFoundResponse(w, r)
		default:
			app.ServerErrorResponse(w, r, err)
		}
		return
	}
	err = app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"payment": payment}, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}
