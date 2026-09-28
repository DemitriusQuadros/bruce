package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"bruce/internal/domain"
)

func TestSanitizeHistory_PlainTextAlternating(t *testing.T) {
	history := []domain.Message{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there!"},
		{Role: "user", Content: "How are you?"},
	}

	result := sanitizeHistory(history)
	assert.Len(t, result, 3)
	assert.Equal(t, "user", result[0].Role)
	assert.Equal(t, "Hello", result[0].Content)
	assert.Equal(t, "assistant", result[1].Role)
	assert.Equal(t, "Hi there!", result[1].Content)
	assert.Equal(t, "user", result[2].Role)
	assert.Equal(t, "How are you?", result[2].Content)
}

func TestSanitizeHistory_MergeConsecutivePlainText(t *testing.T) {
	history := []domain.Message{
		{Role: "user", Content: "Hello"},
		{Role: "user", Content: "World"},
		{Role: "assistant", Content: "Response 1"},
		{Role: "assistant", Content: "Response 2"},
	}

	result := sanitizeHistory(history)
	assert.Len(t, result, 2)
	assert.Equal(t, "user", result[0].Role)
	assert.Equal(t, "Hello\nWorld", result[0].Content)
	assert.Equal(t, "assistant", result[1].Role)
	assert.Equal(t, "Response 1\nResponse 2", result[1].Content)
}

func TestSanitizeHistory_FiltersSystemMessages(t *testing.T) {
	history := []domain.Message{
		{Role: "system", Content: "System prompt"},
		{Role: "user", Content: "Hello"},
		{Role: "system", Content: "Another system message"},
		{Role: "assistant", Content: "Hi"},
	}

	result := sanitizeHistory(history)
	assert.Len(t, result, 2)
	assert.Equal(t, "user", result[0].Role)
	assert.Equal(t, "Hello", result[0].Content)
	assert.Equal(t, "assistant", result[1].Role)
	assert.Equal(t, "Hi", result[1].Content)
}

func TestSanitizeHistory_PreservesMultiToolResults(t *testing.T) {
	// Simulates an agent loop turn with 3 parallel tool executions
	history := []domain.Message{
		{Role: "user", Content: "Check my email for jobs"},
		{
			Role: "assistant",
			Type: "tool_call",
			ToolCalls: []domain.ToolCall{
				{ID: "toolu_1", Name: "email_search", Input: map[string]interface{}{"query": "job"}},
				{ID: "toolu_2", Name: "email_search", Input: map[string]interface{}{"query": "linkedin"}},
				{ID: "toolu_3", Name: "email_search", Input: map[string]interface{}{"query": "offer"}},
			},
		},
		{
			Role: "user",
			Type: "tool_result",
			ToolResult: &domain.ToolResult{
				ID:      "toolu_1",
				Content: "found 10 emails",
			},
		},
		{
			Role: "user",
			Type: "tool_result",
			ToolResult: &domain.ToolResult{
				ID:      "toolu_2",
				Content: "found 5 emails",
			},
		},
		{
			Role: "user",
			Type: "tool_result",
			ToolResult: &domain.ToolResult{
				ID:      "toolu_3",
				Content: "found 2 emails",
			},
		},
	}

	result := sanitizeHistory(history)
	// All 3 tool results must be preserved as distinct messages
	assert.Len(t, result, 5)
	assert.Equal(t, "user", result[0].Role)
	assert.Equal(t, "Check my email for jobs", result[0].Content)

	assert.Equal(t, "assistant", result[1].Role)
	assert.Equal(t, "tool_call", result[1].Type)
	assert.Len(t, result[1].ToolCalls, 3)

	assert.Equal(t, "user", result[2].Role)
	assert.Equal(t, "tool_result", result[2].Type)
	assert.NotNil(t, result[2].ToolResult)
	assert.Equal(t, "toolu_1", result[2].ToolResult.ID)

	assert.Equal(t, "user", result[3].Role)
	assert.Equal(t, "tool_result", result[3].Type)
	assert.NotNil(t, result[3].ToolResult)
	assert.Equal(t, "toolu_2", result[3].ToolResult.ID)

	assert.Equal(t, "user", result[4].Role)
	assert.Equal(t, "tool_result", result[4].Type)
	assert.NotNil(t, result[4].ToolResult)
	assert.Equal(t, "toolu_3", result[4].ToolResult.ID)
}
