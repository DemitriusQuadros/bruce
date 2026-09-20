package repository

import (
	"database/sql"
	"fmt"
	"time"

	"bruce/internal/domain"
)

// SessionSummaryRepository defines data access for conversation summaries.
type SessionSummaryRepository interface {
	Get(sessionID string) (*domain.SessionSummary, error)
	Upsert(summary *domain.SessionSummary) error
	Delete(sessionID string) error
}

type sqliteSessionSummaryRepo struct {
	db *sql.DB
}

// NewSessionSummaryRepository returns a SQLite-backed implementation of SessionSummaryRepository.
func NewSessionSummaryRepository(db *sql.DB) SessionSummaryRepository {
	return &sqliteSessionSummaryRepo{db: db}
}

func (r *sqliteSessionSummaryRepo) Get(sessionID string) (*domain.SessionSummary, error) {
	row := r.db.QueryRow(
		`SELECT session_id, summary, last_summarized_msg_id, message_count, updated_at
		 FROM session_summaries
		 WHERE session_id = ?`,
		sessionID,
	)

	var s domain.SessionSummary
	var updatedAtStr string
	err := row.Scan(&s.SessionID, &s.Summary, &s.LastSummarizedMsgID, &s.MessageCount, &updatedAtStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No summary found is not an error
		}
		return nil, fmt.Errorf("session_summary get: %w", err)
	}

	t, err := parseSQLiteTime(updatedAtStr)
	if err == nil {
		s.UpdatedAt = t
	} else {
		s.UpdatedAt = time.Now().UTC()
	}

	return &s, nil
}

func (r *sqliteSessionSummaryRepo) Upsert(summary *domain.SessionSummary) error {
	nowStr := time.Now().UTC().Format("2006-01-02 15:04:05")
	_, err := r.db.Exec(
		`INSERT INTO session_summaries (session_id, summary, last_summarized_msg_id, message_count, updated_at)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(session_id) DO UPDATE SET
		   summary = excluded.summary,
		   last_summarized_msg_id = excluded.last_summarized_msg_id,
		   message_count = excluded.message_count,
		   updated_at = excluded.updated_at`,
		summary.SessionID, summary.Summary, summary.LastSummarizedMsgID, summary.MessageCount, nowStr,
	)
	if err != nil {
		return fmt.Errorf("session_summary upsert: %w", err)
	}
	return nil
}

func (r *sqliteSessionSummaryRepo) Delete(sessionID string) error {
	_, err := r.db.Exec(`DELETE FROM session_summaries WHERE session_id = ?`, sessionID)
	if err != nil {
		return fmt.Errorf("session_summary delete: %w", err)
	}
	return nil
}
