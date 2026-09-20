package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"bruce/internal/domain"
	"bruce/internal/repository"
)

func TestSessionSummaryHandler(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Ensure session_summaries table exists
	_, err := db.Exec(`
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

	sessionRepo := repository.NewSessionRepository(db)
	summaryRepo := repository.NewSessionSummaryRepository(db)

	session, err := sessionRepo.Create("web", "chan-api-1", "Summary Test")
	require.NoError(t, err)

	handler := SessionSummaryHandler()

	t.Run("Session Not Found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/sessions/non-existent/summary", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "non-existent"})
		ctx := context.WithValue(req.Context(), "sessionRepo", sessionRepo)
		ctx = context.WithValue(ctx, "sessionSummaryRepo", summaryRepo)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("No Summary Yet", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/sessions/"+session.ID+"/summary", nil)
		req = mux.SetURLVars(req, map[string]string{"id": session.ID})
		ctx := context.WithValue(req.Context(), "sessionRepo", sessionRepo)
		ctx = context.WithValue(ctx, "sessionSummaryRepo", summaryRepo)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var resp sessionSummaryResponse
		err := json.Unmarshal(rr.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, session.ID, resp.SessionID)
		assert.Empty(t, resp.Summary)
		assert.Equal(t, 0, resp.MessageCount)
	})

	t.Run("With Summary", func(t *testing.T) {
		err := summaryRepo.Upsert(&domain.SessionSummary{
			SessionID:           session.ID,
			Summary:             "• Discussed background summarization.\n• Implemented memory system.",
			LastSummarizedMsgID: "msg-123",
			MessageCount:        15,
			UpdatedAt:           time.Now().UTC(),
		})
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/sessions/"+session.ID+"/summary", nil)
		req = mux.SetURLVars(req, map[string]string{"id": session.ID})
		ctx := context.WithValue(req.Context(), "sessionRepo", sessionRepo)
		ctx = context.WithValue(ctx, "sessionSummaryRepo", summaryRepo)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var resp sessionSummaryResponse
		err = json.Unmarshal(rr.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, session.ID, resp.SessionID)
		assert.Contains(t, resp.Summary, "Discussed background summarization")
		assert.Equal(t, 15, resp.MessageCount)
	})
}
