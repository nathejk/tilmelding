package data

import (
	"database/sql"
	"errors"

	"github.com/nathejk/shared-go/types"
)

type TeamModel struct {
	DB *sql.DB
}

// Deprecated: only GetContact remains here, and it is blocked upstream —
// patrulje.querier.GetContact exists in shared-go but is commented out, so
// there is nothing to migrate to. Once that is uncommented this whole package
// goes away.
//
// GetPatrulje was removed in favour of the shared-go patrulje entity's own
// querier (patrulje.GetByID), and GetKlan in favour of klan.GetByID; both read
// the same rows.
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

// The Klan type and GetKlan were removed: the shared-go klan entity owns the
// same rows and its querier's GetByID replaces them. The one column this query
// selected that klan.GetByID does not is `reservedMemberCount`, which is no
// longer on the wire either — see klanTeamResponse in cmd/api/klan.go.
