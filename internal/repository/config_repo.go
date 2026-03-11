package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"bruce/internal/domain"
)

// ConfigRepository defines the data access contract for runtime config entries.
type ConfigRepository interface {
	Set(ctx context.Context, key, value string) error
	Get(ctx context.Context, key string) (*domain.ConfigEntry, error)
	List(ctx context.Context) ([]*domain.ConfigEntry, error)
	Delete(ctx context.Context, key string) error
}

// SQLiteConfigRepository is the SQLite-backed implementation of ConfigRepository.
type SQLiteConfigRepository struct {
	db *sql.DB
}

// NewConfigRepository returns a new SQLiteConfigRepository.
func NewConfigRepository(db *sql.DB) ConfigRepository {
	return &SQLiteConfigRepository{db: db}
}

// Set upserts a config entry.
func (r *SQLiteConfigRepository) Set(ctx context.Context, key, value string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO config_entries (key, value, updated_at) VALUES (?, ?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
		key, value, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("config set: %w", err)
	}
	return nil
}

// Get fetches a config entry by key.
func (r *SQLiteConfigRepository) Get(ctx context.Context, key string) (*domain.ConfigEntry, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT key, value, updated_at FROM config_entries WHERE key = ?`, key,
	)
	e := &domain.ConfigEntry{}
	if err := row.Scan(&e.Key, &e.Value, &e.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("config entry not found: %s", key)
		}
		return nil, fmt.Errorf("config get: %w", err)
	}
	return e, nil
}

// List returns all config entries.
func (r *SQLiteConfigRepository) List(ctx context.Context) ([]*domain.ConfigEntry, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT key, value, updated_at FROM config_entries ORDER BY key ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("config list: %w", err)
	}
	defer rows.Close()

	var entries []*domain.ConfigEntry
	for rows.Next() {
		e := &domain.ConfigEntry{}
		if err := rows.Scan(&e.Key, &e.Value, &e.UpdatedAt); err != nil {
			return nil, fmt.Errorf("config list scan: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// Delete removes a config entry by key.
func (r *SQLiteConfigRepository) Delete(ctx context.Context, key string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM config_entries WHERE key = ?`, key)
	if err != nil {
		return fmt.Errorf("config delete: %w", err)
	}
	return nil
}
