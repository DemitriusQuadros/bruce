// Package repository provides database access implementations for Bruce domain types.
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"bruce/internal/domain"
)

// SessionRepository defines the data access contract for sessions.
type SessionRepository interface {
	Create(ctx context.Context, s *domain.Session) error
	GetByID(ctx context.Context, id string) (*domain.Session, error)
	List(ctx context.Context) ([]*domain.Session, error)
	Update(ctx context.Context, s *domain.Session) error
	Delete(ctx context.Context, id string) error
}

// SQLiteSessionRepository is the SQLite-backed implementation of SessionRepository.
type SQLiteSessionRepository struct {
	db *sql.DB
}

// NewSessionRepository returns a new SQLiteSessionRepository.
func NewSessionRepository(db *sql.DB) SessionRepository {
	return &SQLiteSessionRepository{db: db}
}

// Create inserts a new session record.
func (r *SQLiteSessionRepository) Create(ctx context.Context, s *domain.Session) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO sessions (id, name, system_prompt, connector, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		s.ID, s.Name, s.SystemPrompt, s.Connector, s.CreatedAt, s.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("session create: %w", err)
	}
	return nil
}

// GetByID fetches a session by its primary key.
func (r *SQLiteSessionRepository) GetByID(ctx context.Context, id string) (*domain.Session, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, name, system_prompt, connector, created_at, updated_at FROM sessions WHERE id = ?`, id,
	)
	s := &domain.Session{}
	if err := row.Scan(&s.ID, &s.Name, &s.SystemPrompt, &s.Connector, &s.CreatedAt, &s.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("session not found: %s", id)
		}
		return nil, fmt.Errorf("session get: %w", err)
	}
	return s, nil
}

// List returns all sessions ordered by creation time descending.
func (r *SQLiteSessionRepository) List(ctx context.Context) ([]*domain.Session, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, system_prompt, connector, created_at, updated_at FROM sessions ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("session list: %w", err)
	}
	defer rows.Close()

	var sessions []*domain.Session
	for rows.Next() {
		s := &domain.Session{}
		if err := rows.Scan(&s.ID, &s.Name, &s.SystemPrompt, &s.Connector, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("session list scan: %w", err)
		}
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

// Update modifies an existing session record.
func (r *SQLiteSessionRepository) Update(ctx context.Context, s *domain.Session) error {
	s.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(ctx,
		`UPDATE sessions SET name = ?, system_prompt = ?, connector = ?, updated_at = ? WHERE id = ?`,
		s.Name, s.SystemPrompt, s.Connector, s.UpdatedAt, s.ID,
	)
	if err != nil {
		return fmt.Errorf("session update: %w", err)
	}
	return nil
}

// Delete removes a session and all its associated messages (via FK cascade).
func (r *SQLiteSessionRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("session delete: %w", err)
	}
	return nil
}
