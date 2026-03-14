package tools

import (
	"context"

	"bruce/internal/ai"
)

// Tool is the interface every tool must implement.
type Tool interface {
	Name() string
	Definition() ai.ToolDefinition
	Execute(ctx context.Context, input map[string]interface{}) (interface{}, error)
}

// ToolMetadata is a lightweight summary for introspection (no execution).
type ToolMetadata struct {
	Name        string
	Description string
	InputSchema map[string]interface{}
}
