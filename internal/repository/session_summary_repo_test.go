package repository

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"bruce/internal/domain"
)

func setupSummaryTestDB(t *testing.T) (*sql.DB, SessionRepository, MessageRepository, SessionSummaryRepository) {
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

	return db, NewSessionRepository(db), NewMessageRepository(db), NewSessionSummaryRepository(db)
}

func TestSessionSummaryRepo_CRUD(t *testing.T) {
	db, sessionRepo, _, summaryRepo := setupSummaryTestDB(t)
	defer db.Close()

	session, err := sessionRepo.Create("web", "chan-1", "Test Session")
	require.NoError(t, err)

	// 1. Get before insert -> returns nil, nil
	s, err := summaryRepo.Get(session.ID)
	assert.NoError(t, err)
	assert.Nil(t, s)

	// 2. Upsert initial summary
	initial := &domain.SessionSummary{
		SessionID:           session.ID,
		Summary:             "User prefers Go. Working on Bruce memory system.",
		LastSummarizedMsgID: "msg-1",
		MessageCount:        10,
	}
	err = summaryRepo.Upsert(initial)
	assert.NoError(t, err)

	// 3. Get after insert
	got, err := summaryRepo.Get(session.ID)
	assert.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, session.ID, got.SessionID)
	assert.Equal(t, initial.Summary, got.Summary)
	assert.Equal(t, "msg-1", got.LastSummarizedMsgID)
	assert.Equal(t, 10, got.MessageCount)

	// 4. Upsert update
	updated := &domain.SessionSummary{
		SessionID:           session.ID,
		Summary:             "User prefers Go. Finished memory system.",
		LastSummarizedMsgID: "msg-2",
		MessageCount:        20,
	}
	err = summaryRepo.Upsert(updated)
	assert.NoError(t, err)

	gotUpdated, err := summaryRepo.Get(session.ID)
	assert.NoError(t, err)
	require.NotNil(t, gotUpdated)
	assert.Equal(t, "User prefers Go. Finished memory system.", gotUpdated.Summary)
	assert.Equal(t, 20, gotUpdated.MessageCount)

	// 5. Delete
	err = summaryRepo.Delete(session.ID)
	assert.NoError(t, err)
	gotDeleted, err := summaryRepo.Get(session.ID)
	assert.NoError(t, err)
	assert.Nil(t, gotDeleted)
}

func TestMessageRepo_CountAndOlderMessages(t *testing.T) {
	db, sessionRepo, msgRepo, _ := setupSummaryTestDB(t)
	defer db.Close()

	session, err := sessionRepo.Create("web", "chan-2", "Count Test")
	require.NoError(t, err)

	// Insert 5 messages with distinct timestamps
	baseTime := time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC)
	for i := 1; i <= 5; i++ {
		err := msgRepo.Insert(&domain.Message{
			ID:        string(rune('a' - 1 + i)),
			SessionID: session.ID,
			Role:      "user",
			Content:   string(rune('0' + i)),
			Timestamp: baseTime.Add(time.Duration(i) * time.Minute),
		})
		require.NoError(t, err)
	}

	// 1. CountBySession
	count, err := msgRepo.CountBySession(session.ID)
	assert.NoError(t, err)
	assert.Equal(t, 5, count)

	// 2. GetLastMessage
	last, err := msgRepo.GetLastMessage(session.ID)
	assert.NoError(t, err)
	require.NotNil(t, last)
	assert.Equal(t, "5", last.Content)

	// 3. GetOlderMessages keeping 2 recent
	// Should return first 3 messages: "1", "2", "3"
	older, err := msgRepo.GetOlderMessages(session.ID, 2)
	assert.NoError(t, err)
	require.Len(t, older, 3)
	assert.Equal(t, "1", older[0].Content)
	assert.Equal(t, "2", older[1].Content)
	assert.Equal(t, "3", older[2].Content)

	// 4. GetOlderMessages keeping 5 recent -> 0 older messages
	none, err := msgRepo.GetOlderMessages(session.ID, 5)
	assert.NoError(t, err)
	assert.Empty(t, none)
}
