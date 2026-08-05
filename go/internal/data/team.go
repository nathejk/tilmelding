package data

import (
	"database/sql"
	"errors"

	"github.com/nathejk/shared-go/types"
)

type TeamModel struct {
	DB *sql.DB
}

// Removed as orphaned: the Team type and its no-op Validate, the shared
// query() helper, and GetStartedTeamIDs / GetPatruljer / RequestedSeniorCount /
// GetLastPatruljeID — none had callers, in handlers or in this package, and
// none were part of Models.Teams once it was trimmed to what is actually used
// (GetPatrulje, GetKlan, GetContact). GetDiscontinuedTeamIDs went earlier under
// task 028 because it also joined the unprojected `patruljemerged` table.

type Patrulje struct {
	ID          types.TeamID `json:"id"`
	Number      string       `json:"number"`
	Status      string       `json:"status"`
	Name        string       `json:"name"`
	Group       string       `json:"group"`
	Korps       string       `json:"korps"`
	Liga        string       `json:"liga"`
	MemberCount int          `json:"memberCount"`
}
type Klan struct {
	ID                  types.TeamID       `json:"id"`
	Status              types.SignupStatus `json:"status"`
	Name                string             `json:"name"`
	Group               string             `json:"group"`
	Korps               string             `json:"korps"`
	MemberCount         int                `json:"memberCount"`
	ReservedMemberCount int                `json:"reservedMemberCount"`
}
type Contact struct {
	TeamID     types.TeamID       `json:"teamId"`
	Name       string             `json:"name"`
	Address    string             `json:"address"`
	PostalCode string             `json:"postal"`
	Email      types.EmailAddress `json:"email"`
	Phone      types.PhoneNumber  `json:"phone"`
	Role       string             `json:"role"`
}

func (m TeamModel) GetPatrulje(teamID types.TeamID) (*Patrulje, error) {
	if len(teamID) == 0 {
		return nil, ErrRecordNotFound
	}

	query := `SELECT p.teamId, p.teamNumber, p.name, p.groupName, p.korps, p.liga, p.memberCount
		FROM patrulje p
		JOIN patruljestatus ps ON p.teamId = ps.teamID
		WHERE p.teamId = ?`
	var p Patrulje
	err := m.DB.QueryRow(query, teamID).Scan(
		&p.ID,
		&p.Number,
		&p.Name,
		&p.Group,
		&p.Korps,
		&p.Liga,
		&p.MemberCount,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	return &p, nil
}

func (m TeamModel) GetKlan(teamID types.TeamID) (*Klan, error) {
	if len(teamID) == 0 {
		return nil, ErrRecordNotFound
	}

	query := `SELECT t.teamId, t.name, t.groupName, t.korps, t.memberCount, t.reservedMemberCount, t.signupStatus
		FROM klan t
		JOIN patruljestatus ts ON t.teamId = ts.teamID
		WHERE t.teamId = ?`
	var t Klan
	err := m.DB.QueryRow(query, teamID).Scan(
		&t.ID,
		&t.Name,
		&t.Group,
		&t.Korps,
		&t.MemberCount,
		&t.ReservedMemberCount,
		&t.Status,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	return &t, nil
}

func (m TeamModel) GetContact(teamID types.TeamID) (*Contact, error) {
	if len(teamID) == 0 {
		return nil, ErrRecordNotFound
	}

	query := `SELECT p.contactName, p.contactPhone, p.contactEmail, p.contactRole
		FROM patrulje p
		JOIN patruljestatus ps ON p.teamId = ps.teamID
		WHERE p.teamId = ?`
	c := Contact{TeamID: teamID}
	err := m.DB.QueryRow(query, teamID).Scan(
		&c.Name,
		&c.Phone,
		&c.Email,
		&c.Role,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	return &c, nil
}
