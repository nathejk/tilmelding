package personnel

import (
	"database/sql"
	"log"

	"github.com/nathejk/shared-go/types"
	"nathejk.dk/pkg/tablerow"
	"nathejk.dk/superfluids/streaminterface"

	_ "embed"
)

type Staff struct {
	ID          types.UserID       `json:"id"`
	Type        types.TeamType     `json:"type"`
	Status      types.SignupStatus `json:"status"`
	Name        string             `json:"name"`
	Email       types.EmailAddress `json:"email"`
	Phone       types.PhoneNumber  `json:"phone"`
	Group       string             `json:"group"`
	Korps       string             `json:"korps"`
	Klan        string             `json:"klan"`
	TshirtSize  string             `json:"tshirtSize"`
	Additionals map[string]any     `json:"additionals"`
}

type table struct {
	commander
	consumer
	querier
}

func New(p streaminterface.Publisher, w tablerow.Consumer, r *sql.DB) *table {
	q := querier{db: r}
	c := commander{p: p}
	table := &table{commander: c, consumer: consumer{w: w}, querier: q}
	if err := w.Consume(table.CreateTableSql()); err != nil {
		log.Fatalf("Error creating table %q", err)
	}
	return table
}

//go:embed table.sql
var tableSchema string

func (t *table) CreateTableSql() string {
	return tableSchema
}
