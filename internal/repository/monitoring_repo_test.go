package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMonitoringRepository_GetLogs_WithFilters(t *testing.T) {
	db := setupMonitoringTestDB(t)
	defer db.Close()

	repo := NewMonitoringRepository(db)
	ctx := context.Background()

	// Insert test logs
	_, err := db.ExecContext(ctx, `
		INSERT INTO log_entries (id, timestamp, level, logger, message, context, enabled)
		VALUES (?, ?, ?, ?, ?, ?, 1)
	`, "log1", time.Now().UTC(), "ERROR", "handler", "error message", `{"error":"test"}`)
	require.NoError(t, err)

	_, err = db.ExecContext(ctx, `
		INSERT INTO log_entries (id, timestamp, level, logger, message, context, enabled)
		VALUES (?, ?, ?, ?, ?, ?, 1)
	`, "log2", time.Now().UTC(), "INFO", "worker", "info message", `{}`)
	require.NoError(t, err)

	// Query with level filter
	opts := LogQueryOptions{Level: stringPtr("ERROR")}
	logs, err := repo.GetLogs(ctx, opts)
	require.NoError(t, err)
	assert.Equal(t, 1, len(logs))
	assert.Equal(t, "ERROR", string(logs[0].Level))

	// Query with logger filter
	opts = LogQueryOptions{Logger: stringPtr("worker")}
	logs, err = repo.GetLogs(ctx, opts)
	require.NoError(t, err)
	assert.Equal(t, 1, len(logs))
	assert.Equal(t, "worker", logs[0].Logger)
}

func TestMonitoringRepository_GetLogs_WithTimeRange(t *testing.T) {
	db := setupMonitoringTestDB(t)
	defer db.Close()

	repo := NewMonitoringRepository(db)
	ctx := context.Background()

	now := time.Now().UTC()
	past := now.Add(-24 * time.Hour)
	future := now.Add(24 * time.Hour)

	// Insert test log
	_, err := db.ExecContext(ctx, `
		INSERT INTO log_entries (id, timestamp, level, logger, message, context, enabled)
		VALUES (?, ?, ?, ?, ?, ?, 1)
	`, "log1", now, "INFO", "handler", "message", `{}`)
	require.NoError(t, err)

	// Query with time range
	opts := LogQueryOptions{StartTime: &past, EndTime: &future}
	logs, err := repo.GetLogs(ctx, opts)
	require.NoError(t, err)
	assert.Equal(t, 1, len(logs))

	// Query with time range that excludes the log
	opts = LogQueryOptions{StartTime: &future}
	logs, err = repo.GetLogs(ctx, opts)
	require.NoError(t, err)
	assert.Equal(t, 0, len(logs))
}

func TestMonitoringRepository_GetMetrics(t *testing.T) {
	db := setupMonitoringTestDB(t)
	defer db.Close()

	repo := NewMonitoringRepository(db)
	ctx := context.Background()

	// Insert test metrics
	_, err := db.ExecContext(ctx, `
		INSERT INTO metric_snapshots
		(id, timestamp, metric_name, metric_type, value, labels, aggregation_window_seconds)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, "m1", time.Now().UTC(), "http_request_total", "COUNTER", 100, `{"status":"200"}`, 60)
	require.NoError(t, err)

	_, err = db.ExecContext(ctx, `
		INSERT INTO metric_snapshots
		(id, timestamp, metric_name, metric_type, value, labels, aggregation_window_seconds)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, "m2", time.Now().UTC(), "http_request_duration_ms", "HISTOGRAM", 45.5, `{"percentile":"p50"}`, 60)
	require.NoError(t, err)

	// Query all metrics
	metrics, err := repo.GetMetrics(ctx, MetricQueryOptions{})
	require.NoError(t, err)
	assert.Equal(t, 2, len(metrics))

	// Query by name
	opts := MetricQueryOptions{MetricName: stringPtr("http_request_total")}
	metrics, err = repo.GetMetrics(ctx, opts)
	require.NoError(t, err)
	assert.Equal(t, 1, len(metrics))
	assert.Equal(t, "http_request_total", metrics[0].MetricName)
	assert.Equal(t, 100.0, metrics[0].Value)
}

