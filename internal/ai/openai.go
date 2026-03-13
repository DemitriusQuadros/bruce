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

type openaiProvider struct {
	apiKey     string
	model      string
	maxTokens  int
	httpClient *http.Client
	baseURL    string
}

// NewOpenAIProvider returns an LLMService backed by the OpenAI Chat Completions API.
func NewOpenAIProvider(cfg config.OpenAIConfig) LLMService {
	return &openaiProvider{
		apiKey:     cfg.APIKey,
		model:      cfg.Model,
		maxTokens:  cfg.MaxTokens,
		httpClient: &http.Client{Timeout: 60 * time.Second},
		baseURL:    "https://api.openai.com",
	}
}

type openaiRequest struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_tokens"`
	Messages  []openaiMessage `json:"messages"`
}

type openaiMessage struct {
	Role    string `json:"role"` // "system" | "user" | "assistant"
	Content string `json:"content"`
}

type openaiResponse struct {
	Choices []openaiChoice `json:"choices"`
}

type openaiChoice struct {
	Message openaiMessage `json:"message"`
}

// GenerateResponse builds the OpenAI request, calls the API, and returns the reply text.
func (o *openaiProvider) GenerateResponse(ctx context.Context, systemPrompt string, history []domain.Message) (string, error) {
	cleaned := sanitizeHistory(history)

	// Build messages: system prompt first, then conversation history.
	msgs := make([]openaiMessage, 0, len(cleaned)+1)

	if systemPrompt != "" {
		msgs = append(msgs, openaiMessage{Role: "system", Content: systemPrompt})
	}

	for _, m := range cleaned {
		msgs = append(msgs, openaiMessage{Role: m.Role, Content: m.Content})
	}

	reqBody := openaiRequest{
		Model:     o.model,
		MaxTokens: o.maxTokens,
		Messages:  msgs,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		o.baseURL+"/v1/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+o.apiKey)

	resp, err := o.httpClient.Do(httpReq)
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
	case 429:
		return "", ErrRateLimited
	case 500, 502, 503:
		return "", ErrProviderDown
	case 400:
		return "", fmt.Errorf("%w: %s", ErrBadRequest, string(respBytes))
	default:
		return "", fmt.Errorf("openai API %d: %s", resp.StatusCode, string(respBytes))
	}

	var apiResp openaiResponse
	if err := json.Unmarshal(respBytes, &apiResp); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	if len(apiResp.Choices) == 0 {
		return "", fmt.Errorf("openai returned empty choices")
	}
	return apiResp.Choices[0].Message.Content, nil
}
