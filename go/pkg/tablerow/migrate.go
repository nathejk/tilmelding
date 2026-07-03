package tablerow

import (
	"database/sql"
	"fmt"
)

// EnsureColumn adds a column to a table if it doesn't already exist.
// It's the missing piece of the "CREATE TABLE IF NOT EXISTS" pattern this
// codebase uses: that statement creates new tables but never adds columns
// to existing ones.
//
// ddl is the part of an ALTER TABLE statement that follows "ADD COLUMN",
// for example:
//
//	"sizes VARCHAR(255) NOT NULL DEFAULT '' AFTER eligibleFor"
//
// The check uses INFORMATION_SCHEMA.COLUMNS, so it works on both MySQL
// and MariaDB without requiring a specific server version. EnsureColumn
// is idempotent and safe to call on every startup.
func EnsureColumn(db *sql.DB, w Consumer, table, column, ddl string) error {
	var n int
	err := db.QueryRow(`
		SELECT COUNT(*)
		FROM INFORMATION_SCHEMA.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE()
		  AND TABLE_NAME = ?
		  AND COLUMN_NAME = ?`, table, column).Scan(&n)
	if err != nil {
		return fmt.Errorf("check column %s.%s: %w", table, column, err)
	}
	if n > 0 {
		return nil
	}
	if err := w.Consume(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s", table, ddl)); err != nil {
		return fmt.Errorf("add column %s.%s: %w", table, column, err)
	}
	return nil
}

// EnsureIndex adds a named index to a table if it doesn't already exist.
// Like EnsureColumn but for indexes; ddl is the full ALTER TABLE
// statement so the caller has full control over UNIQUE, FULLTEXT,
// composite columns, etc.:
//
//	"ALTER TABLE order_line ADD INDEX idx_order_line_member (memberId)"
func EnsureIndex(db *sql.DB, w Consumer, table, index, ddl string) error {
	var n int
	err := db.QueryRow(`
		SELECT COUNT(*)
		FROM INFORMATION_SCHEMA.STATISTICS
		WHERE TABLE_SCHEMA = DATABASE()
		  AND TABLE_NAME = ?
		  AND INDEX_NAME = ?`, table, index).Scan(&n)
	if err != nil {
		return fmt.Errorf("check index %s.%s: %w", table, index, err)
	}
	if n > 0 {
		return nil
	}
	if err := w.Consume(ddl); err != nil {
		return fmt.Errorf("add index %s.%s: %w", table, index, err)
	}
	return nil
}
