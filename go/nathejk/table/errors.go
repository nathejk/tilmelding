package table

import "github.com/nathejk/shared-go/tables"

// Aliases of the module-wide sentinels in github.com/nathejk/shared-go/tables.
//
// The entity packages moved to shared-go and return that module's sentinels.
// Aliasing rather than redeclaring keeps a single error identity, so
// errors.Is(err, table.ErrRecordNotFound) still matches an error produced by a
// shared entity. Redeclaring them here with errors.New would create distinct
// values and silently break every comparison that crosses the boundary.
var (
	ErrRecordNotFound     = tables.ErrRecordNotFound
	ErrEditConflict       = tables.ErrEditConflict
	ErrVerificationFailed = tables.ErrVerificationFailed
)
