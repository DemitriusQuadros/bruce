package ai

import (
	"context"
	"errors"
	"testing"

	"bruce/internal/domain"
	"bruce/internal/monitoring"
)

// mockLLM implements LLMService for testing
type mockLLM struct {
	responses []*ToolCallResponse
	callCount int
}

func (m *mockLLM) GenerateResponse(ctx context.Context, systemPrompt string, history []domain.Message) (string, error) {
	return "", errors.New("not implemented")
}

func (m *mockLLM) GenerateWithTools(ctx context.Context, systemPrompt string, messages []domain.Message, tools []ToolDefinition) (*ToolCallResponse, error) {
	if m.callCount >= len(m.responses) {
		return nil, errors.New("no more responses")
	}
	resp := m.responses[m.callCount]
	m.callCount++
	return resp, nil
}

// mockRegistry implements ToolRegistry for testing
type mockRegistry struct {
	definitions []ToolDefinition
	execFunc    func(ctx context.Context, name string, input map[string]interface{}) (string, error)
}

func (m *mockRegistry) GetDefinitions() []ToolDefinition {
	return m.definitions
}

func (m *mockRegistry) Execute(ctx context.Context, name string, input map[string]interface{}) (string, error) {
	if m.execFunc != nil {
		return m.execFunc(ctx, name, input)
	}
	return "ok", nil
}

func TestRunAgentLoop(t *testing.T) {
	tests := []struct {
		name        string
		responses   []*ToolCallResponse
		execFunc    func(context.Context, string, map[string]interface{}) (string, error)
		maxRetries  int
		wantText    string
		wantErr     error
		wantErrType error
	}{
		{
			name: "happy path: no tools, immediate text",
			responses: []*ToolCallResponse{
				{
					Text:     "Hello, world!",
					Complete: true,
				},
			},
			maxRetries: 5,
			wantText:   "Hello, world!",
			wantErr:    nil,
		},
		{
			name: "one tool call with result",
			responses: []*ToolCallResponse{
				{
					ToolCalls: []ToolCall{
						{
							ID:    "call_1",
							Name:  "get_weather",
							Input: map[string]interface{}{"location": "SF"},
						},
					},
					Complete: false,
				},
				{
					Text:     "It's 72°F in San Francisco",
					Complete: true,
				},
			},
			execFunc: func(ctx context.Context, name string, input map[string]interface{}) (string, error) {
				return "72°F", nil
			},
			maxRetries: 5,
			wantText:   "It's 72°F in San Francisco",
			wantErr:    nil,
		},
		{
			name: "tool execution error",
			responses: []*ToolCallResponse{
				{
					ToolCalls: []ToolCall{
						{
							ID:    "call_1",
							Name:  "failing_tool",
							Input: map[string]interface{}{},
						},
					},
					Complete: false,
				},
				{
					Text:     "I couldn't fetch that information",
					Complete: true,
				},
			},
			execFunc: func(ctx context.Context, name string, input map[string]interface{}) (string, error) {
				return "", errors.New("tool failed")
			},
			maxRetries: 5,
			wantText:   "I couldn't fetch that information",
			wantErr:    nil,
		},
		{
			name: "max retries exceeded",
			responses: []*ToolCallResponse{
				{
					ToolCalls: []ToolCall{
						{
							ID:    "call_1",
							Name:  "tool1",
							Input: map[string]interface{}{},
						},
					},
					Complete: false,
				},
				{
					ToolCalls: []ToolCall{
						{
							ID:    "call_2",
							Name:  "tool2",
							Input: map[string]interface{}{},
						},
					},
					Complete: false,
				},
				{
					ToolCalls: []ToolCall{
						{
							ID:    "call_3",
							Name:  "tool3",
							Input: map[string]interface{}{},
						},
					},
					Complete: false,
				},
			},
			execFunc: func(ctx context.Context, name string, input map[string]interface{}) (string, error) {
				return "result", nil
			},
			maxRetries:  2,
			wantText:    "",
			wantErrType: ErrMaxRetriesExceeded,
		},
		{
			name: "multiple tool calls in one turn",
			responses: []*ToolCallResponse{
				{
					ToolCalls: []ToolCall{
						{
							ID:    "call_1",
							Name:  "get_temp",
							Input: map[string]interface{}{},
						},
						{
							ID:    "call_2",
							Name:  "get_humidity",
							Input: map[string]interface{}{},
						},
					},
					Complete: false,
				},
				{
					Text:     "It's 72°F and 60% humid",
					Complete: true,
				},
			},
			execFunc: func(ctx context.Context, name string, input map[string]interface{}) (string, error) {
				if name == "get_temp" {
					return "72°F", nil
				}
				return "60%", nil
			},
			maxRetries: 5,
			wantText:   "It's 72°F and 60% humid",
			wantErr:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			llm := &mockLLM{responses: tt.responses}
			registry := &mockRegistry{
				definitions: []ToolDefinition{
					{Name: "get_weather", Description: "Get weather"},
					{Name: "get_temp", Description: "Get temperature"},
					{Name: "get_humidity", Description: "Get humidity"},
					{Name: "failing_tool", Description: "A tool that fails"},
				},
				execFunc: tt.execFunc,
			}

			result, err := RunAgentLoop(
				context.Background(),
				llm,
				registry,
				"You are a helpful assistant",
				[]domain.Message{},
				tt.maxRetries,
				nil, // logger
			)

			if tt.wantErrType != nil {
				if !errors.Is(err, tt.wantErrType) {
					t.Errorf("expected error %v, got %v", tt.wantErrType, err)
				}
			} else if err != nil && tt.wantErr == nil {
				t.Errorf("unexpected error: %v", err)
			}

			if result != tt.wantText {
				t.Errorf("expected text %q, got %q", tt.wantText, result)
			}
		})
	}
}

func TestRunAgentLoopLogging(t *testing.T) {
	// Test that logging is called correctly
	llm := &mockLLM{
		responses: []*ToolCallResponse{
			{
				ToolCalls: []ToolCall{
					{
						ID:    "call_1",
						Name:  "test_tool",
						Input: map[string]interface{}{"key": "value"},
					},
				},
				Complete: false,
			},
			{
				Text:     "Done",
				Complete: true,
			},
		},
	}

	registry := &mockRegistry{
		definitions: []ToolDefinition{
			{Name: "test_tool", Description: "Test tool"},
		},
		execFunc: func(ctx context.Context, name string, input map[string]interface{}) (string, error) {
			return "test result", nil
		},
	}

	logger := monitoring.NewStructuredLogger(nil)

	result, err := RunAgentLoop(
		context.Background(),
		llm,
		registry,
		"System prompt",
		[]domain.Message{},
		5,
		logger,
	)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != "Done" {
		t.Errorf("expected 'Done', got %q", result)
	}
}
