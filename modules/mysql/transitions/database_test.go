package transitions

import "testing"

func TestPingSuccessAndFailure(t *testing.T) {
	db := &databaseTransitions{}

	ok := db.Ping(testDSN)
	if !ok.Success {
		t.Skipf("skipping: no mysql reachable at test DSN: %v", ok.Error)
	}
	if connected, _ := ok.Response.(map[string]interface{})["connected"].(bool); !connected {
		t.Fatalf("expected connected=true, got %#v", ok.Response)
	}

	bad := db.Ping("root:test@tcp(127.0.0.1:1)/testdb")
	if bad.Success {
		t.Fatalf("expected Ping against unreachable target to fail")
	}
	if bad.Error == nil {
		t.Fatalf("expected an error on failed Ping")
	}
}
