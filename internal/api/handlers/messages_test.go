package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"bruce/internal/domain"
	"bruce/internal/repository"
)

// TestMessagesHandlerGet tests the GET /api/v1/sessions/{id}/messages endpoint.
func TestMessagesHandlerGet(t *testing.T) {
	db := setupMessageTestDB(t)
	defer db.Close()

	sessionRepo := repository.NewSessionRepository(db)
	messageRepo := repository.NewMessageRepository(db)

	// Create a session.
	s, err := sessionRepo.FindOrCreate("whatsapp", "123")
	require.NoError(t, err)

	// Insert test messages.
	now := time.Now()
	msg1 := &domain.Message{
		ID:        uuid.New().String(),
		SessionID: s.ID,
		Role:      "user",
		Content:   "Hello",
		Timestamp: now.Add(-2 * time.Second),
	}
	err = messageRepo.Insert(msg1)
	require.NoError(t, err)

	msg2 := &domain.Message{
		ID:        uuid.New().String(),
		SessionID: s.ID,
		Role:      "assistant",
		Content:   "Hi there!",
		Timestamp: now.Add(-1 * time.Second),
	}
	err = messageRepo.Insert(msg2)
	require.NoError(t, err)

	handler := MessagesHandler()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/sessions/"+s.ID+"/messages", nil)
	req = mux.SetURLVars(req, map[string]string{"id": s.ID})
	req = req.WithContext(context.WithValue(
		context.WithValue(req.Context(), "messageRepo", messageRepo),
		"sessionRepo", sessionRepo,
	))

	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result []messageResponse
	err = json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)

	// Should return 2 messages in reverse order (newest first).
	assert.Equal(t, 2, len(result))
	assert.Equal(t, "assistant", result[0].Role)
	assert.Equal(t, "user", result[1].Role)
}

// TestMessagesHandlerGetNotFound tests 404 when session doesn't exist.
func TestMessagesHandlerGetNotFound(t *testing.T) {
	db := setupMessageTestDB(t)
	defer db.Close()

	sessionRepo := repository.NewSessionRepository(db)
	messageRepo := repository.NewMessageRepository(db)

	handler := MessagesHandler()
	fakeID := uuid.New().String()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/sessions/"+fakeID+"/messages", nil)
	req = mux.SetURLVars(req, map[string]string{"id": fakeID})
	req = req.WithContext(context.WithValue(
		context.WithValue(req.Context(), "messageRepo", messageRepo),
		"sessionRepo", sessionRepo,
	))

	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestMessagesHandlerGetWithLimit tests the limit parameter.
func TestMessagesHandlerGetWithLimit(t *testing.T) {
	db := setupMessageTestDB(t)
	defer db.Close()

	sessionRepo := repository.NewSessionRepository(db)
	messageRepo := repository.NewMessageRepository(db)

	s, err := sessionRepo.FindOrCreate("whatsapp", "456")
	require.NoError(t, err)

	// Insert 5 messages.
	now := time.Now()
	for i := 0; i < 5; i++ {
		msg := &domain.Message{
			ID:        uuid.New().String(),
			SessionID: s.ID,
			Role:      "user",
			Content:   "Message " + string(rune('0'+i)),
			Timestamp: now.Add(time.Duration(i-5) * time.Second),
		}
		err = messageRepo.Insert(msg)
		require.NoError(t, err)
	}

	handler := MessagesHandler()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/sessions/"+s.ID+"/messages?limit=3", nil)
	req = mux.SetURLVars(req, map[string]string{"id": s.ID})
	req = req.WithContext(context.WithValue(
		context.WithValue(req.Context(), "messageRepo", messageRepo),
		"sessionRepo", sessionRepo,
	))

	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result []messageResponse
	err = json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)

	// Should respect limit of 3.
	assert.Equal(t, 3, len(result))
}

// TestMessagesHandlerGetWithMaxLimit tests that limit caps at 200.
func TestMessagesHandlerGetWithMaxLimit(t *testing.T) {
	db := setupMessageTestDB(t)
	defer db.Close()

	sessionRepo := repository.NewSessionRepository(db)
	messageRepo := repository.NewMessageRepository(db)

	s, err := sessionRepo.FindOrCreate("whatsapp", "789")
	require.NoError(t, err)

	handler := MessagesHandler()
	// Request with limit > 200 (should be capped at 200).
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/sessions/"+s.ID+"/messages?limit=500", nil)
	req = mux.SetURLVars(req, map[string]string{"id": s.ID})
	req = req.WithContext(context.WithValue(
		context.WithValue(req.Context(), "messageRepo", messageRepo),
		"sessionRepo", sessionRepo,
	))

	w := httptest.NewRecorder()
	handler(w, req)

	// Should succeed (empty result since no messages, but valid response).
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestMessagesHandlerGetEmptySession tests retrieving messages from an empty session.
func TestMessagesHandlerGetEmptySession(t *testing.T) {
	db := setupMessageTestDB(t)
	defer db.Close()

	sessionRepo := repository.NewSessionRepository(db)
	messageRepo := repository.NewMessageRepository(db)

	s, err := sessionRepo.FindOrCreate("whatsapp", "999")
	require.NoError(t, err)

	handler := MessagesHandler()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/sessions/"+s.ID+"/messages", nil)
	req = mux.SetURLVars(req, map[string]string{"id": s.ID})
	req = req.WithContext(context.WithValue(
		context.WithValue(req.Context(), "messageRepo", messageRepo),
		"sessionRepo", sessionRepo,
	))

	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result []messageResponse
	err = json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, 0, len(result))
}

// setupMessageTestDB creates an in-memory SQLite database for message tests.
func setupMessageTestDB(t *testing.T) *sql.DB {
	db := setupTestDB(t)

	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS messages (
			id TEXT PRIMARY KEY,
			session_id TEXT NOT NULL,
			role TEXT NOT NULL,
			content TEXT,
			timestamp DATETIME NOT NULL,
			FOREIGN KEY (session_id) REFERENCES sessions(id)
		)
	`)
	require.NoError(t, err)

	return db
}
