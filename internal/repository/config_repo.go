package repository

import (
	"database/sql"
	"fmt"

	"bruce/internal/domain"
)

// ConfigRepository defines the data access contract for runtime config entries.
type ConfigRepository interface {
	Upsert(key, value string) error
	Get(key string) (string, error)
	GetAll() ([]*domain.ConfigEntry, error)
}

// SQLiteConfigRepository is the SQLite-backed implementation of ConfigRepository.
type SQLiteConfigRepository struct {
	db *sql.DB
}

// NewConfigRepository returns a new SQLiteConfigRepository.
func NewConfigRepository(db *sql.DB) ConfigRepository {
	return &SQLiteConfigRepository{db: db}
}

// Upsert inserts or overwrites a config entry.
func (r *SQLiteConfigRepository) Upsert(key, value string) error {
	_, err := r.db.Exec(
		`INSERT INTO config_entries (key, value, updated_at) VALUES (?, ?, datetime('now'))
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = datetime('now')`,
		key, value,
	)
	if err != nil {
		return fmt.Errorf("config upsert: %w", err)
	}
	return nil
}

// Get returns the value for key, or an error if not found.
func (r *SQLiteConfigRepository) Get(key string) (string, error) {
	var value string
	err := r.db.QueryRow(`SELECT value FROM config_entries WHERE key = ?`, key).Scan(&value)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("config entry not found: %s", key)
		}
		return "", fmt.Errorf("config get: %w", err)
	}
	return value, nil
}

// GetAll returns all config entries ordered by key ascending.
func (r *SQLiteConfigRepository) GetAll() ([]*domain.ConfigEntry, error) {
	rows, err := r.db.Query(`SELECT key, value, updated_at FROM config_entries ORDER BY key ASC`)
	if err != nil {
		return nil, fmt.Errorf("config get_all: %w", err)
	}
	defer rows.Close()

	var entries []*domain.ConfigEntry
	for rows.Next() {
		e := &domain.ConfigEntry{}
		var updatedAt string
		if err := rows.Scan(&e.Key, &e.Value, &updatedAt); err != nil {
			return nil, fmt.Errorf("config get_all scan: %w", err)
		}
		var parseErr error
		e.UpdatedAt, parseErr = parseSQLiteTime(updatedAt)
		if parseErr != nil {
			return nil, fmt.Errorf("parse updated_at: %w", parseErr)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}
