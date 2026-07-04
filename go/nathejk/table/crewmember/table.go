package crewmember

import (
	"database/sql"
	"log"

	"github.com/doug-martin/goqu/v9"
	"github.com/jrgensen/stream"
	"nathejk.dk/pkg/tablerow"

	_ "embed"
)

type table struct {
	commander
	consumer
	querier
}

func New(p stream.Publisher, w tablerow.Consumer, r *sql.DB) *table {
	q := querier{db: r, r: goqu.New("mysql", r)}
	t := &table{commander: commander{p: p, q: &q}, consumer: consumer{w: w}, querier: q}
	if err := w.Consume(t.CreateTableSql()); err != nil {
		log.Printf("Error creating table %q", err)
	}
	return t
}

//go:embed table.sql
var tableSchema string

func (t *table) CreateTableSql() string {
	return tableSchema
}
