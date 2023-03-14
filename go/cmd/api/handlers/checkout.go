package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/nathejk/shared-go/types"
	"nathejk.dk/table"
)

type CheckoutQueries interface {
	Patrulje(types.TeamID) (*table.Patrulje, error)
	Spejdere(types.TeamID) ([]table.Spejder, error)
}
type CheckoutCommands interface {
	RequestSeats(types.TeamID) (types.SignupStatus, int, error)
	//RequestPayment
	//	Patrulje(types.TeamID) (*table.Patrulje, error)
	//	Spejdere(types.TeamID) ([]table.Spejder, error)
}

func Checkout(q CheckoutQueries, c CheckoutCommands) http.HandlerFunc {

	type request struct {
	}
	type response struct {
		TeamID            types.TeamID       `json:"teamId"`
		Status            types.SignupStatus `json:"status"`
		UnpaidMemberCount int                `json:"unpaidMemberCount"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		teamID := types.TeamID(r.URL.Path[14:])
		status, unpaidMemberCount, err := c.RequestSeats(teamID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response{
			TeamID:            teamID,
			Status:            status,
			UnpaidMemberCount: unpaidMemberCount,
		})
	}
}

type MobilepayCommands interface {
	RequestMobilepayLink(types.TeamID, types.PhoneNumber, int) error
}

func Mobilepay(c MobilepayCommands) http.HandlerFunc {

	type request struct {
		TeamID            types.TeamID      `json:"teamId"`
		UnpaidMemberCount int               `json:"unpaidMemberCount"`
		Phone             types.PhoneNumber `json:"phone"`
	}
	type response struct {
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err := c.RequestMobilepayLink(req.TeamID, req.Phone, req.UnpaidMemberCount)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response{})
	}
}
