package mysql

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/kuetix/std-mysql/modules"

	"github.com/kuetix/engine"
	"github.com/kuetix/engine/engine/domain"
	engineModule "github.com/kuetix/engine/modules"
	_ "github.com/go-sql-driver/mysql"
)

// testDSN points at a disposable local MySQL instance used for package tests.
const testDSN = "root:test@tcp(192.168.34.8:13306)/testdb"

// runWorkflow runs a mysql workflow through the real engine (parsing,
// dependency injection, and transition dispatch), with args passed as native
// Go values via Context (as an HTTP handler or another workflow would).
func runWorkflow(t *testing.T, name string, args map[string]interface{}) *workflowResult {
	t.Helper()

	engineModule.Enable()
	modules.Enable()

	responses := engine.RunWorkflow("production", &domain.Options{
		EngineName: "mysql-test",
		ConfigName: "engine",
		Workflow:   "@mysql/" + name,
		Amount:     1,
		Retry:      1,
		LogPath:    "stdout",
		Context: map[string]interface{}{
			"args": args,
		},
	})

	res, ok := responses[name]
	if !ok {
		t.Fatalf("workflow %q: no response returned", name)
	}
	return &workflowResult{StatusCode: res.StatusCode, Err: res.GetError(), Response: res.Response}
}

type workflowResult struct {
	StatusCode int
	Err        error
	Response   interface{}
}

func chdirToPackageRoot(t *testing.T) {
	t.Helper()
	original, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if _, err := os.Stat(filepath.Join(original, "workflows")); err != nil {
		t.Fatalf("expected to run from package root (workflows dir not found): %v", err)
	}
}

func checkMySQLReachable(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("mysql", testDSN)
	if err != nil {
		t.Fatalf("failed to open test connection: %v", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		t.Skipf("skipping: no mysql reachable at test DSN: %v", err)
	}
	return db
}

func TestPingWorkflowSuccessAndFailure(t *testing.T) {
	chdirToPackageRoot(t)
	checkMySQLReachable(t).Close()

	ok := runWorkflow(t, "ping", map[string]interface{}{"dsn": testDSN})
	if ok.StatusCode != 200 {
		t.Fatalf("expected ping success, got status=%d response=%#v", ok.StatusCode, ok.Response)
	}
	connected, _ := ok.Response.(map[string]interface{})["connected"].(bool)
	if !connected {
		t.Fatalf("expected connected=true, got %#v", ok.Response)
	}
	if target, _ := ok.Response.(map[string]interface{})["target"].(string); target == testDSN {
		t.Fatalf("expected target to be redacted (no credentials), got %q", target)
	}

	bad := runWorkflow(t, "ping", map[string]interface{}{"dsn": "root:test@tcp(127.0.0.1:1)/testdb"})
	if bad.StatusCode != 503 {
		t.Fatalf("expected statusCode=503 on failure, got %d", bad.StatusCode)
	}
	errMsg, _ := bad.Response.(map[string]interface{})["error"].(string)
	if errMsg == "" {
		t.Fatalf("expected non-empty error message in failure response, got %#v", bad.Response)
	}
}

func TestQueryExecWorkflowsEndToEnd(t *testing.T) {
	chdirToPackageRoot(t)
	db := checkMySQLReachable(t)
	defer db.Close()

	if _, err := db.Exec("DROP TABLE IF EXISTS kuetix_wf_users"); err != nil {
		t.Fatalf("failed to drop test table: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE kuetix_wf_users (
		id INT AUTO_INCREMENT PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		role VARCHAR(255) NOT NULL
	)`); err != nil {
		t.Fatalf("failed to create test table: %v", err)
	}
	t.Cleanup(func() { _, _ = db.Exec("DROP TABLE IF EXISTS kuetix_wf_users") })

	insert := runWorkflow(t, "exec", map[string]interface{}{
		"dsn":     testDSN,
		"sqlText": "INSERT INTO kuetix_wf_users (name, role) VALUES (?, ?)",
		"args":    []interface{}{"Anar", "maintainer"},
	})
	if insert.StatusCode != 200 {
		t.Fatalf("exec insert failed: status=%d response=%#v", insert.StatusCode, insert.Response)
	}
	insertedID, _ := insert.Response.(map[string]interface{})["lastInsertId"].(int64)
	if insertedID == 0 {
		t.Fatalf("expected non-zero lastInsertId, got %#v", insert.Response)
	}

	runWorkflow(t, "exec", map[string]interface{}{
		"dsn":     testDSN,
		"sqlText": "INSERT INTO kuetix_wf_users (name, role) VALUES (?, ?)",
		"args":    []interface{}{"Guest", "viewer"},
	})

	row := runWorkflow(t, "query_row", map[string]interface{}{
		"dsn":     testDSN,
		"sqlText": "SELECT id, name, role FROM kuetix_wf_users WHERE id = ?",
		"args":    []interface{}{insertedID},
	})
	if row.StatusCode != 200 {
		t.Fatalf("query_row failed: status=%d response=%#v", row.StatusCode, row.Response)
	}
	if name, _ := row.Response.(map[string]interface{})["name"].(string); name != "Anar" {
		t.Fatalf("unexpected query_row response: %#v", row.Response)
	}

	all := runWorkflow(t, "query", map[string]interface{}{
		"dsn":     testDSN,
		"sqlText": "SELECT id, name, role FROM kuetix_wf_users ORDER BY id",
		"args":    []interface{}{},
	})
	if all.StatusCode != 200 {
		t.Fatalf("query failed: status=%d response=%#v", all.StatusCode, all.Response)
	}
	rows, ok := all.Response.([]map[string]interface{})
	if !ok || len(rows) != 2 {
		t.Fatalf("unexpected query response: %#v", all.Response)
	}

	update := runWorkflow(t, "exec", map[string]interface{}{
		"dsn":     testDSN,
		"sqlText": "UPDATE kuetix_wf_users SET role = ? WHERE id = ?",
		"args":    []interface{}{"lead", insertedID},
	})
	if update.StatusCode != 200 {
		t.Fatalf("exec update failed: status=%d response=%#v", update.StatusCode, update.Response)
	}

	del := runWorkflow(t, "exec", map[string]interface{}{
		"dsn":     testDSN,
		"sqlText": "DELETE FROM kuetix_wf_users WHERE id = ?",
		"args":    []interface{}{insertedID},
	})
	if del.StatusCode != 200 {
		t.Fatalf("exec delete failed: status=%d response=%#v", del.StatusCode, del.Response)
	}

	missing := runWorkflow(t, "query_row", map[string]interface{}{
		"dsn":     testDSN,
		"sqlText": "SELECT id FROM kuetix_wf_users WHERE id = ?",
		"args":    []interface{}{insertedID},
	})
	if missing.StatusCode != 404 {
		t.Fatalf("expected statusCode=404 for missing row, got %d (response=%#v)", missing.StatusCode, missing.Response)
	}

	badSQL := runWorkflow(t, "query", map[string]interface{}{
		"dsn":     testDSN,
		"sqlText": "SELECT * FROM this_table_does_not_exist",
		"args":    []interface{}{},
	})
	if badSQL.StatusCode != 400 {
		t.Fatalf("expected statusCode=400 for invalid SQL, got %d (response=%#v)", badSQL.StatusCode, badSQL.Response)
	}
}
