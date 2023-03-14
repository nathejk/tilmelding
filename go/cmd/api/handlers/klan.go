package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/nathejk/shared-go/types"
	"nathejk.dk/table"
)

type KlanQueries interface {
	Klan(types.TeamID) (*table.Klan, error)
	Seniorer(types.TeamID) ([]table.Senior, error)
}

func Klan(q KlanQueries) http.HandlerFunc {

	type request struct {
	}
	type member struct {
		MemberID   types.MemberID    `json:"id"`
		Name       string            `json:"name"`
		Address    string            `json:"address"`
		PostalCode string            `json:"postalCode"`
		City       string            `json:"city"`
		Email      types.Email       `json:"email"`
		Phone      types.PhoneNumber `json:"phone"`
		Birthday   types.Date        `json:"birthday"`
		Returning  bool              `json:"returning"`
	}
	type response struct {
		TeamID          types.TeamID `json:"teamId"`
		Name            string       `json:"name"`
		GroupName       string       `json:"groupName"`
		Korps           string       `json:"korps"`
		Members         []member     `json:"spejdere"`
		PaidMemberCount int          `json:"paidMemberCount"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		p, err := q.Klan(types.TeamID(r.URL.Path[10:]))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if p == nil {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		members := []member{}
		s, e := q.Seniorer(p.TeamID)
		if e != nil {
			http.Error(w, e.Error(), http.StatusBadRequest)
			return
		}
		for _, m := range s {
			members = append(members, member{
				MemberID:   m.MemberID,
				Name:       m.Name,
				Address:    m.Address,
				PostalCode: m.PostalCode,
				City:       m.City,
				Email:      m.Email,
				Phone:      m.Phone,
				Birthday:   m.Birthday,
				Returning:  m.Returning,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response{
			TeamID:    p.TeamID,
			Name:      p.Name,
			GroupName: p.GroupName,
			Korps:     p.Korps,
			Members:   members,
		})
	}
}

type UpdateKlanCommands interface {
	UpdateKlan(types.TeamID, string, string, string) error
	UpdateSenior(types.MemberID, types.TeamID, string, string, string, string, types.Email, types.PhoneNumber, types.Date, bool) error
	DeleteSenior(types.MemberID) error
}

func UpdateKlan(c UpdateKlanCommands) http.HandlerFunc {

	type member struct {
		MemberID   types.MemberID    `json:"id"`
		Name       string            `json:"name"`
		Address    string            `json:"address"`
		PostalCode string            `json:"postalCode"`
		City       string            `json:"city"`
		Email      types.Email       `json:"email"`
		Phone      types.PhoneNumber `json:"phone"`
		Birthday   types.Date        `json:"birthday"`
		Returning  bool              `json:"returning"`
		Deleted    bool              `json:"deleted"`
	}
	type request struct {
		TeamID    types.TeamID `json:"teamId"`
		Name      string       `json:"name"`
		GroupName string       `json:"groupName"`
		Korps     string       `json:"korps"`
		Members   []member     `json:"spejdere"`
	}
	type response struct {
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err := c.UpdateKlan(req.TeamID, req.Name, req.GroupName, req.Korps)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		for _, m := range req.Members {
			if m.Deleted {
				err = c.DeleteSenior(m.MemberID)
			} else {
				err = c.UpdateSenior(m.MemberID, req.TeamID, m.Name, m.Address, m.PostalCode, m.City, m.Email, m.Phone, m.Birthday, m.Returning)
			}
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}

		/*
			p, err := q.Patrulje(types.TeamID(r.URL.Path[14:]))
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if p == nil {
				http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
				return
			}
			members := []member{}
			s, e := q.Spejdere(types.TeamID(r.URL.Path[14:]))
			if e != nil {
				http.Error(w, e.Error(), http.StatusBadRequest)
				return
			}
			for _, m := range s {
				members = append(members, member{
					MemberID:    m.MemberID,
					Name:        m.Name,
					Address:     m.Address,
					PostalCode:  m.PostalCode,
					City:        m.City,
					Email:       m.Email,
					Phone:       m.Phone,
					PhoneParent: m.PhoneParent,
					Birthday:    m.Birthday,
					Returning:   m.Returning,
				})
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response{
				TeamID:       p.TeamID,
				Name:         p.Name,
				GroupName:    p.GroupName,
				Korps:        p.Korps,
				ContactName:  p.ContactName,
				ContactPhone: p.ContactPhone,
				ContactEmail: p.ContactEmail,
				ContactRole:  p.ContactRole,
				Members:      members,
			})
		*/
	}
}
