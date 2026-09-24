// Package repository provides database access implementations for Bruce domain types.
package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"bruce/internal/domain"
)

// SessionRepository defines the data access contract for sessions.
type SessionRepository interface {
	FindOrCreate(connectorType, channelID string) (*domain.Session, error)
	GetAll() ([]*domain.Session, error)
	GetByID(id string) (*domain.Session, error)
	GetByConnectorType(connectorType string) ([]*domain.Session, error)
	Create(connectorType, channelID, title string) (*domain.Session, error)
	Delete(id string) error
	UpdateTitle(id, title string) error
	UpdateSystemPrompt(id, prompt string) error
	SetActive(id string, active bool) error
	UpdateProviderOverride(id, provider string) error
}

// SQLiteSessionRepository is the SQLite-backed implementation of SessionRepository.
type SQLiteSessionRepository struct {
	db *sql.DB
}

// NewSessionRepository returns a new SQLiteSessionRepository.
func NewSessionRepository(db *sql.DB) SessionRepository {
	return &SQLiteSessionRepository{db: db}
}

// FindOrCreate returns the existing session for (connectorType, channelID), or inserts a new one.
func (r *SQLiteSessionRepository) FindOrCreate(connectorType, channelID string) (*domain.Session, error) {
	id := uuid.New().String()
	_, err := r.db.Exec(
		`INSERT OR IGNORE INTO sessions (id, connector_type, channel_id, title, system_prompt, is_active, created_at, updated_at)
		 VALUES (?, ?, ?, '', '', 1, datetime('now'), datetime('now'))`,
		id, connectorType, channelID,
	)
	if err != nil {
		return nil, fmt.Errorf("session find_or_create insert: %w", err)
	}

	row := r.db.QueryRow(
		`SELECT id, connector_type, channel_id, title, system_prompt, is_active, provider_override, created_at, updated_at
		 FROM sessions WHERE connector_type = ? AND channel_id = ?`,
		connectorType, channelID,
	)
	return scanSession(row)
}

// GetAll returns all sessions ordered by creation time descending.
func (r *SQLiteSessionRepository) GetAll() ([]*domain.Session, error) {
	rows, err := r.db.Query(
		`SELECT id, connector_type, channel_id, title, system_prompt, is_active, provider_override, created_at, updated_at
		 FROM sessions ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("session get_all: %w", err)
	}
	defer rows.Close()

	var sessions []*domain.Session
	for rows.Next() {
		s, err := scanSessionRows(rows)
		if err != nil {
			return nil, fmt.Errorf("session get_all scan: %w", err)
		}
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

// GetByID fetches a session by its primary key.
func (r *SQLiteSessionRepository) GetByID(id string) (*domain.Session, error) {
	row := r.db.QueryRow(
		`SELECT id, connector_type, channel_id, title, system_prompt, is_active, provider_override, created_at, updated_at
		 FROM sessions WHERE id = ?`,
		id,
	)
	s, err := scanSession(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("session not found: %s", id)
		}
		return nil, fmt.Errorf("session get_by_id: %w", err)
	}
	return s, nil
}

// UpdateSystemPrompt sets the system_prompt for a session.
func (r *SQLiteSessionRepository) UpdateSystemPrompt(id, prompt string) error {
	_, err := r.db.Exec(
		`UPDATE sessions SET system_prompt = ?, updated_at = datetime('now') WHERE id = ?`,
		prompt, id,
	)
	if err != nil {
		return fmt.Errorf("session update_system_prompt: %w", err)
	}
	return nil
}

// SetActive toggles the is_active flag for a session.
func (r *SQLiteSessionRepository) SetActive(id string, active bool) error {
	_, err := r.db.Exec(
		`UPDATE sessions SET is_active = ?, updated_at = datetime('now') WHERE id = ?`,
		active, id,
	)
	if err != nil {
		return fmt.Errorf("session set_active: %w", err)
	}
	return nil
}

// UpdateProviderOverride sets the provider_override for a session.
func (r *SQLiteSessionRepository) UpdateProviderOverride(id, provider string) error {
	_, err := r.db.Exec(
		`UPDATE sessions SET provider_override = ?, updated_at = datetime('now') WHERE id = ?`,
		provider, id,
	)
	if err != nil {
		return fmt.Errorf("session update_provider_override: %w", err)
	}
	return nil
}

// GetByConnectorType returns all sessions for a given connector type.
func (r *SQLiteSessionRepository) GetByConnectorType(connectorType string) ([]*domain.Session, error) {
	rows, err := r.db.Query(
		`SELECT id, connector_type, channel_id, title, system_prompt, is_active, provider_override, created_at, updated_at
		 FROM sessions WHERE connector_type = ? ORDER BY updated_at DESC, rowid DESC`,
		connectorType,
	)
	if err != nil {
		return nil, fmt.Errorf("session get_by_connector_type: %w", err)
	}
	defer rows.Close()

	var sessions []*domain.Session
	for rows.Next() {
		s, err := scanSessionRows(rows)
		if err != nil {
			return nil, fmt.Errorf("session get_by_connector_type scan: %w", err)
		}
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

// Create inserts a new session and returns it.
func (r *SQLiteSessionRepository) Create(connectorType, channelID, title string) (*domain.Session, error) {
	id := uuid.New().String()
	_, err := r.db.Exec(
		`INSERT INTO sessions (id, connector_type, channel_id, title, system_prompt, is_active, created_at, updated_at)
		 VALUES (?, ?, ?, ?, '', 1, datetime('now'), datetime('now'))`,
		id, connectorType, channelID, title,
	)
	if err != nil {
		return nil, fmt.Errorf("session create: %w", err)
	}

	row := r.db.QueryRow(
		`SELECT id, connector_type, channel_id, title, system_prompt, is_active, provider_override, created_at, updated_at
		 FROM sessions WHERE id = ?`,
		id,
	)
	return scanSession(row)
}

// Delete removes a session by ID. CASCADE handles messages.
func (r *SQLiteSessionRepository) Delete(id string) error {
	_, err := r.db.Exec(`DELETE FROM sessions WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("session delete: %w", err)
	}
	return nil
}