func TestMonitoringRepository_GetConfig(t *testing.T) {
	db := setupMonitoringTestDB(t)
	defer db.Close()

	repo := NewMonitoringRepository(db)
	ctx := context.Background()

	// Insert config
	_, err := db.ExecContext(ctx, `
		INSERT INTO monitoring_config (key, value) VALUES (?, ?)
	`, "test_key", "test_value")
	require.NoError(t, err)

	// Get config
	value, err := repo.GetConfig(ctx, "test_key")
	require.NoError(t, err)
	assert.Equal(t, "test_value", value)
}

func TestMonitoringRepository_SetConfig(t *testing.T) {
	db := setupMonitoringTestDB(t)
	defer db.Close()

	repo := NewMonitoringRepository(db)
	ctx := context.Background()

	// Set new config
	err := repo.SetConfig(ctx, "new_key", "new_value")
	require.NoError(t, err)

	// Verify it was set
	value, err := repo.GetConfig(ctx, "new_key")
	require.NoError(t, err)
	assert.Equal(t, "new_value", value)

	// Update existing config
	err = repo.SetConfig(ctx, "new_key", "updated_value")
	require.NoError(t, err)

	value, err = repo.GetConfig(ctx, "new_key")
	require.NoError(t, err)
	assert.Equal(t, "updated_value", value)
}

func TestMonitoringRepository_DeleteOldLogs(t *testing.T) {
	db := setupMonitoringTestDB(t)
	defer db.Close()

	repo := NewMonitoringRepository(db)
	ctx := context.Background()

	// Insert old log (31 days ago)
	oldTime := time.Now().UTC().AddDate(0, 0, -31)
	_, err := db.ExecContext(ctx, `
		INSERT INTO log_entries (id, timestamp, level, logger, message, context, enabled)
		VALUES (?, ?, ?, ?, ?, ?, 1)
	`, "old_log", oldTime, "INFO", "handler", "old message", `{}`)
	require.NoError(t, err)

	// Insert recent log
	_, err = db.ExecContext(ctx, `
		INSERT INTO log_entries (id, timestamp, level, logger, message, context, enabled)
		VALUES (?, ?, ?, ?, ?, ?, 1)
	`, "recent_log", time.Now().UTC(), "INFO", "handler", "recent message", `{}`)
	require.NoError(t, err)

	// Delete logs older than 30 days
	deleted, err := repo.DeleteOldLogs(ctx, 30)
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)

	// Verify recent log still exists
	var count int
	row := db.QueryRow(`SELECT COUNT(*) FROM log_entries`)
	require.NoError(t, row.Scan(&count))
	assert.Equal(t, 1, count)
}

func setupMonitoringTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)

	// Create tables
	schema := `
		CREATE TABLE log_entries (
			id TEXT PRIMARY KEY,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
			level TEXT NOT NULL CHECK(level IN ('DEBUG', 'INFO', 'WARN', 'ERROR')),
			logger TEXT NOT NULL,
			message TEXT NOT NULL,
			context TEXT,
			enabled BOOLEAN DEFAULT 1
		);

		CREATE TABLE metric_snapshots (
			id TEXT PRIMARY KEY,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
			metric_name TEXT NOT NULL,
			metric_type TEXT NOT NULL CHECK(metric_type IN ('COUNTER', 'GAUGE', 'HISTOGRAM')),
			value REAL NOT NULL,
			labels TEXT,
			aggregation_window_seconds INTEGER DEFAULT 60
		);

		CREATE TABLE monitoring_config (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE INDEX idx_log_entries_timestamp ON log_entries(timestamp DESC);
		CREATE INDEX idx_log_entries_level ON log_entries(level);
		CREATE INDEX idx_log_entries_logger ON log_entries(logger);
		CREATE INDEX idx_metric_snapshots_timestamp ON metric_snapshots(timestamp DESC);
		CREATE INDEX idx_metric_snapshots_name ON metric_snapshots(metric_name);
		CREATE INDEX idx_metric_snapshots_composite ON metric_snapshots(metric_name, timestamp DESC);
	`

	_, err = db.Exec(schema)
	require.NoError(t, err)

	return db
}

func stringPtr(s string) *string {
	return &s
}
