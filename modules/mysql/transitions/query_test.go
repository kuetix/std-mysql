package transitions

import "testing"

func TestQueryQueryRowExec(t *testing.T) {
	withCleanTestTable(t, "kuetix_users", `CREATE TABLE kuetix_users (
		id INT AUTO_INCREMENT PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		role VARCHAR(255) NOT NULL
	)`)

	exec := &execTransitions{}
	query := &queryTransitions{}

	insertResult := exec.Exec(testDSN, "INSERT INTO kuetix_users (name, role) VALUES (?, ?)", []interface{}{"Anar", "maintainer"})
	if !insertResult.Success {
		t.Fatalf("Exec insert failed: %v", insertResult.Error)
	}
	insertMap, ok := insertResult.Response.(map[string]interface{})
	if !ok {
		t.Fatalf("unexpected Exec response: %#v", insertResult.Response)
	}
	insertedID, _ := insertMap["lastInsertId"].(int64)
	if insertedID == 0 {
		t.Fatalf("expected non-zero lastInsertId, got %#v", insertMap)
	}
	if affected, _ := insertMap["rowsAffected"].(int64); affected != 1 {
		t.Fatalf("expected rowsAffected=1, got %#v", insertMap)
	}

	exec.Exec(testDSN, "INSERT INTO kuetix_users (name, role) VALUES (?, ?)", []interface{}{"Guest", "viewer"})

	rowResult := query.QueryRow(testDSN, "SELECT id, name, role FROM kuetix_users WHERE id = ?", []interface{}{insertedID})
	if !rowResult.Success {
		t.Fatalf("QueryRow failed: %v", rowResult.Error)
	}
	row, ok := rowResult.Response.(map[string]interface{})
	if !ok || row["name"] != "Anar" || row["role"] != "maintainer" {
		t.Fatalf("unexpected QueryRow response: %#v", rowResult.Response)
	}

	allResult := query.Query(testDSN, "SELECT id, name, role FROM kuetix_users ORDER BY id", nil)
	if !allResult.Success {
		t.Fatalf("Query failed: %v", allResult.Error)
	}
	rows, ok := allResult.Response.([]map[string]interface{})
	if !ok || len(rows) != 2 {
		t.Fatalf("unexpected Query response: %#v", allResult.Response)
	}

	updateResult := exec.Exec(testDSN, "UPDATE kuetix_users SET role = ? WHERE id = ?", []interface{}{"lead", insertedID})
	if !updateResult.Success {
		t.Fatalf("Exec update failed: %v", updateResult.Error)
	}
	if affected, _ := updateResult.Response.(map[string]interface{})["rowsAffected"].(int64); affected != 1 {
		t.Fatalf("expected rowsAffected=1 on update, got %#v", updateResult.Response)
	}

	deleteResult := exec.Exec(testDSN, "DELETE FROM kuetix_users WHERE id = ?", []interface{}{insertedID})
	if !deleteResult.Success {
		t.Fatalf("Exec delete failed: %v", deleteResult.Error)
	}

	missing := query.QueryRow(testDSN, "SELECT id FROM kuetix_users WHERE id = ?", []interface{}{insertedID})
	if missing.Success {
		t.Fatalf("expected QueryRow on deleted row to fail")
	}
	if missing.StatusCode != 404 {
		t.Fatalf("expected statusCode=404 for missing row, got %d", missing.StatusCode)
	}
}
