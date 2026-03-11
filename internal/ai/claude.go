// Package ai defines the LLM service interface and provides the Claude implementation.
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"bruce/internal/config"
	"bruce/internal/domain"
)

// ErrRateLimited is returned when the Claude API responds with 429 or 529.
// Asynq checks for this and retries the task with exponential backoff.
var ErrRateLimited = errors.New("claude: rate limited")

// LLMService is the abstraction over any LLM provider.
// Claude is the only implementation for MVP; the interface makes testing trivial.
type LLMService interface {
	GenerateResponse(ctx context.Context, systemPrompt string, history []domain.Message) (string, error)
}

type claudeService struct {
	apiKey     string
	model      string
	maxTokens  int
	httpClient *http.Client
}

// NewClaudeService returns an LLMService backed by the Anthropic Claude API.
func NewClaudeService(cfg *config.Config) LLMService {
	return &claudeService{
		apiKey:     cfg.Claude.APIKey,
		model:      cfg.Claude.Model,
		maxTokens:  cfg.Claude.MaxTokens,
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
func (c *claudeService) GenerateResponse(ctx context.Context, systemPrompt string, history []domain.Message) (string, error) {
	// Separate system-role messages from the conversation; concatenate into the system field.
	systemParts := []string{}
	if systemPrompt != "" {
		systemParts = append(systemParts, systemPrompt)
	}
	var conv []domain.Message
	for _, m := range history {
		if m.Role == "system" {
			systemParts = append(systemParts, m.Content)
		} else {
			conv = append(conv, m)
		}
	}

	// Anthropic requires strict user/assistant alternation — merge consecutive same-role messages.
	merged := mergeConsecutive(conv)

	msgs := make([]anthropicMessage, len(merged))
	for i, m := range merged {
		msgs[i] = anthropicMessage{Role: m.Role, Content: m.Content}
	}

	reqBody := anthropicRequest{
		Model:     c.model,
		MaxTokens: c.maxTokens,
		System:    strings.Join(systemParts, "\n"),
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

// mergeConsecutive joins consecutive messages with the same role using a newline.
func mergeConsecutive(msgs []domain.Message) []domain.Message {
	if len(msgs) == 0 {
		return msgs
	}
	result := []domain.Message{msgs[0]}
	for _, m := range msgs[1:] {
		last := &result[len(result)-1]
		if last.Role == m.Role {
			last.Content = last.Content + "\n" + m.Content
		} else {
			result = append(result, m)
		}
	}
	return result
}
