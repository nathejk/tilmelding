package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"

	jsonapi "nathejk.dk/cmd/api/app"
)

// migrateStaffToCrewHandler renames pre-rename string values from
// "staff" / "friend" / "staff.friend" to "crew" / "crew" / "crew" across
// the projection tables. Run this once after deploying the rename so
// that the runtime layer (which only emits and recognises "crew") stops
// having to fall back through legacy values for queries.
//
// Affected columns:
//
//   - personnel.userType: 'staff' | 'friend' → 'crew'
//   - payment.orderType:  'staff' | 'friend' → 'crew'
//   - orders.ownerType:   'staff' → 'crew'
//   - product.sku:        'participation.staff' / 'participation.staff.friend'
//     → 'participation.crew'
//   - product.eligibleFor: any 'staff' token → 'crew'
//   - order_line.productSku: per the SKU rename above
//
// Notes:
//
//   - The product rename collapses two SKUs onto one. If both rows
//     exist for the same year, the friend variant is dropped (the
//     remaining participation.staff row gets renamed). Re-running is
//     safe but won't re-introduce the dropped row.
//   - The handler is idempotent. After the first run, subsequent calls
//     report 0 rows touched everywhere.
//
// Auth: same MIGRATE_TOKEN scheme as migrateLegacyOrdersHandler. If the
// env var is unset the endpoint returns 404.
//
// Query params:
//
//	dry=1   (optional; report counts without changing anything)
//
// Response: JSON with per-table row counts and the dryRun flag.
func (app *application) migrateStaffToCrewHandler(w http.ResponseWriter, r *http.Request) {
	expected := os.Getenv("MIGRATE_TOKEN")
	if expected == "" {
		app.NotFoundResponse(w, r)
		return
	}
	if r.Header.Get("X-Migrate-Token") != expected {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	dryRun := r.URL.Query().Get("dry") == "1"

	report, err := runStaffToCrewRename(r.Context(), app.db, dryRun)
	if err != nil {
		app.logger.PrintError(err, nil)
		app.ServerErrorResponse(w, r, err)
		return
	}
	if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{
		"dryRun":  dryRun,
		"changes": report,
	}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// staffToCrewChange records the rows-affected count for a single SQL
// change so the response can show what moved.
type staffToCrewChange struct {
	Step string `json:"step"`
	Rows int64  `json:"rows"`
	Note string `json:"note,omitempty"`
}

func runStaffToCrewRename(ctx context.Context, db *sql.DB, dryRun bool) ([]staffToCrewChange, error) {
	// Each step is paired with a count query so dry-run can report what
	// _would_ change without touching the table. Running for real
	// executes the UPDATE and reports RowsAffected.
	steps := []struct {
		name    string
		count   string
		exec    string
		execArg []any
		note    string
	}{
		{
			name:  "personnel.userType",
			count: `SELECT COUNT(*) FROM personnel WHERE userType IN ('staff', 'friend')`,
			exec:  `UPDATE personnel SET userType='crew' WHERE userType IN ('staff', 'friend')`,
		},
		{
			name:  "payment.orderType",
			count: `SELECT COUNT(*) FROM payment WHERE orderType IN ('staff', 'friend')`,
			exec:  `UPDATE payment SET orderType='crew' WHERE orderType IN ('staff', 'friend')`,
		},
		{
			name:  "orders.ownerType",
			count: `SELECT COUNT(*) FROM orders WHERE ownerType='staff'`,
			exec:  `UPDATE orders SET ownerType='crew' WHERE ownerType='staff'`,
		},
		{
			name:  "product.sku.staff.friend",
			count: `SELECT COUNT(*) FROM product WHERE sku='participation.staff.friend'`,
			exec:  `DELETE FROM product WHERE sku='participation.staff.friend'`,
			note:  "dropped — friend variant collapses into participation.crew",
		},
		{
			name:  "product.sku.staff",
			count: `SELECT COUNT(*) FROM product WHERE sku='participation.staff'`,
			exec:  `UPDATE product SET sku='participation.crew', name='Crew-deltagelse', eligibleFor=REPLACE(eligibleFor, 'staff', 'crew') WHERE sku='participation.staff'`,
		},
		{
			name:  "product.eligibleFor.staff",
			count: `SELECT COUNT(*) FROM product WHERE FIND_IN_SET('staff', eligibleFor) > 0`,
			exec:  `UPDATE product SET eligibleFor=REPLACE(eligibleFor, 'staff', 'crew') WHERE FIND_IN_SET('staff', eligibleFor) > 0`,
		},
		{
			name:  "order_line.productSku",
			count: `SELECT COUNT(*) FROM order_line WHERE productSku IN ('participation.staff', 'participation.staff.friend')`,
			exec:  `UPDATE order_line SET productSku='participation.crew' WHERE productSku IN ('participation.staff', 'participation.staff.friend')`,
		},
	}

	out := make([]staffToCrewChange, 0, len(steps))
	for _, s := range steps {
		change := staffToCrewChange{Step: s.name, Note: s.note}
		if dryRun {
			var n int64
			if err := db.QueryRowContext(ctx, s.count).Scan(&n); err != nil {
				return nil, fmt.Errorf("%s count: %w", s.name, err)
			}
			change.Rows = n
		} else {
			res, err := db.ExecContext(ctx, s.exec, s.execArg...)
			if err != nil {
				return nil, fmt.Errorf("%s exec: %w", s.name, err)
			}
			change.Rows, _ = res.RowsAffected()
		}
		out = append(out, change)
	}
	return out, nil
}
