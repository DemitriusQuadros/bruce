package repository

import (
	"database/sql"
	"fmt"
	"time"

	"bruce/internal/domain"
	"bruce/internal/monitoring"
)

// MessageRepository defines the data access contract for messages.
type MessageRepository interface {
	Insert(msg *domain.Message) error
	GetContextWindow(sessionID string, limit int) ([]*domain.Message, error)
	GetRecent(sessionID string, limit int) ([]*domain.Message, error)
}

// SQLiteMessageRepository is the SQLite-backed implementation of MessageRepository.
type SQLiteMessageRepository struct {
	db               *sql.DB
	metricsCollector *monitoring.Collector
}

// NewMessageRepository returns a new SQLiteMessageRepository.
func NewMessageRepository(db *sql.DB) MessageRepository {
	return &SQLiteMessageRepository{db: db, metricsCollector: nil}
}

// NewMessageRepositoryWithMetrics returns a new SQLiteMessageRepository with metrics collection.
func NewMessageRepositoryWithMetrics(db *sql.DB, collector *monitoring.Collector) MessageRepository {
	return &SQLiteMessageRepository{db: db, metricsCollector: collector}
}

// Insert stores a new message record.
func (r *SQLiteMessageRepository) Insert(msg *domain.Message) error {
	start := time.Now()
	_, err := r.db.Exec(
		`INSERT INTO messages (id, session_id, role, content, timestamp) VALUES (?, ?, ?, ?, ?)`,
		msg.ID, msg.SessionID, msg.Role, msg.Content, msg.Timestamp.Format("2006-01-02 15:04:05.000000000"),
	)
	if r.metricsCollector != nil {
		status := "success"
		if err != nil {
			status = "error"
		}
		r.metricsCollector.IncrementCounter("db_operation_total", map[string]string{
			"operation": "Insert",
			"entity":    "message",
			"result":    status,
		})
		r.metricsCollector.RecordHistogram("db_operation_duration_ms", float64(time.Since(start).Milliseconds()), map[string]string{
			"operation": "Insert",
			"entity":    "message",
			"status":    status,
		})
	}
	if err != nil {
		return fmt.Errorf("message insert: %w", err)
	}
	return nil
}

// GetContextWindow returns up to limit messages for sessionID in chronological order (oldest first).
// It fetches the N most recent messages and reverses them so the LLM receives history oldest→newest.
func (r *SQLiteMessageRepository) GetContextWindow(sessionID string, limit int) ([]*domain.Message, error) {
	start := time.Now()
	rows, err := r.db.Query(
		`SELECT id, session_id, role, content, timestamp
		 FROM (
		   SELECT id, session_id, role, content, timestamp
		   FROM messages
		   WHERE session_id = ?
		   ORDER BY timestamp DESC
		   LIMIT ?
		 )
		 ORDER BY timestamp ASC`,
		sessionID, limit,
	)
	if err != nil {
		if r.metricsCollector != nil {
			r.metricsCollector.RecordHistogram("db_operation_duration_ms", float64(time.Since(start).Milliseconds()), map[string]string{
				"operation": "GetContextWindow",
				"entity":    "message",
				"status":    "error",
			})
		}
		return nil, fmt.Errorf("message get_context_window: %w", err)
	}
	defer rows.Close()
	result, err := scanMessages(rows)
	if r.metricsCollector != nil {
		status := "success"
		if err != nil {
			status = "error"
		}
		r.metricsCollector.RecordHistogram("db_operation_duration_ms", float64(time.Since(start).Milliseconds()), map[string]string{
			"operation": "GetContextWindow",
			"entity":    "message",
			"status":    status,
		})
		r.metricsCollector.IncrementCounter("db_operation_total", map[string]string{
			"operation": "GetContextWindow",
			"entity":    "message",
		})
	}
	return result, err
}

// GetRecent returns up to limit messages for sessionID ordered newest first.
func (r *SQLiteMessageRepository) GetRecent(sessionID string, limit int) ([]*domain.Message, error) {
	start := time.Now()
	rows, err := r.db.Query(
		`SELECT id, session_id, role, content, timestamp
		 FROM messages
		 WHERE session_id = ?
		 ORDER BY timestamp DESC
		 LIMIT ?`,
		sessionID, limit,
	)
	if err != nil {
		if r.metricsCollector != nil {
			r.metricsCollector.RecordHistogram("db_operation_duration_ms", float64(time.Since(start).Milliseconds()), map[string]string{
				"operation": "GetRecent",
				"entity":    "message",
				"status":    "error",
			})
		}
		return nil, fmt.Errorf("message get_recent: %w", err)
	}
	defer rows.Close()
	result, err := scanMessages(rows)
	if r.metricsCollector != nil {
		status := "success"
		if err != nil {
			status = "error"
		}
		r.metricsCollector.RecordHistogram("db_operation_duration_ms", float64(time.Since(start).Milliseconds()), map[string]string{
			"operation": "GetRecent",
			"entity":    "message",
			"status":    status,
		})
		r.metricsCollector.IncrementCounter("db_operation_total", map[string]string{
			"operation": "GetRecent",
			"entity":    "message",
		})
	}
	return result, err
}

func scanMessages(rows *sql.Rows) ([]*domain.Message, error) {
	var messages []*domain.Message
	for rows.Next() {
		m := &domain.Message{}
		var ts string
		if err := rows.Scan(&m.ID, &m.SessionID, &m.Role, &m.Content, &ts); err != nil {
			return nil, fmt.Errorf("message scan: %w", err)
		}
		var err error
		m.Timestamp, err = parseSQLiteTime(ts)
		if err != nil {
			return nil, fmt.Errorf("parse timestamp: %w", err)
		}
		messages = append(messages, m)
	}
	return messages, rows.Err()
}
