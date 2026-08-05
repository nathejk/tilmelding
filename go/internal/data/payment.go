package data

import (
	"database/sql"
	"errors"

	"github.com/nathejk/shared-go/types"
)

type Payment struct {
	Reference string         `json:"reference"`
	TeamID    types.TeamID   `json:"teamId"`
	Currency  types.Currency `json:"currency"`
	Amount    int            `json:"amount"`
	Method    string         `json:"method"`
	Status    string         `json:"status"`
	CreatedAt string         `json:"createdAt"`
	ChangedAt string         `json:"changedAt"`
}

type PaymentModel struct {
	DB *sql.DB
}

/*
func (m SignupModel) GetAll(filters Filters) ([]table.Year, Metadata, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `SELECT slug, name, theme, story, cityDeparture, cityDestination FROM year ORDER BY slug DESC`
	args := []any{} //filters.Year, filters.Year}
	rows, err := m.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, Metadata{}, err
	}
	defer rows.Close()

	totalRecords := 0
	years := []table.Year{}
	for rows.Next() {
		var r table.Year
		err := rows.Scan(&r.Slug, &r.Name, &r.Theme, &r.Story, &r.CityDeparture, &r.CityDestination)
		if err != nil {
			return nil, Metadata{}, err
		}
		years = append(years, r)
	}

	// When the rows.Next() loop has finished, call rows.Err() to retrieve any error
	// that was encountered during the iteration.
	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}
	metadata := calculateMetadata(filters.Year, totalRecords, filters.Page, filters.PageSize)

	return years, metadata, nil
	/*
		_, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		years := []*Year{
			&Year{Slug: "2023", Name: "Nathejk 2024", Theme: "Ex Nihilo", CityDeparture: "Kalundborg", CityDestination: "Stenlille"},
			&Year{Slug: "2022", Name: "Nathejk 2022", Theme: "Ufomania", CityDeparture: "Faxe", CityDestination: "Ringsted"},
			&Year{Slug: "2021", Name: "Nathejk 2021", Theme: "Kong Etruds Sværd", CityDeparture: "Helsingør", CityDestination: "Hillerød"},
		}
		return years, Metadata{}, nil
*/
//}

func (m PaymentModel) GetByID(teamID types.TeamID) (*Signup, error) {
	if len(teamID) == 0 {
		return nil, ErrRecordNotFound
	}

	query := `SELECT teamId, teamType, name, email, emailPending, phone, phonePending, pincode, createdAt
		FROM signup
		WHERE teamId = ?`
	var p Signup
	err := m.DB.QueryRow(query, teamID).Scan(
		&p.TeamID,
		&p.TeamType,
		&p.Name,
		&p.Email,
		&p.EmailPending,
		&p.Phone,
		&p.PhonePending,
		&p.Pincode,
		&p.CreatedAt,
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

// PaymentModel.ConfirmBySecret was removed together with the confirm projector:
// it read a `confirm` table that nothing populates or creates any more, it had
// no callers, and it is not part of PaymentInterface. Note that
// shared-go/tables/payment/queries.go carries a copy of this same method and
// should get the same treatment.
