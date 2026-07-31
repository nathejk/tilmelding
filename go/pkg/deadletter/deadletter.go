// Package deadletter provides a safety net around the SQL projection writer.
//
// The projection consumers turn immutable stream events into rows in the
// read-model tables. A single event whose payload cannot be written (for
// example a value that overflows its VARCHAR column) must never take down the
// whole consumer loop, and — because the stream uses an ordered consumer with
// no redelivery — it must not be silently dropped either. This package captures
// such failures in a `deadletter` table so they can be inspected and replayed.
package deadletter

import (
	"database/sql"
	"log"
	"sync/atomic"

	"nathejk.dk/pkg/tablerow"

	_ "embed"
)

//go:embed table.sql
var tableSchema string

// Writer decorates a tablerow.Consumer with a dead-letter safety net.
//
// Until Arm is called it is a transparent pass-through: any error from the
// underlying Consume is returned unchanged, so startup work (table creation,
// product seeding) still fails loudly and visibly.
//
// After Arm, a failing statement can never crash the process. The statement
// and its error are recorded in the deadletter table and Consume returns nil,
// so the projection loop keeps moving. If recording the dead-letter itself
// fails (e.g. the database is unreachable), the failure is logged and nil is
// still returned — nothing here is ever fatal.
type Writer struct {
	inner tablerow.Consumer
	db    *sql.DB
	armed atomic.Bool
}

// New wraps inner. db is used to persist captured statements and should point
// at the same database inner writes to.
func New(inner tablerow.Consumer, db *sql.DB) *Writer {
	return &Writer{inner: inner, db: db}
}

// CreateTableSql returns the DDL for the deadletter table. Run it (loudly)
// during startup, before Arm, so the safety net exists before it is needed.
func (w *Writer) CreateTableSql() string { return tableSchema }

// Reset empties the deadletter table. The projections are rebuilt from the
// event stream on every start (the ordered consumer replays from sequence
// zero), so entries captured during the previous run's replay are stale and
// must be cleared before the new replay begins — otherwise they accumulate
// across restarts.
func (w *Writer) Reset() error {
	_, err := w.db.Exec("DELETE FROM deadletter")
	return err
}

// Count returns the number of statements currently captured in the deadletter
// table.
func (w *Writer) Count() (int, error) {
	var n int
	err := w.db.QueryRow("SELECT COUNT(*) FROM deadletter").Scan(&n)
	return n, err
}

// Arm switches the writer from pass-through to capture mode. Call it once,
// after startup DDL/seeding and before the message loop starts consuming.
func (w *Writer) Arm() { w.armed.Store(true) }

// Consume runs query against the underlying writer. Before Arm it behaves
// exactly like the wrapped writer. After Arm a failure is captured in the
// deadletter table and nil is returned so the caller never treats it as fatal.
func (w *Writer) Consume(query string) error {
	err := w.inner.Consume(query)
	if err == nil {
		return nil
	}
	if !w.armed.Load() {
		return err
	}
	if recErr := w.capture(query, err); recErr != nil {
		log.Printf("deadletter: could not record failing statement: %v (original error: %v)", recErr, err)
		return nil
	}
	log.Printf("deadletter: captured failing statement: %v", err)
	return nil
}

func (w *Writer) capture(query string, cause error) error {
	_, err := w.db.Exec(
		`INSERT INTO deadletter (query, error) VALUES (?, ?)`,
		query, cause.Error(),
	)
	return err
}
