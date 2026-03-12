package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"bruce/internal/domain"
	"bruce/internal/repository"
	"bruce/internal/worker"
)

// TestFullAPIWorkflow tests a complete end-to-end API workflow.
func TestFullAPIWorkflow(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	sessionRepo := repository.NewSessionRepository(db)
	messageRepo := repository.NewMessageRepository(db)
	configRepo := repository.NewConfigRepository(db)
	registry := worker.NewDispatcherRegistry()

	router := NewRouter(time.Now().Add(-5*time.Second), &mockAsynqmonHandler{})

	wrappedRouter := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		ctx = context.WithValue(ctx, "sessionRepo", sessionRepo)
		ctx = context.WithValue(ctx, "messageRepo", messageRepo)
		ctx = context.WithValue(ctx, "configRepo", configRepo)
		ctx = context.WithValue(ctx, "dispatcherRegistry", registry)
		router.ServeHTTP(w, r.WithContext(ctx))
	})

	// 1. Check health.
	t.Run("health check", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/health", nil)
		w := httptest.NewRecorder()
		wrappedRouter.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var health map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &health)
		assert.Equal(t, "ok", health["status"])
	})

	// 2. Get initial config (should be empty).
	t.Run("get initial config", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/config", nil)
		w := httptest.NewRecorder()
		wrappedRouter.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var config []interface{}
		json.Unmarshal(w.Body.Bytes(), &config)
		assert.Greater(t, len(config), 0)
	})

	// 3. Update config.
	t.Run("update config", func(t *testing.T) {
		payload := map[string]string{
			"key":   "ui.default_system_prompt",
			"value": "You are a helpful assistant.",
		}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPut, "/api/v1/config", bytes.NewReader(body))
		w := httptest.NewRecorder()
		wrappedRouter.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response map[string]bool
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.True(t, response["ok"])
	})

	// 4. Get sessions (should be empty).
	t.Run("get initial sessions", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/sessions", nil)
		w := httptest.NewRecorder()
		wrappedRouter.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var sessions []interface{}
		json.Unmarshal(w.Body.Bytes(), &sessions)
		assert.Equal(t, 0, len(sessions))
	})

	// 5. Create a session via repository (simulating connector creating it).
	var sessionID string
	t.Run("create session", func(t *testing.T) {
		s, err := sessionRepo.FindOrCreate("whatsapp", "123456789")
		require.NoError(t, err)
		sessionID = s.ID

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/sessions", nil)
		w := httptest.NewRecorder()
		wrappedRouter.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var sessions []map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &sessions)
		assert.Equal(t, 1, len(sessions))
		assert.Equal(t, sessionID, sessions[0]["id"])
	})

	// 6. Get specific session.
	t.Run("get session by id", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/sessions/"+sessionID, nil)
		w := httptest.NewRecorder()
		wrappedRouter.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var session map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &session)
		assert.Equal(t, sessionID, session["id"])
		assert.Equal(t, "whatsapp", session["connector_type"])
	})

	// 7. Update session system prompt.
	t.Run("update session system prompt", func(t *testing.T) {
		payload := map[string]string{
			"system_prompt": "You are Bruce, a helpful assistant.",
		}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPatch, "/api/v1/sessions/"+sessionID, bytes.NewReader(body))
		w := httptest.NewRecorder()
		wrappedRouter.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var session map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &session)
		assert.Equal(t, "You are Bruce, a helpful assistant.", session["system_prompt"])
	})

	// 8. Deactivate session.
	t.Run("deactivate session", func(t *testing.T) {
		payload := map[string]bool{
			"is_active": false,
		}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPatch, "/api/v1/sessions/"+sessionID, bytes.NewReader(body))
		w := httptest.NewRecorder()
		wrappedRouter.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var session map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &session)
		assert.False(t, session["is_active"].(bool))
	})

	// 9. Add messages.
	t.Run("add messages", func(t *testing.T) {
		now := time.Now()
		msg1 := &domain.Message{
			ID:        uuid.New().String(),
			SessionID: sessionID,
			Role:      "user",
			Content:   "Hello, Bruce!",
			Timestamp: now.Add(-2 * time.Second),
		}
		err := messageRepo.Insert(msg1)
		require.NoError(t, err)

		msg2 := &domain.Message{
			ID:        uuid.New().String(),
			SessionID: sessionID,
			Role:      "assistant",
			Content:   "Hello! How can I help you?",
			Timestamp: now.Add(-1 * time.Second),
		}
		err = messageRepo.Insert(msg2)
		require.NoError(t, err)

		// Get messages.
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/sessions/"+sessionID+"/messages", nil)
		w := httptest.NewRecorder()
		wrappedRouter.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var messages []map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &messages)
		assert.Equal(t, 2, len(messages))
		// Messages should be in reverse order (newest first).
		assert.Equal(t, "assistant", messages[0]["role"])
		assert.Equal(t, "user", messages[1]["role"])
	})

	// 10. Get connectors status.
	t.Run("get connectors status", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/connectors", nil)
		w := httptest.NewRecorder()
		wrappedRouter.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var connectors []map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &connectors)
		assert.Equal(t, 2, len(connectors))
		assert.Equal(t, "whatsapp", connectors[0]["type"])
		assert.Equal(t, "discord", connectors[1]["type"])
	})

	// 11. Verify 404 for non-existent session.
	t.Run("404 for non-existent session", func(t *testing.T) {
		fakeID := uuid.New().String()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/sessions/"+fakeID, nil)
		w := httptest.NewRecorder()
		wrappedRouter.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		var errResp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &errResp)
		assert.Equal(t, "session not found", errResp["error"])
	})
}

