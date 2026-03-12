package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"bruce/internal/repository"
)

// TestSessionsHandlerList tests the GET /api/v1/sessions endpoint.
func TestSessionsHandlerList(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	sessionRepo := repository.NewSessionRepository(db)

	// Insert test sessions.
	s1, err := sessionRepo.FindOrCreate("whatsapp", "123456")
	require.NoError(t, err)

	s2, err := sessionRepo.FindOrCreate("discord", "user789")
	require.NoError(t, err)

	handler := SessionsHandler()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/sessions", nil)
	req = req.WithContext(context.WithValue(req.Context(), "sessionRepo", sessionRepo))

	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result []sessionResponse
	err = json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, 2, len(result))
	assert.Equal(t, s1.ID, result[0].ID)
	assert.Equal(t, "whatsapp", result[0].ConnectorType)
	assert.Equal(t, s2.ID, result[1].ID)
	assert.Equal(t, "discord", result[1].ConnectorType)
}

// TestSessionsHandlerGetByID tests the GET /api/v1/sessions/{id} endpoint.
func TestSessionsHandlerGetByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	sessionRepo := repository.NewSessionRepository(db)
	s, err := sessionRepo.FindOrCreate("whatsapp", "555")
	require.NoError(t, err)

	handler := SessionsHandler()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/sessions/"+s.ID, nil)
	req = mux.SetURLVars(req, map[string]string{"id": s.ID})
	req = req.WithContext(context.WithValue(req.Context(), "sessionRepo", sessionRepo))

	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result sessionResponse
	err = json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, s.ID, result.ID)
	assert.Equal(t, "whatsapp", result.ConnectorType)
	assert.Equal(t, "555", result.ChannelID)
}

// TestSessionsHandlerGetNotFound tests 404 on non-existent session.
func TestSessionsHandlerGetNotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	sessionRepo := repository.NewSessionRepository(db)

	handler := SessionsHandler()
	fakeID := uuid.New().String()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/sessions/"+fakeID, nil)
	req = mux.SetURLVars(req, map[string]string{"id": fakeID})
	req = req.WithContext(context.WithValue(req.Context(), "sessionRepo", sessionRepo))

	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestSessionsHandlerPatch tests the PATCH /api/v1/sessions/{id} endpoint.
func TestSessionsHandlerPatch(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	sessionRepo := repository.NewSessionRepository(db)
	s, err := sessionRepo.FindOrCreate("whatsapp", "777")
	require.NoError(t, err)

	// Test updating system prompt.
	handler := SessionsHandler()

	payload := struct {
		SystemPrompt *string `json:"system_prompt"`
	}{
		SystemPrompt: stringPtr("You are a helpful assistant."),
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest(http.MethodPatch, "/api/v1/sessions/"+s.ID, bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"id": s.ID})
	req = req.WithContext(context.WithValue(req.Context(), "sessionRepo", sessionRepo))

	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result sessionResponse
	err = json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, "You are a helpful assistant.", result.SystemPrompt)

	// Verify DB was updated.
	updated, err := sessionRepo.GetByID(s.ID)
	require.NoError(t, err)
	assert.Equal(t, "You are a helpful assistant.", updated.SystemPrompt)
}

// TestSessionsHandlerPatchActive tests updating is_active flag.
func TestSessionsHandlerPatchActive(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	sessionRepo := repository.NewSessionRepository(db)
	s, err := sessionRepo.FindOrCreate("whatsapp", "888")
	require.NoError(t, err)
	assert.True(t, s.IsActive)

	handler := SessionsHandler()

	payload := struct {
		IsActive *bool `json:"is_active"`
	}{
		IsActive: boolPtr(false),
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest(http.MethodPatch, "/api/v1/sessions/"+s.ID, bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"id": s.ID})
	req = req.WithContext(context.WithValue(req.Context(), "sessionRepo", sessionRepo))

	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result sessionResponse
	err = json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)

	assert.False(t, result.IsActive)
}

// setupTestDB creates an in-memory SQLite database for testing.
func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			connector_type TEXT NOT NULL,
			channel_id TEXT NOT NULL,
			system_prompt TEXT,
			is_active INTEGER DEFAULT 1,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			UNIQUE(connector_type, channel_id)
		)
	`)
	require.NoError(t, err)

	return db
}

// stringPtr returns a pointer to a string.
func stringPtr(s string) *string {
	return &s
}

// boolPtr returns a pointer to a bool.
func boolPtr(b bool) *bool {
	return &b
}
