package tablerow

import (
	"github.com/jrgensen/stream"
)

type SQLPrimaryKeys map[string]interface{}

type SQLTableRow interface {
	PrimaryKeys() SQLPrimaryKeys
	Sql() string
	CreateTableSql() (string, string)
}

type Consumer interface {
	Consume(string) error
}

type SQLTableCreator interface {
	CreateTableSql() string
}

type EntityChangedPublisher interface {
	stream.Publisher

	Changed(body interface{}) error
	Deleted(body interface{}) error
}
