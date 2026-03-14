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
	Tools     []openaiTool    `json:"tools,omitempty"`
}

type openaiMessage struct {
	Role       string                   `json:"role"` // "system" | "user" | "assistant"
	Content    interface{}              `json:"content"` // string or []openaiContent
	ToolCalls  []openaiToolCall         `json:"tool_calls,omitempty"`
	ToolCallID string                   `json:"tool_call_id,omitempty"`
	Name       string                   `json:"name,omitempty"`
}

type openaiContent struct {
	Type      string `json:"type"` // "text" | "tool_result"
	Text      string `json:"text,omitempty"`
	ToolUseID string `json:"tool_use_id,omitempty"`
	Content   string `json:"content,omitempty"`
	IsError   bool   `json:"is_error,omitempty"`
}

type openaiToolCall struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"` // "function"
	Function openaiToolFunction     `json:"function"`
}

type openaiToolFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string
}

type openaiTool struct {
	Type     string              `json:"type"` // "function"
	Function openaiToolDef       `json:"function"`
}

type openaiToolDef struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

type openaiResponse struct {
	Choices []openaiChoice `json:"choices"`
}

type openaiChoice struct {
	Message      openaiMessage `json:"message"`
	FinishReason string        `json:"finish_reason"` // "stop" | "tool_calls"
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

	// Extract text from Content (can be string or []openaiContent)
	content := apiResp.Choices[0].Message.Content
	if str, ok := content.(string); ok {
		return str, nil
	}
	return "", nil
}

// GenerateWithTools calls OpenAI API with tool definitions and handles function calls.
func (o *openaiProvider) GenerateWithTools(ctx context.Context, systemPrompt string, messages []domain.Message, tools []ToolDefinition) (*ToolCallResponse, error) {
	cleaned := sanitizeHistory(messages)

	// Convert tools to OpenAI format
	openaiTools := make([]openaiTool, len(tools))
	for i, tool := range tools {
		openaiTools[i] = openaiTool{
			Type: "function",
			Function: openaiToolDef{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.InputSchema,
			},
		}
	}

	// Build messages
	msgs := make([]openaiMessage, 0)

	// System prompt
	if systemPrompt != "" {
		msgs = append(msgs, openaiMessage{
			Role:    "system",
			Content: systemPrompt,
		})
	}

	// Conversation history
	for _, m := range cleaned {
		if m.Type == "tool_result" && m.ToolResult != nil {
			// Tool result message
			msgs = append(msgs, openaiMessage{
				Role:       "tool",
				ToolCallID: m.ToolResult.ID,
				Content:    m.ToolResult.Content,
			})
		} else if m.Type == "tool_call" {
			// Assistant tool call message (we'll skip this; OpenAI doesn't need it separately)
			continue
		} else {
			// Regular text message
			msgs = append(msgs, openaiMessage{
				Role:    m.Role,
				Content: m.Content,
			})
		}
	}

	reqBody := openaiRequest{
		Model:     o.model,
		MaxTokens: o.maxTokens,
		Messages:  msgs,
		Tools:     openaiTools,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		o.baseURL+"/v1/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+o.apiKey)

	resp, err := o.httpClient.Do(httpReq)
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
	case 429:
		return nil, ErrRateLimited
	case 500, 502, 503:
		return nil, ErrProviderDown
	case 400:
		return nil, fmt.Errorf("%w: %s", ErrBadRequest, string(respBytes))
	default:
		return nil, fmt.Errorf("openai API %d: %s", resp.StatusCode, string(respBytes))
	}

	var apiResp openaiResponse
	if err := json.Unmarshal(respBytes, &apiResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("openai returned empty choices")
	}

	choice := apiResp.Choices[0]
	text := ""

	// Extract text content
	if content, ok := choice.Message.Content.(string); ok {
		text = content
	}

	// Check for tool calls
	if choice.FinishReason == "tool_calls" && len(choice.Message.ToolCalls) > 0 {
		var toolCalls []ToolCall
		for _, tc := range choice.Message.ToolCalls {
			// Parse the arguments JSON string
			var args map[string]interface{}
			if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
				args = make(map[string]interface{})
			}

			toolCalls = append(toolCalls, ToolCall{
				ID:    tc.ID,
				Name:  tc.Function.Name,
				Input: args,
			})
		}
		return &ToolCallResponse{
			ToolCalls: toolCalls,
			Text:      text,
			Complete:  false,
		}, nil
	}

	return &ToolCallResponse{
		Text:     text,
		Complete: true,
	}, nil
}
