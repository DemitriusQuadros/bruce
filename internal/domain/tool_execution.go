package domain

import "time"

// ToolExecution records a single tool call made by the LLM during a session.
type ToolExecution struct {
	ID         string
	SessionID  string
	ToolName   string
	Input      string // JSON
	Output     string // JSON or plain text
	LatencyMS  int64
	Success    bool
	ErrorMsg   string
	ExecutedAt time.Time
}
