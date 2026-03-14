package ai

import (
	"context"
	"errors"

	"bruce/internal/domain"
)

// ToolDefinition describes a tool that an LLM can use.
type ToolDefinition struct {
	Name        string                 // Tool name (must match what registry exposes)
	Description string                 // Human-readable description for the LLM
	InputSchema map[string]interface{} // JSON Schema describing parameters
}

// ToolCall represents a tool invocation requested by the LLM.
type ToolCall struct {
	ID    string                 // Unique ID assigned by LLM provider
	Name  string                 // Tool name to invoke
	Input map[string]interface{} // Parsed tool arguments
}

// ToolCallResponse is returned by GenerateWithTools.
type ToolCallResponse struct {
	Text      string     // Final text (non-empty when Complete == true)
	ToolCalls []ToolCall // Tool calls requested by LLM (non-empty when not Complete)
	Complete  bool       // true when no more tool calls are expected
}

// LLMService is the abstraction over any LLM provider.
type LLMService interface {
	GenerateResponse(ctx context.Context, systemPrompt string, history []domain.Message) (string, error)
	GenerateWithTools(ctx context.Context, systemPrompt string, messages []domain.Message, tools []ToolDefinition) (*ToolCallResponse, error)
}

// Sentinel errors — provider-agnostic, used by the worker for retry decisions.
var (
	ErrRateLimited       = errors.New("ai: rate limited")         // 429 from any provider
	ErrProviderDown      = errors.New("ai: provider unavailable") // 5xx from any provider
	ErrBadRequest        = errors.New("ai: bad request")          // 4xx non-auth (don't retry)
	ErrUnknownProvider   = errors.New("ai: unknown provider")
	ErrMaxRetriesExceeded = errors.New("ai: agent loop max retries exceeded")
)

// ProviderName is the canonical string key for each provider.
// Used in config, session overrides, and the registry.
type ProviderName string

const (
	ProviderClaude ProviderName = "claude"
	ProviderGemini ProviderName = "gemini"
	ProviderOpenAI ProviderName = "openai"
)

// contextKey type for session ID propagation
type contextKey string

const contextKeySessionID contextKey = "session_id"

// WithSessionID injects a session ID into the context for provider resolution.
func WithSessionID(ctx context.Context, sessionID string) context.Context {
	return context.WithValue(ctx, contextKeySessionID, sessionID)
}

// sessionIDFromContext extracts the session ID from context if present.
func sessionIDFromContext(ctx context.Context) (string, bool) {
	sessionID, ok := ctx.Value(contextKeySessionID).(string)
	return sessionID, ok && sessionID != ""
}
