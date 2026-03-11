package domain

import "time"

// Message represents a single chat message within a session.
type Message struct {
	ID        string
	SessionID string
	Role      string // "user" or "assistant"
	Content   string
	CreatedAt time.Time
}
