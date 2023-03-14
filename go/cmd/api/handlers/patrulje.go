package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/nathejk/shared-go/types"
	"nathejk.dk/table"
)

type ViewPatruljeQueries interface {
	Patrulje(types.TeamID) (*table.Patrulje, error)
	Spejdere(types.TeamID) ([]table.Spejder, error)
}

func ViewPatrulje(q ViewPatruljeQueries) http.HandlerFunc {

	type request struct {
	}
	type member struct {
		MemberID    types.MemberID    `json:"id"`
		Name        string            `json:"name"`
		Address     string            `json:"address"`
		PostalCode  string            `json:"postalCode"`
		City        string            `json:"city"`
		Email       types.Email       `json:"email"`
		Phone       types.PhoneNumber `json:"phone"`
		PhoneParent types.PhoneNumber `json:"phoneParent"`
		Birthday    types.Date        `json:"birthday"`
		Returning   bool              `json:"returning"`
	}
	type response struct {
		TeamID          types.TeamID      `json:"patruljeId"`
		Name            string            `json:"name"`
		GroupName       string            `json:"groupName"`
		Korps           string            `json:"korps"`
		ContactName     string            `json:"contactName"`
		ContactPhone    types.PhoneNumber `json:"contactPhone"`
		ContactEmail    types.Email       `json:"contactEmail"`
		ContactRole     string            `json:"contactRole"`
		Members         []member          `json:"spejdere"`
		PaidMemberCount int               `json:"paidMemberCount"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
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
	}
}

type UpdatePatruljeCommands interface {
	UpdatePatrulje(types.TeamID, string, string, string, string, types.PhoneNumber, types.Email, string) error
	UpdateSpejder(types.MemberID, types.TeamID, string, string, string, string, types.Email, types.PhoneNumber, types.PhoneNumber, types.Date, bool) error
	DeleteSpejder(types.MemberID) error
}

func UpdatePatrulje(c UpdatePatruljeCommands) http.HandlerFunc {

	type member struct {
		MemberID    types.MemberID    `json:"id"`
		Name        string            `json:"name"`
		Address     string            `json:"address"`
		PostalCode  string            `json:"postalCode"`
		City        string            `json:"city"`
		Email       types.Email       `json:"email"`
		Phone       types.PhoneNumber `json:"phone"`
		PhoneParent types.PhoneNumber `json:"phoneParent"`
		Birthday    types.Date        `json:"birthday"`
		Returning   bool              `json:"returning"`
		Deleted     bool              `json:"deleted"`
	}
	type request struct {
		TeamID       types.TeamID      `json:"patruljeId"`
		Name         string            `json:"name"`
		GroupName    string            `json:"groupName"`
		Korps        string            `json:"korps"`
		ContactName  string            `json:"contactName"`
		ContactPhone types.PhoneNumber `json:"contactPhone"`
		ContactEmail types.Email       `json:"contactEmail"`
		ContactRole  string            `json:"contactRole"`
		Members      []member          `json:"spejdere"`
	}
	type response struct {
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err := c.UpdatePatrulje(req.TeamID, req.Name, req.GroupName, req.Korps, req.ContactName, req.ContactPhone, req.ContactEmail, req.ContactRole)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		for _, m := range req.Members {
			if m.Deleted {
				err = c.DeleteSpejder(m.MemberID)
			} else {
				err = c.UpdateSpejder(m.MemberID, req.TeamID, m.Name, m.Address, m.PostalCode, m.City, m.Email, m.Phone, m.PhoneParent, m.Birthday, m.Returning)
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
