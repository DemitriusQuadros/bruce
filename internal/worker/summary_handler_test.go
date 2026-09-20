package worker

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"bruce/internal/ai"
	"bruce/internal/config"
	"bruce/internal/domain"
	"bruce/internal/repository"
)

type stubSummaryLLM struct {
	response string
}

func (s *stubSummaryLLM) GenerateResponse(ctx context.Context, systemPrompt string, history []domain.Message) (string, error) {
	return s.response, nil
}

func (s *stubSummaryLLM) GenerateWithTools(ctx context.Context, systemPrompt string, history []domain.Message, tools []ai.ToolDefinition) (*ai.ToolCallResponse, error) {
	return &ai.ToolCallResponse{Text: s.response, Complete: true}, nil
}

func setupSummaryWorkerTestDB(t *testing.T) (*sql.DB, repository.SessionRepository, repository.MessageRepository, repository.SessionSummaryRepository) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			connector_type TEXT NOT NULL,
			channel_id TEXT NOT NULL,
			title TEXT NOT NULL DEFAULT '',
			system_prompt TEXT NOT NULL DEFAULT '',
			is_active INTEGER NOT NULL DEFAULT 1,
			provider_override TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT (datetime('now')),
			updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
			UNIQUE(connector_type, channel_id)
		);
		CREATE TABLE IF NOT EXISTS messages (
			id TEXT PRIMARY KEY,
			session_id TEXT NOT NULL,
			role TEXT NOT NULL,
			content TEXT NOT NULL,
			timestamp DATETIME NOT NULL DEFAULT (datetime('now')),
			FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
		);
		CREATE TABLE IF NOT EXISTS session_summaries (
			session_id TEXT PRIMARY KEY,
			summary TEXT NOT NULL DEFAULT '',
			last_summarized_msg_id TEXT NOT NULL DEFAULT '',
			message_count INTEGER NOT NULL DEFAULT 0,
			updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
			FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
		);
	`)
	require.NoError(t, err)

	return db, repository.NewSessionRepository(db), repository.NewMessageRepository(db), repository.NewSessionSummaryRepository(db)
}

func TestHandleSummarizeSessionTask_Success(t *testing.T) {
	db, sessionRepo, msgRepo, summaryRepo := setupSummaryWorkerTestDB(t)
	defer db.Close()

	session, err := sessionRepo.Create("web", "chan-sum-1", "Summarize Test")
	require.NoError(t, err)

	// Insert 15 messages so it exceeds the keepRecent (10) threshold
	baseTime := time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC)
	for i := 1; i <= 15; i++ {
		err := msgRepo.Insert(&domain.Message{
			ID:        string(rune('a' - 1 + i)),
			SessionID: session.ID,
			Role:      "user",
			Content:   "Important project decision " + string(rune('0'+i)),
			Timestamp: baseTime.Add(time.Duration(i) * time.Minute),
		})
		require.NoError(t, err)
	}

	stubLLM := &stubSummaryLLM{
		response: "• User made key decisions 1 through 5.\n• Architecture is set up with SQLite and Asynq.",
	}

	cfg := &config.Config{
		Claude: config.ClaudeConfig{ContextWindow: 10},
	}

	proc := NewProcessor(sessionRepo, msgRepo, nil, stubLLM, nil, cfg)
	proc.SetSummaryRepo(summaryRepo)

	task, err := NewSummarizeSessionTask(SummarizeSessionPayload{SessionID: session.ID})
	require.NoError(t, err)

	err = proc.HandleSummarizeSessionTask(context.Background(), task)
	assert.NoError(t, err)

	// Verify summary was saved
	summary, err := summaryRepo.Get(session.ID)
	assert.NoError(t, err)
	require.NotNil(t, summary)
	assert.Equal(t, session.ID, summary.SessionID)
	assert.Contains(t, summary.Summary, "User made key decisions 1 through 5")
	assert.Equal(t, 15, summary.MessageCount)
	assert.Equal(t, "j", summary.LastSummarizedMsgID) // 15 - 5 = 10th message ('j')
}
