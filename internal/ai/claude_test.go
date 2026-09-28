package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"bruce/internal/config"
	"bruce/internal/domain"
)

func newTestClaudeProvider(serverURL string) *claudeProvider {
	cfg := config.ClaudeConfig{
		APIKey:    "test-key",
		Model:     "claude-3-5-haiku-20241022",
		MaxTokens: 1024,
	}
	p := NewClaudeProvider(cfg).(*claudeProvider)
	p.baseURL = serverURL
	return p
}

func TestClaudeProvider_MultiToolResultsPreservedInRequest(t *testing.T) {
	var capturedReq anthropicRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "test-key", r.Header.Get("x-api-key"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "/v1/messages", r.URL.Path)

		err := json.NewDecoder(r.Body).Decode(&capturedReq)
		require.NoError(t, err)

		resp := anthropicResponse{
			Content: []struct {
				Type  string      `json:"type"`
				Text  string      `json:"text,omitempty"`
				ID    string      `json:"id,omitempty"`
				Name  string      `json:"name,omitempty"`
				Input interface{} `json:"input,omitempty"`
			}{
				{Type: "text", Text: "I found 3 emails and summarized them."},
			},
			StopReason: "end_turn",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := newTestClaudeProvider(server.URL)

	messages := []domain.Message{
		{Role: "user", Content: "Check my email"},
		{
			Role: "assistant",
			Type: "tool_call",
			ToolCalls: []domain.ToolCall{
				{ID: "toolu_01", Name: "email_search", Input: map[string]interface{}{"q": "a"}},
				{ID: "toolu_02", Name: "email_search", Input: map[string]interface{}{"q": "b"}},
				{ID: "toolu_03", Name: "email_search", Input: map[string]interface{}{"q": "c"}},
			},
		},
		{
			Role: "user",
			Type: "tool_result",
			ToolResult: &domain.ToolResult{
				ID:      "toolu_01",
				Content: "email 1",
			},
		},
		{
			Role: "user",
			Type: "tool_result",
			ToolResult: &domain.ToolResult{
				ID:      "toolu_02",
				Content: "email 2",
			},
		},
		{
			Role: "user",
			Type: "tool_result",
			ToolResult: &domain.ToolResult{
				ID:      "toolu_03",
				Content: "email 3",
			},
		},
	}

	toolDefs := []ToolDefinition{
		{
			Name:        "email_search",
			Description: "Search email",
			InputSchema: map[string]interface{}{"type": "object"},
		},
	}

	resp, err := p.GenerateWithTools(context.Background(), "System prompt", messages, toolDefs)
	require.NoError(t, err)
	assert.True(t, resp.Complete)
	assert.Equal(t, "I found 3 emails and summarized them.", resp.Text)

	// Verify the Anthropic request structure:
	// messages[0]: user message ("Check my email")
	// messages[1]: assistant message with 3 tool_use blocks
	// messages[2]: user message with ALL 3 tool_result blocks merged into a single user turn!
	require.Len(t, capturedReq.Messages, 3)

	assert.Equal(t, "user", capturedReq.Messages[0].Role)
	assert.Equal(t, "assistant", capturedReq.Messages[1].Role)
	assert.Equal(t, "user", capturedReq.Messages[2].Role)

	// Inspect content blocks of message 2 (tool results)
	contentBytes, err := json.Marshal(capturedReq.Messages[2].Content)
	require.NoError(t, err)

	var toolResults []anthropicContent
	require.NoError(t, json.Unmarshal(contentBytes, &toolResults))

	require.Len(t, toolResults, 3, "All 3 tool results must be present in the user turn")
	assert.Equal(t, "tool_result", toolResults[0].Type)
	assert.Equal(t, "toolu_01", toolResults[0].ToolUseID)
	assert.Equal(t, "email 1", toolResults[0].Content)

	assert.Equal(t, "tool_result", toolResults[1].Type)
	assert.Equal(t, "toolu_02", toolResults[1].ToolUseID)
	assert.Equal(t, "email 2", toolResults[1].Content)

	assert.Equal(t, "tool_result", toolResults[2].Type)
	assert.Equal(t, "toolu_03", toolResults[2].ToolUseID)
	assert.Equal(t, "email 3", toolResults[2].Content)
}
