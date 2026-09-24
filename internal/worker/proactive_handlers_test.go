package worker

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sync"
	"testing"

	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"bruce/internal/ai"
	"bruce/internal/config"
	"bruce/internal/domain"
)

type mockProactiveDispatcher struct {
	mu       sync.Mutex
	messages []string
	channels []string
}

func (m *mockProactiveDispatcher) Send(channelID string, message string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.channels = append(m.channels, channelID)
	m.messages = append(m.messages, message)
	return nil
}

type mockProactiveLLM struct {
	response string
	err      error
}

func (m *mockProactiveLLM) GenerateResponse(ctx context.Context, systemPrompt string, history []domain.Message) (string, error) {
	return m.response, m.err
}

func (m *mockProactiveLLM) GenerateWithTools(ctx context.Context, systemPrompt string, messages []domain.Message, tools []ai.ToolDefinition) (*ai.ToolCallResponse, error) {
	return &ai.ToolCallResponse{Text: m.response, Complete: true}, m.err
}

type mockProactiveToolRegistry struct {
	output string
}

func (m *mockProactiveToolRegistry) GetDefinitions() []ai.ToolDefinition {
	return nil
}

func (m *mockProactiveToolRegistry) Execute(ctx context.Context, name string, input map[string]interface{}) (string, error) {
	return m.output, nil
}

func TestHandleEvaluateWatchTask_MatchAndDeduplication(t *testing.T) {
	disp := &mockProactiveDispatcher{}
	dispRegistry := NewDispatcherRegistry()
	dispRegistry.Register("discord", disp)

	llm := &mockProactiveLLM{
		response: "MATCH: Urgent email from contractor regarding invoice #1024",
	}
	toolReg := &mockProactiveToolRegistry{
		output: "Subject: Urgent invoice #1024 from contractor@example.com",
	}

	proc := NewProcessor(nil, nil, nil, llm, dispRegistry, &config.Config{})
	proc.SetToolRegistry(toolReg)

	// 1. First run - should match and send alert
	payload1 := EvaluateWatchPayload{
		TaskID:          "watch-1",
		TargetConnector: "discord",
		TargetChannelID: "chan-123",
		Title:           "Contractor Watch",
		Condition:       "Urgent contractor emails",
		TargetTools:     []string{"gmail_search"},
		LastResultHash:  "",
	}
	b1, err := json.Marshal(payload1)
	require.NoError(t, err)

	task1 := asynq.NewTask(TaskEvaluateWatch, b1)
	err = proc.HandleEvaluateWatchTask(context.Background(), task1)
	require.NoError(t, err)

	require.Len(t, disp.messages, 1)
	assert.Contains(t, disp.messages[0], "Urgent email from contractor")

	// 2. Second run with same content and matching hash - should be deduplicated
	hasher := sha256.New()
	hasher.Write([]byte("Urgent email from contractor regarding invoice #1024"))
	lastHash := hex.EncodeToString(hasher.Sum(nil))

	payload2 := EvaluateWatchPayload{
		TaskID:          "watch-1",
		TargetConnector: "discord",
		TargetChannelID: "chan-123",
		Title:           "Contractor Watch",
		Condition:       "Urgent contractor emails",
		TargetTools:     []string{"gmail_search"},
		LastResultHash:  lastHash,
	}
	b2, err := json.Marshal(payload2)
	require.NoError(t, err)

	task2 := asynq.NewTask(TaskEvaluateWatch, b2)
	err = proc.HandleEvaluateWatchTask(context.Background(), task2)
	require.NoError(t, err)

	// Length should still be 1 (no new message sent)
	assert.Len(t, disp.messages, 1)
}

func TestHandleExecuteScheduledReportTask(t *testing.T) {
	disp := &mockProactiveDispatcher{}
	dispRegistry := NewDispatcherRegistry()
	dispRegistry.Register("whatsapp", disp)

	llm := &mockProactiveLLM{
		response: "Here is your morning briefing:\n- 10:00 AM Team Standup\n- 3 open PRs to review",
	}

	proc := NewProcessor(nil, nil, nil, llm, dispRegistry, &config.Config{})

	payload := ExecuteScheduledReportPayload{
		TaskID:          "report-1",
		TargetConnector: "whatsapp",
		TargetChannelID: "5511999999999",
		Title:           "Daily Morning Briefing",
		Prompt:          "Summarize calendar and PRs",
	}
	b, err := json.Marshal(payload)
	require.NoError(t, err)

	task := asynq.NewTask(TaskExecuteScheduledReport, b)
	err = proc.HandleExecuteScheduledReportTask(context.Background(), task)
	require.NoError(t, err)

	require.Len(t, disp.messages, 1)
	assert.Equal(t, "5511999999999", disp.channels[0])
	assert.Contains(t, disp.messages[0], "Daily Morning Briefing")
	assert.Contains(t, disp.messages[0], "Team Standup")
}
