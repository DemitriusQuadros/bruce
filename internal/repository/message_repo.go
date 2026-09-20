package repository

import (
	"database/sql"
	"fmt"

	"bruce/internal/domain"
)

// MessageRepository defines the data access contract for messages.
type MessageRepository interface {
	Insert(msg *domain.Message) error
	GetContextWindow(sessionID string, limit int) ([]*domain.Message, error)
	GetRecent(sessionID string, limit int) ([]*domain.Message, error)
	CountBySession(sessionID string) (int, error)
	GetOlderMessages(sessionID string, keepRecent int) ([]*domain.Message, error)
	GetLastMessage(sessionID string) (*domain.Message, error)
}

// SQLiteMessageRepository is the SQLite-backed implementation of MessageRepository.
type SQLiteMessageRepository struct {
	db *sql.DB
}

// NewMessageRepository returns a new SQLiteMessageRepository.
func NewMessageRepository(db *sql.DB) MessageRepository {
	return &SQLiteMessageRepository{db: db}
}

// Insert stores a new message record.
func (r *SQLiteMessageRepository) Insert(msg *domain.Message) error {
	_, err := r.db.Exec(
		`INSERT INTO messages (id, session_id, role, content, timestamp) VALUES (?, ?, ?, ?, ?)`,
		msg.ID, msg.SessionID, msg.Role, msg.Content, msg.Timestamp.Format("2006-01-02 15:04:05.000000000"),
	)
	if err != nil {
		return fmt.Errorf("message insert: %w", err)
	}
	return nil
}

// GetContextWindow returns up to limit messages for sessionID in chronological order (oldest first).
// It fetches the N most recent messages and reverses them so the LLM receives history oldest→newest.
func (r *SQLiteMessageRepository) GetContextWindow(sessionID string, limit int) ([]*domain.Message, error) {
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
		return nil, fmt.Errorf("message get_context_window: %w", err)
	}
	defer rows.Close()
	return scanMessages(rows)
}

// GetRecent returns up to limit messages for sessionID ordered newest first.
func (r *SQLiteMessageRepository) GetRecent(sessionID string, limit int) ([]*domain.Message, error) {
	rows, err := r.db.Query(
		`SELECT id, session_id, role, content, timestamp
		 FROM messages
		 WHERE session_id = ?
		 ORDER BY timestamp DESC
		 LIMIT ?`,
		sessionID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("message get_recent: %w", err)
	}
	defer rows.Close()
	return scanMessages(rows)
}

func (r *SQLiteMessageRepository) CountBySession(sessionID string) (int, error) {
	var count int
	err := r.db.QueryRow(
		`SELECT COUNT(*) FROM messages WHERE session_id = ?`,
		sessionID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("message count_by_session: %w", err)
	}
	return count, nil
}

// GetOlderMessages returns messages in chronological order, skipping the most recent `keepRecent` messages.
func (r *SQLiteMessageRepository) GetOlderMessages(sessionID string, keepRecent int) ([]*domain.Message, error) {
	// Calculate how many messages to return
	total, err := r.CountBySession(sessionID)
	if err != nil {
		return nil, err
	}

	olderCount := total - keepRecent
	if olderCount <= 0 {
		return []*domain.Message{}, nil
	}

	rows, err := r.db.Query(
		`SELECT id, session_id, role, content, timestamp
		 FROM messages
		 WHERE session_id = ?
		 ORDER BY timestamp ASC
		 LIMIT ?`,
		sessionID, olderCount,
	)
	if err != nil {
		return nil, fmt.Errorf("message get_older_messages: %w", err)
	}
	defer rows.Close()
	return scanMessages(rows)
}

// GetLastMessage returns the most recent message in the session, or nil if no messages exist.
func (r *SQLiteMessageRepository) GetLastMessage(sessionID string) (*domain.Message, error) {
	row := r.db.QueryRow(
		`SELECT id, session_id, role, content, timestamp
		 FROM messages
		 WHERE session_id = ?
		 ORDER BY timestamp DESC
		 LIMIT 1`,
		sessionID,
	)

	m := &domain.Message{}
	var ts string
	err := row.Scan(&m.ID, &m.SessionID, &m.Role, &m.Content, &ts)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("message get_last_message: %w", err)
	}

	m.Timestamp, err = parseSQLiteTime(ts)
	if err != nil {
		return nil, fmt.Errorf("parse timestamp: %w", err)
	}
	return m, nil
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
