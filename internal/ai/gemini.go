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
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"` // "user" | "model"
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiGenConfig struct {
	MaxOutputTokens int     `json:"maxOutputTokens"`
	Temperature     float32 `json:"temperature,omitempty"`
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
