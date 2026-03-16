package db

import (
"database/sql"
"fmt"

_ "modernc.org/sqlite"
)

func Open(dsn string) (*sql.DB, error) {
	fullDSN := fmt.Sprintf("%s?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=ON&_txlock=immediate", dsn)

	db, err := sql.Open("sqlite", fullDSN)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	// SQLite supports one writer at a time; cap connections accordingly.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}

	return db, nil
}
