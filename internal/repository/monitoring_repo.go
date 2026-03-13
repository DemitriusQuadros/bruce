package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"bruce/internal/monitoring"
)

// MonitoringRepository handles queries for logs and metrics from the database.
type MonitoringRepository struct {
	db *sql.DB
}

// NewMonitoringRepository creates a new MonitoringRepository.
func NewMonitoringRepository(db *sql.DB) *MonitoringRepository {
	return &MonitoringRepository{db: db}
}

// GetLogs retrieves log entries with optional filters.
type LogQueryOptions struct {
	Level     *string
	Logger    *string
	StartTime *time.Time
	EndTime   *time.Time
	Limit     int
}

// GetLogs retrieves log entries from the database.
func (mr *MonitoringRepository) GetLogs(ctx context.Context, opts LogQueryOptions) ([]monitoring.LogEntry, error) {
	if opts.Limit == 0 {
		opts.Limit = 50
	}
	if opts.Limit > 500 {
		opts.Limit = 500
	}

	query := `SELECT id, timestamp, level, logger, message, context FROM log_entries WHERE 1=1`
	args := []interface{}{}

	if opts.Level != nil && *opts.Level != "" {
		query += ` AND level = ?`
		args = append(args, *opts.Level)
	}

	if opts.Logger != nil && *opts.Logger != "" {
		query += ` AND logger = ?`
		args = append(args, *opts.Logger)
	}

	if opts.StartTime != nil {
		query += ` AND timestamp >= ?`
		args = append(args, *opts.StartTime)
	}

	if opts.EndTime != nil {
		query += ` AND timestamp <= ?`
		args = append(args, *opts.EndTime)
	}

	query += ` ORDER BY timestamp DESC LIMIT ?`
	args = append(args, opts.Limit)

	rows, err := mr.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []monitoring.LogEntry

	for rows.Next() {
		var entry monitoring.LogEntry
		var contextJSON sql.NullString

		err := rows.Scan(&entry.ID, &entry.Timestamp, &entry.Level, &entry.Logger, &entry.Message, &contextJSON)
		if err != nil {
			return nil, err
		}

		// Parse context JSON if present
		entry.Context = make(map[string]interface{})
		if contextJSON.Valid && contextJSON.String != "" {
			json.Unmarshal([]byte(contextJSON.String), &entry.Context)
		}

		logs = append(logs, entry)
	}

	return logs, rows.Err()
}

// MetricQueryOptions represents options for querying metrics.
type MetricQueryOptions struct {
	MetricName         *string
	StartTime          *time.Time
	EndTime            *time.Time
	AggregationWindow  int // in seconds, default 60
}

// GetMetrics retrieves metric snapshots from the database.
func (mr *MonitoringRepository) GetMetrics(ctx context.Context, opts MetricQueryOptions) ([]monitoring.MetricSnapshot, error) {
	if opts.AggregationWindow == 0 {
		opts.AggregationWindow = 60
	}

	query := `SELECT id, timestamp, metric_name, metric_type, value, labels, aggregation_window_seconds
			  FROM metric_snapshots WHERE 1=1`
	args := []interface{}{}

	if opts.MetricName != nil && *opts.MetricName != "" {
		query += ` AND metric_name = ?`
		args = append(args, *opts.MetricName)
	}

	if opts.StartTime != nil {
		query += ` AND timestamp >= ?`
		args = append(args, *opts.StartTime)
	}

	if opts.EndTime != nil {
		query += ` AND timestamp <= ?`
		args = append(args, *opts.EndTime)
	}

	query += ` ORDER BY timestamp DESC`

	rows, err := mr.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metrics []monitoring.MetricSnapshot

	for rows.Next() {
		var snapshot monitoring.MetricSnapshot
		var labelsJSON sql.NullString

		err := rows.Scan(
			&snapshot.ID,
			&snapshot.Timestamp,
			&snapshot.MetricName,
			&snapshot.MetricType,
			&snapshot.Value,
			&labelsJSON,
			&snapshot.AggregationWindowSecs,
		)
		if err != nil {
			return nil, err
		}

		// Parse labels JSON if present
		snapshot.Labels = make(map[string]string)
		if labelsJSON.Valid && labelsJSON.String != "" {
			json.Unmarshal([]byte(labelsJSON.String), &snapshot.Labels)
		}

		metrics = append(metrics, snapshot)
	}

	return metrics, rows.Err()
}

// GetConfig retrieves monitoring configuration values.
func (mr *MonitoringRepository) GetConfig(ctx context.Context, key string) (string, error) {
	var value string
	err := mr.db.QueryRowContext(ctx,
		`SELECT value FROM monitoring_config WHERE key = ?`,
		key,
	).Scan(&value)
	if err != nil {
		return "", err
	}
	return value, nil
}

// SetConfig updates a monitoring configuration value.
func (mr *MonitoringRepository) SetConfig(ctx context.Context, key, value string) error {
	_, err := mr.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO monitoring_config (key, value, updated_at) VALUES (?, ?, datetime('now'))`,
		key, value,
	)
	return err
}

// GetAllConfig retrieves all monitoring configuration entries.
func (mr *MonitoringRepository) GetAllConfig(ctx context.Context) (map[string]string, error) {
	rows, err := mr.db.QueryContext(ctx, `SELECT key, value FROM monitoring_config`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	config := make(map[string]string)

	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		config[key] = value
	}

	return config, rows.Err()
}

// DeleteOldLogs deletes log entries older than the specified duration.
func (mr *MonitoringRepository) DeleteOldLogs(ctx context.Context, retentionDays int) (int64, error) {
	result, err := mr.db.ExecContext(ctx,
		fmt.Sprintf(`DELETE FROM log_entries WHERE timestamp < datetime('now', '-%d days')`, retentionDays),
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// DeleteOldMetrics deletes metric snapshots older than the specified duration.
func (mr *MonitoringRepository) DeleteOldMetrics(ctx context.Context, retentionDays int) (int64, error) {
	result, err := mr.db.ExecContext(ctx,
		fmt.Sprintf(`DELETE FROM metric_snapshots WHERE timestamp < datetime('now', '-%d days')`, retentionDays),
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
