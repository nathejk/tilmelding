package data

import (
	"math"

	"github.com/nathejk/shared-go/types"
)

// Filters carries the query parameters the read models accept. Callers only
// ever set TeamID today; Year/Page/PageSize are consumed by the queries and by
// calculateMetadata.
//
// The Sort / SortSafelist fields were removed together with Validate,
// SortColumn, SortDirection, Offset and Limit — no caller invoked any of them,
// in handlers or in this package, so the sort-safelist machinery was dead. Note
// SortColumn panicked on an unrecognised value, which is worth knowing if
// sorting is ever reintroduced: build it to return an error instead.
type Filters struct {
	Year     string
	Page     int
	PageSize int
	TeamID   types.TeamID
}

type Metadata struct {
	Year         string       `json:"year"`
	TeamID       types.TeamID `json:"teamId,omitempty"`
	CurrentPage  int          `json:"current_page,omitempty"`
	PageSize     int          `json:"page_size,omitempty"`
	FirstPage    int          `json:"first_page,omitempty"`
	LastPage     int          `json:"last_page,omitempty"`
	TotalRecords int          `json:"total_records,omitempty"`
}

// The calculateMetadata() function calculates the appropriate pagination metadata
// values given the total number of records, current page, and page size values. Note
// that the last page value is calculated using the math.Ceil() function, which rounds
// up a float to the nearest integer. So, for example, if there were 12 records in total
// and a page size of 5, the last page value would be math.Ceil(12/5) = 3.
func calculateMetadata(year string, totalRecords, page, pageSize int) Metadata {
	if totalRecords == 0 {
		// Note that we return an empty Metadata struct if there are no records.
		return Metadata{}
	}
	return Metadata{
		Year:         year,
		CurrentPage:  page,
		PageSize:     pageSize,
		FirstPage:    1,
		LastPage:     int(math.Ceil(float64(totalRecords) / float64(pageSize))),
		TotalRecords: totalRecords,
	}
}
