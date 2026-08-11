package sqliteutil

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestExecAllowExistsAndUserVersion(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(`CREATE TABLE t (id TEXT PRIMARY KEY)`); err != nil {
		t.Fatal(err)
	}
	if err := ExecAllowExists(db, `ALTER TABLE t ADD COLUMN note TEXT NOT NULL DEFAULT ''`); err != nil {
		t.Fatal(err)
	}
	if err := ExecAllowExists(db, `ALTER TABLE t ADD COLUMN note TEXT NOT NULL DEFAULT ''`); err != nil {
		t.Fatalf("duplicate column should be ignored: %v", err)
	}
	if err := ExecAllowExists(db, `ALTER TABLE missing ADD COLUMN x TEXT`); err == nil {
		t.Fatal("expected real migrate error")
	}
	if err := SetUserVersion(db, 3); err != nil {
		t.Fatal(err)
	}
	v, err := UserVersion(db)
	if err != nil || v != 3 {
		t.Fatalf("version=%d err=%v", v, err)
	}
}
