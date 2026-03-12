// Package domain defines the core domain types for Bruce.
package domain

import "time"

// Session represents a conversation session with an AI assistant.
type Session struct {
	ID               string    `db:"id"`
	ConnectorType    string    `db:"connector_type"` // "whatsapp" | "discord"
	ChannelID        string    `db:"channel_id"`
	SystemPrompt     string    `db:"system_prompt"`
	IsActive         bool      `db:"is_active"`
	ProviderOverride string    `db:"provider_override"` // "" | "claude" | "gemini"
	CreatedAt        time.Time `db:"created_at"`
	UpdatedAt        time.Time `db:"updated_at"`
}
