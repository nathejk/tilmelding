package query

import (
	"nathejk.dk/pkg/types"
	"nathejk.dk/table"
)

type Query interface {
	Patruljer() []table.Patrulje
	Patrulje(ID types.TeamID) (*table.Patrulje, error)
}
