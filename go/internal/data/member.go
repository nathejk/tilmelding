package data

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/nathejk/shared-go/types"
)

// Removed as orphaned: the Member type and its no-op Validate, plus GetInactive
// and the SpejderStatus type it returned. GetInactive had no callers and
// inner-joined `spejderstatus`, a table whose projector writes nothing, so it
// could only ever return zero rows.

type MemberModel struct {
	DB *sql.DB
}

type Spejder struct {
	ID            types.MemberID     `json:"id"`
	MemberID      types.MemberID     `json:"memberId"`
	InitialTeamID types.TeamID       `json:"teamId"`
	CurrentTeamID types.TeamID       `json:"activeTeamId"`
	Status        types.MemberStatus `json:"status"`
	Name          string             `json:"name"`
	Address       string             `json:"address"`
	PostalCode    string             `json:"postalCode"`
	City          string             `json:"city"`
	Email         string             `json:"email"`
	Phone         string             `json:"phone"`
	PhoneParent   string             `json:"phoneContact"`
	Birthday      types.Date         `json:"birthday"`
	Returning     bool               `json:"returning"`
	TShirtSize    string             `json:"tshirtSize"`
}

func (m MemberModel) GetSpejdere(filters Filters) ([]*Spejder, Metadata, error) {
	// Create a context with a 3-second timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `Select
  s.memberId,
  s.teamId,
  IF(ss.status IS NULL, IF(ps.startedUts > 0, 'started', 'paid'), ss.status) AS status,
  name,
  address,
  postalCode,
  city,
  email,
  phone,
  phoneParent,
  birthday,
  ` + "`returning`" + `,
  tshirtsize
from spejder s
join patruljestatus ps on s.teamId = ps.teamId
left join spejderstatus ss on s.memberId = ss.id and s.year = ss.year
WHERE  (LOWER(s.year) = LOWER(?) OR ? = '') AND  (s.teamId = ? OR ? = '')`
	args := []any{filters.Year, filters.Year, filters.TeamID, filters.TeamID}
	rows, err := m.DB.QueryContext(ctx, query, args...)
	if err != nil {
		log.Print(err)
		return nil, Metadata{}, err
	}
	defer rows.Close()

	totalRecords := 0
	spejdere := []*Spejder{}
	for rows.Next() {
		var s Spejder
		if err := rows.Scan(&s.ID, &s.InitialTeamID, &s.Status, &s.Name, &s.Address, &s.PostalCode, &s.City, &s.Email, &s.Phone, &s.PhoneParent, &s.Birthday, &s.Returning, &s.TShirtSize); err != nil {
			log.Print(err)
			return nil, Metadata{}, err
		}
		s.MemberID = s.ID
		spejdere = append(spejdere, &s)
	}
	// When the rows.Next() loop has finished, call rows.Err() to retrieve any error
	// that was encountered during the iteration.
	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}
	metadata := calculateMetadata(filters.Year, totalRecords, filters.Page, filters.PageSize)

	return spejdere, metadata, nil
}

type Senior struct {
	ID         types.MemberID `json:"id"`
	MemberID   types.MemberID `json:"memberId"`
	TeamID     types.TeamID   `json:"teamId"`
	Name       string         `json:"name"`
	Address    string         `json:"address"`
	PostalCode string         `json:"postalCode"`
	City       string         `json:"city"`
	Email      string         `json:"email"`
	Phone      string         `json:"phone"`
	Birthday   types.Date     `json:"birthday"`
	Diet       string         `json:"diet"`
	TShirtSize string         `json:"tshirtSize"`
}

func (m MemberModel) GetSeniore(filters Filters) ([]*Senior, Metadata, error) {
	// Create a context with a 3-second timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `Select
  s.memberId,
  s.teamId,
  name,
  address,
  postalCode,
  city,
  email,
  phone,
  birthday,
  diet,
  tshirtsize
from senior s
WHERE  s.teamId = ?`
	args := []any{filters.TeamID}
	rows, err := m.DB.QueryContext(ctx, query, args...)
	if err != nil {
		log.Print(err)
		return nil, Metadata{}, err
	}
	defer rows.Close()

	totalRecords := 0
	members := []*Senior{}
	for rows.Next() {
		var s Senior
		if err := rows.Scan(&s.ID, &s.TeamID, &s.Name, &s.Address, &s.PostalCode, &s.City, &s.Email, &s.Phone, &s.Birthday, &s.Diet, &s.TShirtSize); err != nil {
			log.Print(err)
			return nil, Metadata{}, err
		}
		s.MemberID = s.ID
		members = append(members, &s)
	}
	// When the rows.Next() loop has finished, call rows.Err() to retrieve any error
	// that was encountered during the iteration.
	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}
	metadata := calculateMetadata(filters.Year, totalRecords, filters.Page, filters.PageSize)

	return members, metadata, nil
}

// TeamModel.GetSpejder was removed (task 028). Like GetDiscontinuedTeamIDs in
// team.go it joined a `patruljemerged` table that nothing projects, and it was
// dead: not part of the Models.Teams interface and never called — the live
// callers use Members.GetSpejdere (plural), which is a different method.
// shared-go/tables/spejder/querier.go carries a copy of this same dead method;
// it should go the same way.