// UpdateTitle sets the title for a session.
func (r *SQLiteSessionRepository) UpdateTitle(id, title string) error {
	_, err := r.db.Exec(
		`UPDATE sessions SET title = ?, updated_at = datetime('now') WHERE id = ?`,
		title, id,
	)
	if err != nil {
		return fmt.Errorf("session update_title: %w", err)
	}
	return nil
}

// scanSession scans a *sql.Row into a domain.Session.
func scanSession(row *sql.Row) (*domain.Session, error) {
	s := &domain.Session{}
	var createdAt, updatedAt string
	if err := row.Scan(&s.ID, &s.ConnectorType, &s.ChannelID, &s.Title, &s.SystemPrompt, &s.IsActive, &s.ProviderOverride, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	var err error
	s.CreatedAt, err = parseSQLiteTime(createdAt)
	if err != nil {
		return nil, fmt.Errorf("parse created_at: %w", err)
	}
	s.UpdatedAt, err = parseSQLiteTime(updatedAt)
	if err != nil {
		return nil, fmt.Errorf("parse updated_at: %w", err)
	}
	return s, nil
}

// scanSessionRows scans a *sql.Rows into a domain.Session.
func scanSessionRows(rows *sql.Rows) (*domain.Session, error) {
	s := &domain.Session{}
	var createdAt, updatedAt string
	if err := rows.Scan(&s.ID, &s.ConnectorType, &s.ChannelID, &s.Title, &s.SystemPrompt, &s.IsActive, &s.ProviderOverride, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	var err error
	s.CreatedAt, err = parseSQLiteTime(createdAt)
	if err != nil {
		return nil, fmt.Errorf("parse created_at: %w", err)
	}
	s.UpdatedAt, err = parseSQLiteTime(updatedAt)
	if err != nil {
		return nil, fmt.Errorf("parse updated_at: %w", err)
	}
	return s, nil
}

// parseSQLiteTime parses the datetime strings produced by SQLite's datetime('now').
var sqliteTimeFormats = []string{
	"2006-01-02 15:04:05.000000000",
	"2006-01-02 15:04:05",
	time.RFC3339,
	"2006-01-02T15:04:05Z",
}

func parseSQLiteTime(s string) (time.Time, error) {
	for _, f := range sqliteTimeFormats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse sqlite time %q", s)
}
