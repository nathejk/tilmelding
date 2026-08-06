package data

import (
	"database/sql"
	"errors"

	"github.com/nathejk/shared-go/types"
)

type TeamModel struct {
	DB *sql.DB
}

// Deprecated: only GetKlan and GetContact remain here. GetPatrulje was removed
// in favour of the shared-go patrulje entity's own querier (patrulje.GetByID),
// which reads the same rows.
//
// GetKlan follows once klan.go declares response structs — swapping it now would
// put klan.Klan's differently-tagged fields straight onto the wire. GetContact
// is blocked upstream: patrulje.querier.GetContact exists in shared-go but is
// commented out, so there is nothing to migrate to.
//
// Removed earlier as orphaned: the Team type and its no-op Validate, the shared
// query() helper, and GetStartedTeamIDs / GetPatruljer / RequestedSeniorCount /
// GetLastPatruljeID. GetDiscontinuedTeamIDs went under task 028 because it also
// joined the unprojected `patruljemerged` table.

type Contact struct {
	TeamID     types.TeamID       `json:"teamId"`
	Name       string             `json:"name"`
	Address    string             `json:"address"`
	PostalCode string             `json:"postal"`
	Email      types.EmailAddress `json:"email"`
	Phone      types.PhoneNumber  `json:"phone"`
	Role       string             `json:"role"`
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

type Klan struct {
	ID                  types.TeamID       `json:"id"`
	Status              types.SignupStatus `json:"status"`
	Name                string             `json:"name"`
	Group               string             `json:"group"`
	Korps               string             `json:"korps"`
	MemberCount         int                `json:"memberCount"`
	ReservedMemberCount int                `json:"reservedMemberCount"`
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

// GetContact was removed: the shared-go patrulje entity owns the same query
// (patrulje.querier.GetContact) and returns an identical Contact struct.
