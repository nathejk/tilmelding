package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/nathejk/shared-go/types"
)

type FrontpageQueries interface {
	IsOpen(types.TeamType) bool
	SignupStart(types.TeamType) *time.Time
}

func Frontpage(q FrontpageQueries) http.HandlerFunc {

	type request struct {
	}
	type response struct {
		SignupStart *time.Time `json:"signupStart"`
		OpenSenior  bool       `json:"openSenior"`
		OpenSpejder bool       `json:"openSpejder"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response{
			SignupStart: q.SignupStart(types.TeamTypeKlan),
			OpenSenior:  q.IsOpen(types.TeamTypeKlan),
			OpenSpejder: q.IsOpen(types.TeamTypePatrulje),
		})
	}
}

type SignupCommands interface {
	Signup(types.TeamType, string, types.PhoneNumber, types.Email) (types.TeamID, error)
}

func SignUp(c SignupCommands) http.HandlerFunc {

	type request struct {
		TeamType types.TeamType    `json:"type"`
		Name     string            `json:"name"`
		Phone    types.PhoneNumber `json:"phone"`
		Email    types.Email       `json:"email"`
	}
	type response struct {
		TeamID types.TeamID `json:"teamId"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if !types.TeamTypes.Exists(req.TeamType) {
			http.Error(w, fmt.Sprintf("Unknown team type %q", req.TeamType), http.StatusBadRequest)
			return
		}
		teamID, err := c.Signup(req.TeamType, req.Name, req.Phone, req.Email)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotAcceptable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(response{
			TeamID: teamID,
		})
	}
}

type ConfirmCommands interface {
	UsePincode(types.TeamID, types.PhoneNumber, string) bool
}

func Confirm(c ConfirmCommands) http.HandlerFunc {

	type request struct {
		TeamID  types.TeamID      `json:"teamId"`
		Phone   types.PhoneNumber `json:"phone"`
		Pincode string            `json:"pincode"`
	}
	type response struct {
		TeamID types.TeamID `json:"teamId"`
		OK     bool
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		ok := c.UsePincode(req.TeamID, req.Phone, req.Pincode)
		if !ok {
			http.Error(w, "unauthorized", http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response{
			TeamID: req.TeamID,
			OK:     true,
		})
	}
}
