// Package database provides SQLite connection setup for Bruce.
package database

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

// NewSQLiteDB opens a SQLite database at dsn with WAL journal mode, NORMAL
// synchronous mode, and foreign key enforcement. MaxOpenConns is set to 1 to
// serialise all access and prevent SQLITE_BUSY errors across goroutines.
func NewSQLiteDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dsn+"?_journal_mode=WAL&_synchronous=NORMAL&_foreign_keys=ON")
	if err != nil {
		return nil, fmt.Errorf("sqlite open: %w", err)
	}

	// Prevent "database is locked" — Asynq and HTTP handlers share the same DB.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("sqlite ping: %w", err)
	}

	return db, nil
}
