package transitions

import (
	"database/sql"
	"testing"

	_ "github.com/go-sql-driver/mysql"
)

// testDSN points at a disposable local MySQL instance used for package tests.
const testDSN = "root:test@tcp(192.168.34.8:13306)/testdb"

// withCleanTestTable drops and recreates a fresh table for the test, so each
// test starts from an empty, known schema.
func withCleanTestTable(t *testing.T, table, ddl string) {
	t.Helper()

	db, err := sql.Open("mysql", testDSN)
	if err != nil {
		t.Fatalf("failed to open test connection: %v", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		t.Skipf("skipping: no mysql reachable at test DSN: %v", err)
	}

	if _, err := db.Exec("DROP TABLE IF EXISTS " + table); err != nil {
		db.Close()
		t.Fatalf("failed to drop test table: %v", err)
	}
	if _, err := db.Exec(ddl); err != nil {
		db.Close()
		t.Fatalf("failed to create test table: %v", err)
	}

	t.Cleanup(func() {
		_, _ = db.Exec("DROP TABLE IF EXISTS " + table)
		db.Close()
	})
}
