package db

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// Open connects to the SQLite database at path, applies any pending
// migrations, and returns a ready connection.
//
// The pool is capped at a single connection. SQLite serialises writes anyway,
// and for a single-user app this removes every opportunity for SQLITE_BUSY at
// the cost of nothing measurable.
func Open(path string) (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"file:%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)&_pragma=synchronous(NORMAL)",
		path,
	)

	conn, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	conn.SetMaxOpenConns(1)

	if err := conn.Ping(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("ping %s: %w", path, err)
	}
	if err := Migrate(conn); err != nil {
		conn.Close()
		return nil, err
	}
	return conn, nil
}
