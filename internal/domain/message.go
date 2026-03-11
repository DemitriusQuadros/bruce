package domain

import "time"

// Message represents a single chat message within a session.
type Message struct {
	ID        string    `db:"id"`
	SessionID string    `db:"session_id"`
	Role      string    `db:"role"` // "user" | "assistant" | "system"
	Content   string    `db:"content"`
	Timestamp time.Time `db:"timestamp"`
}
