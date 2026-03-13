package repository

import (
	"database/sql"
	"fmt"
	"time"

	"bruce/internal/domain"
	"bruce/internal/logging"
	"bruce/internal/monitoring"
)

// ConfigRepository defines the data access contract for runtime config entries.
type ConfigRepository interface {
	Upsert(key, value string) error
	Get(key string) (string, error)
	GetAll() ([]*domain.ConfigEntry, error)
}

// SQLiteConfigRepository is the SQLite-backed implementation of ConfigRepository.
type SQLiteConfigRepository struct {
	db               *sql.DB
	metricsCollector *monitoring.Collector
}

// NewConfigRepository returns a new SQLiteConfigRepository.
func NewConfigRepository(db *sql.DB) ConfigRepository {
	return &SQLiteConfigRepository{db: db, metricsCollector: nil}
}

// NewConfigRepositoryWithMetrics returns a new SQLiteConfigRepository with metrics collection.
func NewConfigRepositoryWithMetrics(db *sql.DB, collector *monitoring.Collector) ConfigRepository {
	return &SQLiteConfigRepository{db: db, metricsCollector: collector}
}

// Upsert inserts or overwrites a config entry.
func (r *SQLiteConfigRepository) Upsert(key, value string) error {
	start := time.Now()
	logging.Debugf("config upsert called: key=%q, value=%q (len=%d)", key, value, len(value))
	_, err := r.db.Exec(
		`INSERT INTO config_entries (key, value, updated_at) VALUES (?, ?, datetime('now'))
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = datetime('now')`,
		key, value,
	)
	if r.metricsCollector != nil {
		status := "success"
		if err != nil {
			status = "error"
		}
		r.metricsCollector.IncrementCounter("db_operation_total", map[string]string{
			"operation": "Upsert",
			"entity":    "config",
			"result":    status,
		})
		r.metricsCollector.RecordHistogram("db_operation_duration_ms", float64(time.Since(start).Milliseconds()), map[string]string{
			"operation": "Upsert",
			"entity":    "config",
			"status":    status,
		})
	}
	if err != nil {
		logging.Errorf("config upsert failed: key=%q, err=%v (type: %T)", key, err, err)
		return fmt.Errorf("config upsert: %w", err)
	}
	logging.Debugf("config upsert success: key=%q", key)
	return nil
}

// Get returns the value for key, or an error if not found.
func (r *SQLiteConfigRepository) Get(key string) (string, error) {
	logging.Debugf("config get called: key=%q", key)

	var value string
	err := r.db.QueryRow(`SELECT value FROM config_entries WHERE key = ?`, key).Scan(&value)
	if err != nil {
		if err == sql.ErrNoRows {
			logging.Debugf("config entry not found in database: key=%q (ErrNoRows)", key)
			return "", fmt.Errorf("config entry not found: %s", key)
		}
		logging.Errorf("config get failed: key=%q, err=%v (type: %T)", key, err, err)
		return "", fmt.Errorf("config get: %w", err)
	}
	logging.Debugf("config get success: key=%q, value=%q (len=%d)", key, value, len(value))
	return value, nil
}

// GetAll returns all config entries ordered by key ascending.
func (r *SQLiteConfigRepository) GetAll() ([]*domain.ConfigEntry, error) {
	start := time.Now()
	logging.Debug("config get_all called")
	rows, err := r.db.Query(`SELECT key, value, updated_at FROM config_entries ORDER BY key ASC`)
	if err != nil {
		if r.metricsCollector != nil {
			r.metricsCollector.RecordHistogram("db_operation_duration_ms", float64(time.Since(start).Milliseconds()), map[string]string{
				"operation": "GetAll",
				"entity":    "config",
				"status":    "error",
			})
		}
		logging.Errorf("config get_all query failed: err=%v (type: %T)", err, err)
		return nil, fmt.Errorf("config get_all: %w", err)
	}
	defer rows.Close()

	var entries []*domain.ConfigEntry
	for rows.Next() {
		e := &domain.ConfigEntry{}
		var updatedAt string
		if err := rows.Scan(&e.Key, &e.Value, &updatedAt); err != nil {
			logging.Errorf("config get_all scan failed: err=%v (type: %T)", err, err)
			return nil, fmt.Errorf("config get_all scan: %w", err)
		}
		var parseErr error
		e.UpdatedAt, parseErr = parseSQLiteTime(updatedAt)
		if parseErr != nil {
			logging.Errorf("config get_all parse failed: key=%q, err=%v", e.Key, parseErr)
			return nil, fmt.Errorf("parse updated_at: %w", parseErr)
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		logging.Errorf("config get_all rows iteration failed: err=%v (type: %T)", err, err)
		return nil, fmt.Errorf("config get_all rows: %w", err)
	}
	if r.metricsCollector != nil {
		r.metricsCollector.RecordHistogram("db_operation_duration_ms", float64(time.Since(start).Milliseconds()), map[string]string{
			"operation": "GetAll",
			"entity":    "config",
			"status":    "success",
		})
		r.metricsCollector.IncrementCounter("db_operation_total", map[string]string{
			"operation": "GetAll",
			"entity":    "config",
		})
	}
	logging.Debugf("config get_all returned %d entries", len(entries))
	return entries, nil
}
