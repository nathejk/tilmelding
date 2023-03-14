package query

import (
	"database/sql"
	"log"

	"nathejk.dk/pkg/types"
)

type bridge struct {
	mapTeam *sql.Stmt
}

func NewBridge(db *sql.DB) *bridge {
	prepare := func(query string) *sql.Stmt {
		stmt, err := db.Prepare(query)
		if err != nil {
			log.Fatal(err)
		}
		return stmt
	}
	q := &bridge{}
	q.mapTeam = prepare("SELECT remoteId FROM nathejk_team WHERE id = ?")

	return q
}

func (q *bridge) MapTeamID(teamID string) types.TeamID {
	var v types.TeamID
	if err := q.mapTeam.QueryRow(teamID).Scan(&v); err != nil {
		return ""
	}
	return v
}
