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
	Tools     []anthropicTool    `json:"tools,omitempty"`
}

type anthropicMessage struct {
	Role    string      `json:"role"` // "user" | "assistant"
	Content interface{} `json:"content"` // string or []anthropicContent (for tool results)
}

type anthropicContent struct {
	Type      string      `json:"type"` // "text", "tool_use", "tool_result"
	Text      string      `json:"text,omitempty"`
	ID        string      `json:"id,omitempty"`
	Name      string      `json:"name,omitempty"`
	Input     interface{} `json:"input,omitempty"`
	ToolUseID string      `json:"tool_use_id,omitempty"`
	Content   string      `json:"content,omitempty"`
	IsError   bool        `json:"is_error,omitempty"`
}

type anthropicTool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"input_schema"`
}

type anthropicResponse struct {
	Content   []struct {
		Type  string      `json:"type"` // "text" or "tool_use"
		Text  string      `json:"text,omitempty"`
		ID    string      `json:"id,omitempty"`
		Name  string      `json:"name,omitempty"`
		Input interface{} `json:"input,omitempty"`
	} `json:"content"`
	StopReason string `json:"stop_reason"` // "end_turn" or "tool_use"
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

// GenerateWithTools calls the Anthropic API with tool definitions and returns tool calls or final text.
func (c *claudeProvider) GenerateWithTools(ctx context.Context, systemPrompt string, messages []domain.Message, tools []ToolDefinition) (*ToolCallResponse, error) {
	cleaned := sanitizeHistory(messages)

	// Build message array, handling tool_use and tool_result specially.
	msgs := make([]anthropicMessage, len(cleaned))
	for i, m := range cleaned {
		switch {
		case m.Type == "tool_call" && len(m.ToolCalls) > 0:
			// Reconstruct the assistant tool_use content blocks Claude requires.
			// Each tool_use block must match a tool_result block in the next user message.
			parts := make([]anthropicContent, 0, len(m.ToolCalls)+1)
			if m.Content != "" {
				parts = append(parts, anthropicContent{Type: "text", Text: m.Content})
			}
			for _, tc := range m.ToolCalls {
				parts = append(parts, anthropicContent{
					Type:  "tool_use",
					ID:    tc.ID,
					Name:  tc.Name,
					Input: tc.Input,
				})
			}
			msgs[i] = anthropicMessage{Role: m.Role, Content: parts}

		case m.Type == "tool_result" && m.ToolResult != nil:
			// Tool results must be sent as content arrays with matching tool_use_id.
			msgs[i] = anthropicMessage{
				Role: m.Role,
				Content: []anthropicContent{
					{
						Type:      "tool_result",
						ToolUseID: m.ToolResult.ID,
						Content:   m.ToolResult.Content,
						IsError:   m.ToolResult.IsError,
					},
				},
			}

		default:
			msgs[i] = anthropicMessage{Role: m.Role, Content: m.Content}
		}
	}

	// Convert tool definitions for Anthropic
	anthropicTools := make([]anthropicTool, len(tools))
	for i, tool := range tools {
		anthropicTools[i] = anthropicTool{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: tool.InputSchema,
		}
	}

	reqBody := anthropicRequest{
		Model:     c.model,
		MaxTokens: c.maxTokens,
		System:    systemPrompt,
		Messages:  msgs,
		Tools:     anthropicTools,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.anthropic.com/v1/messages", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http do: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	switch resp.StatusCode {
	case http.StatusOK:
		// handled below
	case 429, 529:
		return nil, ErrRateLimited
	case 500, 502, 503:
		return nil, ErrProviderDown
	case 400:
		return nil, fmt.Errorf("%w: %s", ErrBadRequest, string(respBytes))
	default:
		return nil, fmt.Errorf("claude API %d: %s", resp.StatusCode, string(respBytes))
	}

	var apiResp anthropicResponse
	if err := json.Unmarshal(respBytes, &apiResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	// Parse response based on stop reason
	if apiResp.StopReason == "tool_use" {
		// Extract tool calls from content
		var toolCalls []ToolCall
		for _, content := range apiResp.Content {
			if content.Type == "tool_use" {
				// Parse input as map
				var inputMap map[string]interface{}
				switch v := content.Input.(type) {
				case map[string]interface{}:
					inputMap = v
				case string:
					if err := json.Unmarshal([]byte(v), &inputMap); err != nil {
						inputMap = make(map[string]interface{})
					}
				default:
					inputMap = make(map[string]interface{})
				}

				toolCalls = append(toolCalls, ToolCall{
					ID:    content.ID,
					Name:  content.Name,
					Input: inputMap,
				})
			}
		}
		return &ToolCallResponse{
			ToolCalls: toolCalls,
			Complete:  false,
		}, nil
	}

	// Extract final text from response
	var finalText string
	for _, content := range apiResp.Content {
		if content.Type == "text" && content.Text != "" {
			finalText = content.Text
			break
		}
	}

	return &ToolCallResponse{
		Text:      finalText,
		Complete:  true,
	}, nil
}
