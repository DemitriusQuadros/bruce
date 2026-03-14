package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hibiken/asynq"

	"bruce/internal/ai"
	"bruce/internal/config"
	"bruce/internal/database"
	"bruce/internal/domain"
	"bruce/internal/monitoring"
	"bruce/internal/repository"
)

// --- test helpers ---

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:?_foreign_keys=ON")
	require.NoError(t, err)
	err = database.RunMigrations(db)
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db
}

func testConfig() *config.Config {
	return &config.Config{
		Claude: config.ClaudeConfig{
			Model:         "claude-opus-4-6",
			MaxTokens:     1024,
			ContextWindow: 15,
		},
	}
}

func newTestProcessor(t *testing.T, db *sql.DB, llm ai.LLMService, cfg ...*config.Config) *Processor {
	t.Helper()
	sessRepo := repository.NewSessionRepository(db)
	msgRepo := repository.NewMessageRepository(db)
	cfgRepo := repository.NewConfigRepository(db)
	monitoringRepo := repository.NewMonitoringRepository(db)
	metricsCollector := monitoring.NewCollector(db)
	structuredLogger := monitoring.NewStructuredLogger(db)
	registry := NewDispatcherRegistry()

	config := testConfig()
	if len(cfg) > 0 && cfg[0] != nil {
		config = cfg[0]
	}

	return NewProcessor(sessRepo, msgRepo, cfgRepo, monitoringRepo, llm, registry,
		metricsCollector, structuredLogger, config)
}

func makeTask(t *testing.T, p ProcessIncomingMessagePayload) *asynq.Task {
	t.Helper()
	data, err := json.Marshal(p)
	require.NoError(t, err)
	return asynq.NewTask(TaskProcessIncomingMessage, data)
}

// --- mock LLM ---

type mockLLM struct {
	response       string
	err            error
	capturedPrompt string
	called         bool
}

func (m *mockLLM) GenerateResponse(_ context.Context, systemPrompt string, _ []domain.Message) (string, error) {
	m.called = true
	m.capturedPrompt = systemPrompt
	return m.response, m.err
}

func (m *mockLLM) GenerateWithTools(_ context.Context, systemPrompt string, _ []domain.Message, _ []ai.ToolDefinition) (*ai.ToolCallResponse, error) {
	m.called = true
	m.capturedPrompt = systemPrompt
	if m.err != nil {
		return nil, m.err
	}
	return &ai.ToolCallResponse{Text: m.response, Complete: true}, nil
}

// --- mock dispatcher ---

type mockDispatcher struct {
	sent []string
}

func (d *mockDispatcher) Send(_ string, message string) error {
	d.sent = append(d.sent, message)
	return nil
}

// --- tests ---

func TestProcessor_SkipsInactiveSession(t *testing.T) {
	db := openTestDB(t)
	sessRepo := repository.NewSessionRepository(db)
	msgRepo := repository.NewMessageRepository(db)
	llm := &mockLLM{response: "hello"}

	// Create session and immediately deactivate it.
	sess, err := sessRepo.FindOrCreate("whatsapp", "+5511999990000")
	require.NoError(t, err)
	require.NoError(t, sessRepo.SetActive(sess.ID, false))

	proc := newTestProcessor(t, db, llm)
	task := makeTask(t, ProcessIncomingMessagePayload{
		ConnectorType: "whatsapp",
		ChannelID:     "+5511999990000",
		Content:       "hello",
	})

	err = proc.HandleProcessIncomingMessageTask(context.Background(), task)
	require.NoError(t, err)
	assert.False(t, llm.called, "LLM must not be called for inactive session")

	// No messages should be stored.
	msgs, err := msgRepo.GetRecent(sess.ID, 10)
	require.NoError(t, err)
	assert.Empty(t, msgs)
}

