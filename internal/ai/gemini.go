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

type geminiProvider struct {
	apiKey     string
	model      string
	maxTokens  int
	httpClient *http.Client
	baseURL    string
}

// NewGeminiProvider returns an LLMService backed by Google Gemini API.
func NewGeminiProvider(cfg config.GeminiConfig) LLMService {
	return &geminiProvider{
		apiKey:     cfg.APIKey,
		model:      cfg.Model,
		maxTokens:  cfg.MaxTokens,
		httpClient: &http.Client{Timeout: 60 * time.Second},
		baseURL:    "https://generativelanguage.googleapis.com/v1beta/models",
	}
}

// Gemini API request shape
type geminiRequest struct {
	SystemInstruction *geminiContent  `json:"system_instruction,omitempty"`
	Contents          []geminiContent `json:"contents"`
	GenerationConfig  geminiGenConfig `json:"generation_config"`
	Tools             []geminiTool    `json:"tools,omitempty"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"` // "user" | "model"
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text             string                  `json:"text,omitempty"`
	FunctionCall     *geminiFunctionCall     `json:"functionCall,omitempty"`
	FunctionResponse *geminiFunctionResponse `json:"functionResponse,omitempty"`
}

type geminiFunctionCall struct {
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"args"`
}

type geminiFunctionResponse struct {
	Name     string                 `json:"name"`
	Response map[string]interface{} `json:"response"` // Must be a JSON object (Struct), not a plain string
}

type geminiGenConfig struct {
	MaxOutputTokens int     `json:"maxOutputTokens"`
	Temperature     float32 `json:"temperature,omitempty"`
}

type geminiTool struct {
	FunctionDeclarations []geminiFunctionDecl `json:"functionDeclarations"`
}

type geminiFunctionDecl struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

// Gemini API response shape
type geminiResponse struct {
	Candidates []struct {
		Content geminiContent `json:"content"`
	} `json:"candidates"`
	PromptFeedback *struct {
		BlockReason string `json:"blockReason"`
	} `json:"promptFeedback"`
}

// GenerateResponse calls the Gemini API and returns the response text.
func (g *geminiProvider) GenerateResponse(ctx context.Context, systemPrompt string, history []domain.Message) (string, error) {
	messages := sanitizeHistory(history)

	reqBody := geminiRequest{
		GenerationConfig: geminiGenConfig{MaxOutputTokens: g.maxTokens},
		Contents:         mapToGeminiContents(messages),
	}

	// System prompt — Gemini uses system_instruction, not a message role
	if systemPrompt != "" {
		reqBody.SystemInstruction = &geminiContent{
			Parts: []geminiPart{{Text: systemPrompt}},
		}
	}

	// URL: /v1beta/models/{model}:generateContent?key={apiKey}
	url := fmt.Sprintf("%s/%s:generateContent?key=%s", g.baseURL, g.model, g.apiKey)

	data, _ := json.Marshal(reqBody)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrProviderDown, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	// Error mapping
	switch resp.StatusCode {
	case 429:
		return "", ErrRateLimited
	case 500, 502, 503:
		return "", ErrProviderDown
	case 400:
		return "", fmt.Errorf("%w: %s", ErrBadRequest, string(body))
	}

	var geminiResp geminiResponse
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return "", fmt.Errorf("gemini: decode response: %w", err)
	}

	// Safety filter check
	if geminiResp.PromptFeedback != nil && geminiResp.PromptFeedback.BlockReason != "" {
		return "", fmt.Errorf("%w: content blocked by Gemini safety filter (%s)",
			ErrBadRequest, geminiResp.PromptFeedback.BlockReason)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini: empty response")
	}

	return geminiResp.Candidates[0].Content.Parts[0].Text, nil
}

// GenerateWithTools calls Gemini API with tool definitions and handles function calls.
func (g *geminiProvider) GenerateWithTools(ctx context.Context, systemPrompt string, messages []domain.Message, tools []ToolDefinition) (*ToolCallResponse, error) {
	sanitized := sanitizeHistory(messages)

	// Convert tools to Gemini format
	geminiTools := make([]geminiTool, 0)
	if len(tools) > 0 {
		funcs := make([]geminiFunctionDecl, len(tools))
		for i, tool := range tools {
			funcs[i] = geminiFunctionDecl{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  cleanGeminiSchema(tool.InputSchema),
			}
		}
		geminiTools = append(geminiTools, geminiTool{FunctionDeclarations: funcs})
	}

	// Convert messages to Gemini format
	contents := make([]geminiContent, 0)
	for _, m := range sanitized {
		role := m.Role
		if role == "assistant" {
			role = "model"
		}
		if role == "system" {
			continue // handled via system_instruction
		}
		// Skip empty tool_call placeholders added by the agent loop for Claude compatibility.
		// Gemini rejects Parts with no initialized data field.
		if m.Type == "tool_call" && m.Content == "" {
			continue
		}

		// Handle tool results — Gemini requires response to be a JSON object (Struct)
		if m.Type == "tool_result" && m.ToolResult != nil {
			contents = append(contents, geminiContent{
				Role: role,
				Parts: []geminiPart{
					{
						FunctionResponse: &geminiFunctionResponse{
							Name:     m.ToolResult.ID,
							Response: map[string]interface{}{"output": m.ToolResult.Content},
						},
					},
				},
			})
		} else {
			contents = append(contents, geminiContent{
				Role:  role,
				Parts: []geminiPart{{Text: m.Content}},
			})
		}
	}

	reqBody := geminiRequest{
		GenerationConfig: geminiGenConfig{MaxOutputTokens: g.maxTokens},
		Contents:         contents,
		Tools:            geminiTools,
	}

	if systemPrompt != "" {
		reqBody.SystemInstruction = &geminiContent{
			Parts: []geminiPart{{Text: systemPrompt}},
		}
	}

	url := fmt.Sprintf("%s/%s:generateContent?key=%s", g.baseURL, g.model, g.apiKey)

	data, _ := json.Marshal(reqBody)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrProviderDown, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	// Error handling
	switch resp.StatusCode {
	case 429:
		return nil, ErrRateLimited
	case 500, 502, 503:
		return nil, ErrProviderDown
	case 400:
		return nil, fmt.Errorf("%w: %s", ErrBadRequest, string(body))
	}

	var geminiResp geminiResponse
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return nil, fmt.Errorf("gemini: decode response: %w", err)
	}

	// Safety filter check
	if geminiResp.PromptFeedback != nil && geminiResp.PromptFeedback.BlockReason != "" {
		return nil, fmt.Errorf("%w: content blocked by Gemini safety filter (%s)",
			ErrBadRequest, geminiResp.PromptFeedback.BlockReason)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("gemini: empty response")
	}

	// Parse response for text or function calls
	respParts := geminiResp.Candidates[0].Content.Parts
	var text string
	var toolCalls []ToolCall

	for _, part := range respParts {
		if part.Text != "" {
			text += part.Text
		}
		if part.FunctionCall != nil {
			toolCalls = append(toolCalls, ToolCall{
				ID:    part.FunctionCall.Name, // Use function name as ID
				Name:  part.FunctionCall.Name,
				Input: part.FunctionCall.Args,
			})
		}
	}

	// Return based on what was called
	if len(toolCalls) > 0 {
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

// mapToGeminiContents converts domain.Message slice to Gemini contents.
// Maps "assistant" role → "model" (Gemini's term).
func mapToGeminiContents(messages []domain.Message) []geminiContent {
	contents := make([]geminiContent, 0, len(messages))
	for _, m := range messages {
		role := m.Role
		if role == "assistant" {
			role = "model"
		}
		if role == "system" {
			continue // system messages handled via system_instruction
		}
		contents = append(contents, geminiContent{
			Role:  role,
			Parts: []geminiPart{{Text: m.Content}},
		})
	}
	return contents
}

// cleanGeminiSchema recursively strips OpenAPI/JSONSchema fields unsupported by Gemini API,
// such as "additionalProperties".
func cleanGeminiSchema(schema map[string]interface{}) map[string]interface{} {
	if schema == nil {
		return nil
	}
	cleaned := make(map[string]interface{})
	for k, v := range schema {
		if k == "additionalProperties" {
			continue
		}
		if subMap, ok := v.(map[string]interface{}); ok {
			cleaned[k] = cleanGeminiSchema(subMap)
		} else if slice, ok := v.([]interface{}); ok {
			cleanedSlice := make([]interface{}, len(slice))
			for i, item := range slice {
				if itemMap, ok := item.(map[string]interface{}); ok {
					cleanedSlice[i] = cleanGeminiSchema(itemMap)
				} else {
					cleanedSlice[i] = item
				}
			}
			cleaned[k] = cleanedSlice
		} else {
			cleaned[k] = v
		}
	}
	return cleaned
}
