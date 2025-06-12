package personnel

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"

	"github.com/nathejk/shared-go/types"
	tables "nathejk.dk/nathejk/table"
)

type querier struct {
	db *sql.DB
}

func (q *querier) GetAll(ctx context.Context, filter Filter) ([]Staff, error) {
	query := `SELECT t.staffId, t.name, t.groupName, t.korps, t.klan, t.signupStatus
		FROM staff t
		JOIN patruljestatus ts ON t.teamId = ts.teamID
		` //WHERE (LOWER(p.year) = LOWER(?) OR ? = '')`
	args := []any{} //filter.YearSlug, filter.YearSlug}
	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	//totalRecords := 0
	staffs := []Staff{}
	for rows.Next() {
		var k Staff
		if err := rows.Scan(&k.ID, &k.Name, &k.Group, &k.Korps, &k.Klan, &k.Status); err != nil {
			//if err := rows.Scan(&klan.TeamID); err != nil {
			return nil, err
		}
		staffs = append(staffs, k)
	}
	// When the rows.Next() loop has finished, call rows.Err() to retrieve any error
	// that was encountered during the iteration.
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return staffs, nil
}

func (q *querier) GetByID(ctx context.Context, staffID types.UserID) (*Staff, error) {
	log.Printf("Inside GetByID( %q )", staffID)
	if len(staffID) == 0 {
		log.Printf("not id found %q", staffID)
		return nil, tables.ErrRecordNotFound
	}

	query := `SELECT t.userId, t.name, t.phone, t.email, t.groupName, t.korps, t.klan, t.signupStatus, t.tshirtSize, t.additionals
		FROM personnel t
		WHERE t.userId = ?`
	var t Staff
	var additionals []byte
	err := q.db.QueryRow(query, staffID).Scan(
		&t.ID,
		&t.Name,
		&t.Phone,
		&t.Email,
		&t.Group,
		&t.Korps,
		&t.Klan,
		&t.Status,
		&t.TshirtSize,
		&additionals,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, tables.ErrRecordNotFound
		default:
			return nil, err
		}
	}
	t.Additionals = map[string]any{}
	if len(additionals) > 0 {
		if err := json.Unmarshal(additionals, &t.Additionals); err != nil {
			return nil, err
		}
	}

	return &t, nil
}
