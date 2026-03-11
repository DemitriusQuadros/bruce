// Package ai defines the LLM service interface and provides a stub implementation.
package ai

import (
	"context"
	"log"

	"bruce/internal/config"
	"bruce/internal/domain"
)

// LLMService is the interface for interacting with a language model.
type LLMService interface {
	Complete(ctx context.Context, systemPrompt string, messages []domain.Message) (string, error)
}

// stubClaudeService is a no-op implementation used until the real Claude client is wired.
type stubClaudeService struct {
	cfg config.ClaudeConfig
}

// NewClaudeService returns an LLMService backed by the Anthropic Claude API.
// In this spec the implementation is a stub that logs "not implemented".
func NewClaudeService(cfg config.ClaudeConfig) LLMService {
	return &stubClaudeService{cfg: cfg}
}

// Complete logs a warning and returns an empty string until fully implemented.
func (s *stubClaudeService) Complete(ctx context.Context, systemPrompt string, messages []domain.Message) (string, error) {
	log.Println("WARNING: claude.Complete not implemented — returning empty response")
	return "", nil
}
