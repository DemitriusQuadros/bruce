// Package database provides schema migration and initialization.
package database

import (
	"database/sql"
	"fmt"
	"log"
)

// RunMigrations executes all pending database schema migrations.
// This handles both fresh databases (via CREATE TABLE IF NOT EXISTS in schema.sql)
// and existing databases that need schema updates.
func RunMigrations(db *sql.DB) error {
	// First, run the DDL from schema.sql to create tables if they don't exist.
	if _, err := db.Exec(Schema); err != nil {
		return fmt.Errorf("exec schema.sql: %w", err)
	}

	// Then, apply incremental migrations for schema updates.
	migrations := []struct {
		name string
		sql  string
	}{
		{
			name: "add_provider_override_to_sessions",
			sql:  `ALTER TABLE sessions ADD COLUMN provider_override TEXT NOT NULL DEFAULT ''`,
		},
	}

	for _, m := range migrations {
		// Check if the column already exists before trying to add it.
		exists, err := columnExists(db, "sessions", "provider_override")
		if err != nil {
			return fmt.Errorf("check column exists: %w", err)
		}
		if exists {
			log.Printf("migration: column provider_override already exists, skipping")
			continue
		}

		// Apply the migration.
		log.Printf("migration: applying %s", m.name)
		if _, err := db.Exec(m.sql); err != nil {
			return fmt.Errorf("migration %s: %w", m.name, err)
		}
		log.Printf("migration: %s applied successfully", m.name)
	}

	return nil
}

// columnExists checks if a column exists in a SQLite table.
// It queries PRAGMA table_info and looks for the specified column by name.
func columnExists(db *sql.DB, table, column string) (bool, error) {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name string
		var typ string
		var notnull int
		var dfltValue interface{}
		var pk int
		if err := rows.Scan(&cid, &name, &typ, &notnull, &dfltValue, &pk); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}
