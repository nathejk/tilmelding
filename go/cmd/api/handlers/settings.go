package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"nathejk.dk/pkg/types"
)

type LimitQueries interface {
	IsOpen(types.TeamType) bool
	MaxSeatCount(types.TeamType) int
	SignupStart(types.TeamType) *time.Time
}

func ViewLimits(q LimitQueries) http.HandlerFunc {

	type request struct {
	}
	type response struct {
		SignupStart    *time.Time `json:"signupStart"`
		MaxSeniorCount int        `json:"maxSeniorCount"`
		OpenSenior     bool       `json:"openSenior"`
		OpenSpejder    bool       `json:"openSpejder"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response{
			SignupStart:    q.SignupStart(types.TeamTypeKlan),
			MaxSeniorCount: q.MaxSeatCount(types.TeamTypeKlan),
			OpenSenior:     q.IsOpen(types.TeamTypeKlan),
			OpenSpejder:    q.IsOpen(types.TeamTypePatrulje),
		})
	}
}

type LimitCommands interface {
	OpenSignup(types.TeamType, int) error
	CloseSignup(types.TeamType) error
	SignupStart(*time.Time) error
}

func SaveLimits(c LimitCommands) http.HandlerFunc {
	type request struct {
		SignupStart    *time.Time `json:"signupStart"`
		MaxSeniorCount int        `json:"maxSeniorCount"`
		OpenSenior     bool       `json:"openSenior"`
		OpenSpejder    bool       `json:"openSpejder"`
	}
	type response struct {
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if req.OpenSpejder {
			c.OpenSignup(types.TeamTypePatrulje, 0)
		} else {
			c.CloseSignup(types.TeamTypePatrulje)
		}
		if req.OpenSenior {
			c.OpenSignup(types.TeamTypeKlan, req.MaxSeniorCount)
		} else {
			c.CloseSignup(types.TeamTypeKlan)
		}
		c.SignupStart(req.SignupStart)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response{})
	}
}

type MailTemplateQueries interface {
	MailTemplate(types.Slug) (string, string)
}

func MailTemplate(q MailTemplateQueries) http.HandlerFunc {

	type request struct {
	}
	type response struct {
		Slug     types.Slug             `json:"slug"`
		Subject  string                 `json:"subject"`
		Template string                 `json:"template"`
		Example  types.MailTemplateData `json:"example"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		slug := types.Slug(r.URL.Path[19:])
		subject, template := q.MailTemplate(slug)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response{
			Slug:     slug,
			Subject:  subject,
			Template: template,
			Example: types.MailTemplateData{
				Name: "Hold Ånd",
			},
		})
	}
}

type MailTemplateCommands interface {
	MailTemplate(types.Slug, string, string) error
}

func SaveMailTemplate(c MailTemplateCommands) http.HandlerFunc {

	type request struct {
		Slug     types.Slug `json:"slug"`
		Subject  string     `json:"subject"`
		Template string     `json:"template"`
	}
	type response struct {
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := c.MailTemplate(req.Slug, req.Subject, req.Template); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response{})
	}
}
