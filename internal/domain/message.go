package domain

import "time"

// ToolResult holds the output from a tool execution.
// Populated when Message.Type == "tool_result".
type ToolResult struct {
	ID      string // Tool call ID from LLM (echoed back to provider)
	Content string // Execution output (or error text)
	IsError bool   // true when tool execution failed
}

// Message represents a single chat message within a session.
type Message struct {
	ID        string    `db:"id"`
	SessionID string    `db:"session_id"`
	Role      string    `db:"role"` // "user" | "assistant" | "system"
	Content   string    `db:"content"`
	Timestamp time.Time `db:"timestamp"`

	// Tool message fields (not persisted to DB — ephemeral)
	Type       string      // "text" | "tool_call" | "tool_result" (empty = "text" for existing rows)
	ToolResult *ToolResult // non-nil only when Type == "tool_result"
}
