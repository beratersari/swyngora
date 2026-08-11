// Package sqliteutil shares SQLite schema-upgrade helpers across stores.
package sqliteutil

import (
	"database/sql"
	"fmt"
	"strings"
)

// IsBenignSchemaErr reports ALTER/CREATE errors that mean the object already exists.
// Unknown errors (disk, syntax, locked in a surprising way) are NOT benign.
func IsBenignSchemaErr(err error) bool {
	if err == nil {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate column") ||
		strings.Contains(msg, "duplicate column name") ||
		strings.Contains(msg, "already exists")
}

// ExecAllowExists runs stmt and ignores only "already exists" / duplicate-column errors.
func ExecAllowExists(db *sql.DB, stmt string) error {
	if db == nil {
		return fmt.Errorf("sqlite db is nil")
	}
	if _, err := db.Exec(stmt); err != nil && !IsBenignSchemaErr(err) {
		return fmt.Errorf("sqlite schema: %s: %w", truncateSQL(stmt), err)
	}
	return nil
}

// SetUserVersion records a monotonic schema version (PRAGMA user_version).
func SetUserVersion(db *sql.DB, v int) error {
	if v < 0 {
		return fmt.Errorf("sqlite user_version must be >= 0")
	}
	_, err := db.Exec(fmt.Sprintf(`PRAGMA user_version = %d`, v))
	if err != nil {
		return fmt.Errorf("set sqlite user_version: %w", err)
	}
	return nil
}

// UserVersion returns PRAGMA user_version (0 on a fresh unversioned file).
func UserVersion(db *sql.DB) (int, error) {
	var v int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&v); err != nil {
		return 0, fmt.Errorf("read sqlite user_version: %w", err)
	}
	return v, nil
}

func truncateSQL(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > 120 {
		return s[:117] + "..."
	}
	return s
}