func TestProcessor_InsertsUserAndAssistantMessages(t *testing.T) {
	db := openTestDB(t)
	sessRepo := repository.NewSessionRepository(db)
	msgRepo := repository.NewMessageRepository(db)
	llm := &mockLLM{response: "I am your assistant"}
	disp := &mockDispatcher{}

	// Create processor and register dispatcher
	proc := newTestProcessor(t, db, llm)
	proc.dispatcher.Register("discord", disp)

	task := makeTask(t, ProcessIncomingMessagePayload{
		ConnectorType: "discord",
		ChannelID:     "channel-123",
		Content:       "what's up?",
	})

	err := proc.HandleProcessIncomingMessageTask(context.Background(), task)
	require.NoError(t, err)

	sess, err := sessRepo.FindOrCreate("discord", "channel-123")
	require.NoError(t, err)

	msgs, err := msgRepo.GetRecent(sess.ID, 10)
	require.NoError(t, err)
	require.Len(t, msgs, 2)

	// GetRecent returns newest first; assistant is inserted after user.
	roles := []string{msgs[0].Role, msgs[1].Role}
	assert.Contains(t, roles, "assistant")
	assert.Contains(t, roles, "user")

	// Find each by role for content assertions.
	byRole := map[string]string{}
	for _, m := range msgs {
		byRole[m.Role] = m.Content
	}
	assert.Equal(t, "what's up?", byRole["user"])
	assert.Equal(t, "I am your assistant", byRole["assistant"])

	// Dispatcher received the reply.
	assert.Equal(t, []string{"I am your assistant"}, disp.sent)
}

func TestProcessor_UsesSessionSystemPromptOverDefault(t *testing.T) {
	db := openTestDB(t)
	sessRepo := repository.NewSessionRepository(db)
	cfgRepo := repository.NewConfigRepository(db)

	// Store a default system prompt in config.
	require.NoError(t, cfgRepo.Upsert("ui.default_system_prompt", "default prompt"))

	// Create session and set a custom system prompt.
	sess, err := sessRepo.FindOrCreate("whatsapp", "+1234567890")
	require.NoError(t, err)
	require.NoError(t, sessRepo.UpdateSystemPrompt(sess.ID, "session-specific prompt"))

	llm := &mockLLM{response: "ok"}
	proc := newTestProcessor(t, db, llm)

	task := makeTask(t, ProcessIncomingMessagePayload{
		ConnectorType: "whatsapp",
		ChannelID:     "+1234567890",
		Content:       "hi",
	})
	require.NoError(t, proc.HandleProcessIncomingMessageTask(context.Background(), task))

	assert.Equal(t, "session-specific prompt", llm.capturedPrompt)
}

func TestProcessor_UsesDBSystemPromptWhenNoSessionPrompt(t *testing.T) {
	db := openTestDB(t)
	cfgRepo := repository.NewConfigRepository(db)

	// Store a default system prompt in config database only.
	require.NoError(t, cfgRepo.Upsert("ui.default_system_prompt", "database default prompt"))

	llm := &mockLLM{response: "ok"}
	proc := newTestProcessor(t, db, llm)

	task := makeTask(t, ProcessIncomingMessagePayload{
		ConnectorType: "discord",
		ChannelID:     "channel-456",
		Content:       "hello",
	})
	require.NoError(t, proc.HandleProcessIncomingMessageTask(context.Background(), task))

	assert.Equal(t, "database default prompt", llm.capturedPrompt)
}

func TestProcessor_UsesYAMLSystemPromptWhenNotInDB(t *testing.T) {
	db := openTestDB(t)

	// Do NOT store a system prompt in config database.
	// The processor should fall back to the YAML config.
	llm := &mockLLM{response: "ok"}
	cfg := testConfig()
	cfg.UI.DefaultSystemPrompt = "yaml default prompt"
	proc := newTestProcessor(t, db, llm, cfg)

	task := makeTask(t, ProcessIncomingMessagePayload{
		ConnectorType: "discord",
		ChannelID:     "channel-789",
		Content:       "hello",
	})
	require.NoError(t, proc.HandleProcessIncomingMessageTask(context.Background(), task))

	assert.Equal(t, "yaml default prompt", llm.capturedPrompt)
}

func TestProcessor_RateLimitedErrorReturnsForRetry(t *testing.T) {
	db := openTestDB(t)
	llm := &mockLLM{err: ai.ErrRateLimited}

	proc := newTestProcessor(t, db, llm)
	task := makeTask(t, ProcessIncomingMessagePayload{
		ConnectorType: "discord",
		ChannelID:     "chan-xyz",
		Content:       "ping",
	})

	err := proc.HandleProcessIncomingMessageTask(context.Background(), task)
	require.Error(t, err)
	assert.ErrorIs(t, err, ai.ErrRateLimited, "rate limit error must propagate so Asynq retries")
}
