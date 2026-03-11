// Package domain defines the core domain types for Bruce.
package domain

import "time"

// Session represents a conversation session with an AI assistant.
type Session struct {
	ID           string
	Name         string
	SystemPrompt string
	Connector    string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
