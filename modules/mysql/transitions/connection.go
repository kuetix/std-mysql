package transitions

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"sync"

	_ "github.com/go-sql-driver/mysql"
)

var (
	dbMu    sync.Mutex
	dbCache = map[string]*sql.DB{}
)

// resolveDSN returns dsn, falling back to the MYSQL_DSN environment variable.
func resolveDSN(dsn string) string {
	dsn = strings.TrimSpace(dsn)
	if dsn != "" {
		return dsn
	}
	return strings.TrimSpace(os.Getenv("MYSQL_DSN"))
}

// resolveDB returns the cached *sql.DB for dsn, opening (and pinging) it on
// first use. database/sql pools connections internally, so one *sql.DB per
// DSN is reused for every call.
func resolveDB(dsn string) (*sql.DB, string, error) {
	resolvedDSN := resolveDSN(dsn)
	if resolvedDSN == "" {
		return nil, "", fmt.Errorf("dsn is required (pass dsn or set MYSQL_DSN)")
	}

	dbMu.Lock()
	defer dbMu.Unlock()

	if cached, ok := dbCache[resolvedDSN]; ok {
		return cached, resolvedDSN, nil
	}

	db, err := sql.Open("mysql", resolvedDSN)
	if err != nil {
		return nil, resolvedDSN, fmt.Errorf("failed to open mysql connection: %w", err)
	}

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, resolvedDSN, fmt.Errorf("failed to connect to mysql: %w", err)
	}

	dbCache[resolvedDSN] = db
	return db, resolvedDSN, nil
}

// redactDSN strips user:password@ from a DSN before it's ever echoed back in
// a response, so credentials never leak into workflow output or logs.
func redactDSN(dsn string) string {
	if at := strings.LastIndex(dsn, "@"); at != -1 {
		return dsn[at+1:]
	}
	return dsn
}

