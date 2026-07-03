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

		messageID, err := app.mailer.Send(string(payment.ReceiptEmail), "payment_received.tmpl", data)
		if err != nil {
			app.logger.PrintError(err, nil)
			return
		}
		body := messages.NathejkMailSent{
			PingType:  types.PingTypePaymentReceived,
			MessageID: messageID,
			Recipient: payment.ReceiptEmail,
			Subject:   "Betaling modtaget",
		}
		switch payment.OrderType {
		case string(types.TeamTypeKlan),
			string(types.TeamTypePatrulje):
			body.TeamID = types.TeamID(payment.OrderForeignKey)

		case string(types.TeamTypeBadut),
			string(types.TeamTypeCrew),
			// Legacy values from the pre-rename era. Kept so the receipt-mail
			// pipeline keeps wiring up MemberID for any payment row that
			// still carries the old orderType. Drop once the DB rename has
			// caught up.
			"staff",
			"friend":
			body.MemberID = types.MemberID(payment.OrderForeignKey)

		case "order":
			// New flow: payment.OrderForeignKey is an order.orderId. Look up
			// the order to recover the owner identity for the receipt event.
			if o, oerr := app.models.Order.GetByID(r.Context(), payment.OrderForeignKey); oerr == nil {
				switch o.OwnerType {
				case types.TeamTypeKlan, types.TeamTypePatrulje:
					body.TeamID = types.TeamID(o.OwnerID)
				case types.TeamTypeBadut, types.TeamTypeCrew:
					body.MemberID = types.MemberID(o.OwnerID)
				}
			} else {
				app.logger.PrintError(oerr, map[string]string{"orderId": payment.OrderForeignKey})
			}
		}
		msg := app.jetstream.MessageFunc()(streaminterface.SubjectFromStr(fmt.Sprintf("NATHEJK.%s.mail.%s.sent", app.config.year, types.PingTypePaymentReceived)))
		msg.SetBody(&body)
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
