package repository

import (
	"context"
	"database/sql"
	"fmt"

	"bruce/internal/domain"
)

// MessageRepository defines the data access contract for messages.
type MessageRepository interface {
	Create(ctx context.Context, m *domain.Message) error
	ListBySession(ctx context.Context, sessionID string) ([]*domain.Message, error)
	Delete(ctx context.Context, id string) error
}

// SQLiteMessageRepository is the SQLite-backed implementation of MessageRepository.
type SQLiteMessageRepository struct {
	db *sql.DB
}

// NewMessageRepository returns a new SQLiteMessageRepository.
func NewMessageRepository(db *sql.DB) MessageRepository {
	return &SQLiteMessageRepository{db: db}
}

// Create inserts a new message record.
func (r *SQLiteMessageRepository) Create(ctx context.Context, m *domain.Message) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO messages (id, session_id, role, content, created_at) VALUES (?, ?, ?, ?, ?)`,
		m.ID, m.SessionID, m.Role, m.Content, m.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("message create: %w", err)
	}
	return nil
}

// ListBySession returns all messages for a session ordered by creation time ascending.
func (r *SQLiteMessageRepository) ListBySession(ctx context.Context, sessionID string) ([]*domain.Message, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, session_id, role, content, created_at FROM messages WHERE session_id = ? ORDER BY created_at ASC`,
		sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("message list: %w", err)
	}
	defer rows.Close()

	var messages []*domain.Message
	for rows.Next() {
		m := &domain.Message{}
		if err := rows.Scan(&m.ID, &m.SessionID, &m.Role, &m.Content, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("message list scan: %w", err)
		}
		messages = append(messages, m)
	}
	return messages, rows.Err()
}

// Delete removes a message by its primary key.
func (r *SQLiteMessageRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM messages WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("message delete: %w", err)
	}
	return nil
}
