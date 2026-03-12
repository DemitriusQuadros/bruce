// Package ai defines the LLM service interface and provides implementations.
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"bruce/internal/config"
	"bruce/internal/domain"
)

type claudeProvider struct {
	apiKey     string
	model      string
	maxTokens  int
	httpClient *http.Client
}

// NewClaudeProvider returns an LLMService backed by the Anthropic Claude API.
func NewClaudeProvider(cfg config.ClaudeConfig) LLMService {
	return &claudeProvider{
		apiKey:     cfg.APIKey,
		model:      cfg.Model,
		maxTokens:  cfg.MaxTokens,
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system,omitempty"`
	Messages  []anthropicMessage `json:"messages"`
}

type anthropicMessage struct {
	Role    string `json:"role"` // "user" | "assistant"
	Content string `json:"content"`
}

type anthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}

// GenerateResponse builds the Anthropic request, calls the API, and returns the reply text.
func (c *claudeProvider) GenerateResponse(ctx context.Context, systemPrompt string, history []domain.Message) (string, error) {
	// Sanitize history: filter system messages and merge consecutive same-role messages
	cleaned := sanitizeHistory(history)

	msgs := make([]anthropicMessage, len(cleaned))
	for i, m := range cleaned {
		msgs[i] = anthropicMessage{Role: m.Role, Content: m.Content}
	}

	reqBody := anthropicRequest{
		Model:     c.model,
		MaxTokens: c.maxTokens,
		System:    systemPrompt,
		Messages:  msgs,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.anthropic.com/v1/messages", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("http do: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	switch resp.StatusCode {
	case http.StatusOK:
		// handled below
	case 429, 529:
		return "", ErrRateLimited
	case 500, 502, 503:
		return "", ErrProviderDown
	case 400:
		return "", fmt.Errorf("%w: %s", ErrBadRequest, string(respBytes))
	default:
		return "", fmt.Errorf("claude API %d: %s", resp.StatusCode, string(respBytes))
	}

	var apiResp anthropicResponse
	if err := json.Unmarshal(respBytes, &apiResp); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	if len(apiResp.Content) == 0 {
		return "", fmt.Errorf("claude returned empty content")
	}
	return apiResp.Content[0].Text, nil
}
