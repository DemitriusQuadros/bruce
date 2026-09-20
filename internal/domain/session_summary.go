package domain

import "time"

// SessionSummary represents a condensed rolling summary of past messages in a session.
type SessionSummary struct {
	SessionID           string    `db:"session_id"`
	Summary             string    `db:"summary"`
	LastSummarizedMsgID string    `db:"last_summarized_msg_id"`
	MessageCount        int       `db:"message_count"`
	UpdatedAt           time.Time `db:"updated_at"`
}
