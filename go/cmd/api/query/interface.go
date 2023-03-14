package query

import (
	"github.com/nathejk/shared-go/types"
	"nathejk.dk/table"
)

type Query interface {
	Patruljer() []table.Patrulje
	Patrulje(ID types.TeamID) (*table.Patrulje, error)
}
