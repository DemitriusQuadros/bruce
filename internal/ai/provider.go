package ai

import (
	"context"
	"errors"

	"bruce/internal/domain"
)

// LLMService is the abstraction over any LLM provider.
// The interface is unchanged from Spec 03; all complexity is hidden behind implementations.
type LLMService interface {
	GenerateResponse(ctx context.Context, systemPrompt string, history []domain.Message) (string, error)
}

// Sentinel errors — provider-agnostic, used by the worker for retry decisions.
var (
	ErrRateLimited     = errors.New("ai: rate limited")         // 429 from any provider
	ErrProviderDown    = errors.New("ai: provider unavailable") // 5xx from any provider
	ErrBadRequest      = errors.New("ai: bad request")          // 4xx non-auth (don't retry)
	ErrUnknownProvider = errors.New("ai: unknown provider")
)

// ProviderName is the canonical string key for each provider.
// Used in config, session overrides, and the registry.
type ProviderName string

const (
	ProviderClaude ProviderName = "claude"
	ProviderGemini ProviderName = "gemini"
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
