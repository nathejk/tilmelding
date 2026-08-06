package personnel

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"

	"github.com/jrgensen/cqrs"
	"github.com/nathejk/shared-go/tables"
	"github.com/nathejk/shared-go/types"
)

type querier struct {
	db cqrs.Reader
}

// GetAll was removed. It could never have worked: it selected `t.staffId` and
// joined on `t.teamId` FROM a table called `staff`, while this entity projects
// `personnel` — with a `userId` primary key and no teamId column at all. Every
// call would have failed with "table doesn't exist". Nothing called it: the
// handlers only use GetByID.
//
// It was also the last reader of `patruljestatus` left in this repo, which is
// why it is going now — see task 028.
//
// filter.go went with it (Filter, Metadata, calculateMetadata had no other
// user), as did PersonnelInterface.GetAll in internal/data.

func (q *querier) GetByID(ctx context.Context, staffID types.UserID) (*Staff, error) {
	log.Printf("Inside GetByID( %q )", staffID)
	if len(staffID) == 0 {
		log.Printf("not id found %q", staffID)
		return nil, tables.ErrRecordNotFound
	}

	query := `SELECT t.userId, t.userType, t.name, t.phone, t.email, t.groupName, t.korps, t.klan, t.signupStatus, t.tshirtSize, t.additionals
		FROM personnel t
		WHERE t.userId = ?`
	var t Staff
	var additionals []byte
	err := q.db.QueryRow(query, staffID).Scan(
		&t.ID,
		&t.Type,
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