// TestAPIErrorResponses tests error response formats.
func TestAPIErrorResponses(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	sessionRepo := repository.NewSessionRepository(db)
	configRepo := repository.NewConfigRepository(db)

	router := NewRouter(time.Now(), &mockAsynqmonHandler{})

	wrappedRouter := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		ctx = context.WithValue(ctx, "sessionRepo", sessionRepo)
		ctx = context.WithValue(ctx, "configRepo", configRepo)
		router.ServeHTTP(w, r.WithContext(ctx))
	})

	// Test invalid PUT config request (missing key).
	t.Run("invalid config request", func(t *testing.T) {
		payload := map[string]string{
			"value": "test",
		}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPut, "/api/v1/config", bytes.NewReader(body))
		w := httptest.NewRecorder()
		wrappedRouter.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var errResp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &errResp)
		assert.Contains(t, errResp["error"], "missing")
		assert.Equal(t, float64(http.StatusBadRequest), errResp["code"])
	})

	// Test unknown config key.
	t.Run("unknown config key", func(t *testing.T) {
		payload := map[string]string{
			"key":   "unknown.key",
			"value": "test",
		}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPut, "/api/v1/config", bytes.NewReader(body))
		w := httptest.NewRecorder()
		wrappedRouter.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
		var errResp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &errResp)
		assert.Contains(t, errResp["error"], "unknown")
	})
}

// TestAPICORSSupport tests CORS header handling.
func TestAPICORSSupport(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	sessionRepo := repository.NewSessionRepository(db)
	configRepo := repository.NewConfigRepository(db)

	router := NewRouter(time.Now(), &mockAsynqmonHandler{})

	wrappedRouter := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		ctx = context.WithValue(ctx, "sessionRepo", sessionRepo)
		ctx = context.WithValue(ctx, "configRepo", configRepo)
		router.ServeHTTP(w, r.WithContext(ctx))
	})

	// Test CORS headers on regular request.
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/config", nil)
	w := httptest.NewRecorder()
	wrappedRouter.ServeHTTP(w, req)

	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "GET")

	// Test preflight response.
	req, _ = http.NewRequest(http.MethodOptions, "/api/v1/config", nil)
	w = httptest.NewRecorder()
	wrappedRouter.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
}
