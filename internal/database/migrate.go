// Package database provides schema migration and initialization.
package database

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
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
	// Migration: add provider_override column.
	if exists, err := columnExists(db, "sessions", "provider_override"); err != nil {
		return fmt.Errorf("check column exists: %w", err)
	} else if !exists {
		log.Printf("migration: applying add_provider_override_to_sessions")
		if _, err := db.Exec(`ALTER TABLE sessions ADD COLUMN provider_override TEXT NOT NULL DEFAULT ''`); err != nil {
			return fmt.Errorf("migration add_provider_override_to_sessions: %w", err)
		}
		log.Printf("migration: add_provider_override_to_sessions applied successfully")
	}

	// Migration: add title column.
	if exists, err := columnExists(db, "sessions", "title"); err != nil {
		return fmt.Errorf("check column exists: %w", err)
	} else if !exists {
		log.Printf("migration: applying add_title_to_sessions")
		if _, err := db.Exec(`ALTER TABLE sessions ADD COLUMN title TEXT NOT NULL DEFAULT ''`); err != nil {
			return fmt.Errorf("migration add_title_to_sessions: %w", err)
		}
		log.Printf("migration: add_title_to_sessions applied successfully")
	}

	// Migration: add 'web' to connector_type CHECK constraint.
	// SQLite doesn't support ALTER CONSTRAINT, so we use the rename-copy pattern.
	if needsWebConstraint, err := checkNeedsWebConstraint(db); err != nil {
		return fmt.Errorf("check web constraint: %w", err)
	} else if needsWebConstraint {
		log.Printf("migration: applying add_web_connector_type")
		if err := migrateWebConnectorType(db); err != nil {
			return fmt.Errorf("migration add_web_connector_type: %w", err)
		}
		log.Printf("migration: add_web_connector_type applied successfully")
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

// checkNeedsWebConstraint returns true if the sessions table CHECK constraint
// does not yet include 'web'.
func checkNeedsWebConstraint(db *sql.DB) (bool, error) {
	var createSQL string
	err := db.QueryRow(
		`SELECT sql FROM sqlite_master WHERE type='table' AND name='sessions'`,
	).Scan(&createSQL)
	if err != nil {
		return false, err
	}
	// If 'web' already appears in the CREATE TABLE SQL, no migration needed.
	if strings.Contains(createSQL, "'web'") {
		return false, nil
	}
	return true, nil
}

// migrateWebConnectorType rebuilds the sessions table with an updated CHECK constraint.
func migrateWebConnectorType(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmts := []string{
		`CREATE TABLE sessions_new (
			id               TEXT    PRIMARY KEY,
			connector_type   TEXT    NOT NULL CHECK(connector_type IN ('whatsapp', 'discord', 'web')),
			channel_id       TEXT    NOT NULL,
			title            TEXT    NOT NULL DEFAULT '',
			system_prompt    TEXT    NOT NULL DEFAULT '',
			is_active        INTEGER NOT NULL DEFAULT 1,
			provider_override TEXT   NOT NULL DEFAULT '',
			created_at       DATETIME NOT NULL DEFAULT (datetime('now')),
			updated_at       DATETIME NOT NULL DEFAULT (datetime('now')),
			UNIQUE(connector_type, channel_id)
		)`,
		`INSERT INTO sessions_new (id, connector_type, channel_id, title, system_prompt, is_active, provider_override, created_at, updated_at)
		 SELECT id, connector_type, channel_id, COALESCE(title, ''), system_prompt, is_active, provider_override, created_at, updated_at FROM sessions`,
		`DROP TABLE sessions`,
		`ALTER TABLE sessions_new RENAME TO sessions`,
	}

	for _, stmt := range stmts {
		if _, err := tx.Exec(stmt); err != nil {
			return fmt.Errorf("migrate web connector: %w (stmt: %.60s...)", err, stmt)
		}
	}

	return tx.Commit()
}
