package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"bruce/internal/repository"
	"bruce/internal/worker"
)

// MockDispatcher for testing.
type MockDispatcher struct{}

func (m *MockDispatcher) Send(channelID, message string) error {
	return nil
}

// TestRouterMiddlewareStack verifies the middleware stack is applied.
func TestRouterMiddlewareStack(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	sessionRepo := repository.NewSessionRepository(db)
	messageRepo := repository.NewMessageRepository(db)
	configRepo := repository.NewConfigRepository(db)
	registry := worker.NewDispatcherRegistry()

	router := NewRouter(time.Now(), &mockAsynqmonHandler{})

	// Wrap router with context injection.
	wrappedRouter := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		ctx = context.WithValue(ctx, "sessionRepo", sessionRepo)
		ctx = context.WithValue(ctx, "messageRepo", messageRepo)
		ctx = context.WithValue(ctx, "configRepo", configRepo)
		ctx = context.WithValue(ctx, "dispatcherRegistry", registry)
		router.ServeHTTP(w, r.WithContext(ctx))
	})

	// Test that middleware doesn't break normal request flow.
	req, _ := http.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	wrappedRouter.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Origin"), "*")
}

// TestRouterCORSHeaders verifies CORS headers are set.
func TestRouterCORSHeaders(t *testing.T) {
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

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/config", nil)
	w := httptest.NewRecorder()
	wrappedRouter.ServeHTTP(w, req)

	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "GET")
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "PUT")
}

// TestRouterOPTIONSPreflight verifies CORS preflight requests work.
func TestRouterOPTIONSPreflight(t *testing.T) {
	router := NewRouter(time.Now(), &mockAsynqmonHandler{})

	req, _ := http.NewRequest(http.MethodOptions, "/api/v1/config", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
}

// TestRouterHealthEndpoint verifies /health is accessible without auth.
func TestRouterHealthEndpoint(t *testing.T) {
	startTime := time.Now().Add(-10 * time.Second) // Set start time 10 seconds ago.
	router := NewRouter(startTime, &mockAsynqmonHandler{})

	req, _ := http.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "ok", response["status"])
	assert.Greater(t, response["uptime_seconds"], float64(0))
}

// TestRouterAPIv1Routes verifies /api/v1 routes are registered.
func TestRouterAPIv1Routes(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	sessionRepo := repository.NewSessionRepository(db)
	messageRepo := repository.NewMessageRepository(db)
	configRepo := repository.NewConfigRepository(db)
	registry := worker.NewDispatcherRegistry()

	router := NewRouter(time.Now(), &mockAsynqmonHandler{})

	wrappedRouter := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		ctx = context.WithValue(ctx, "sessionRepo", sessionRepo)
		ctx = context.WithValue(ctx, "messageRepo", messageRepo)
		ctx = context.WithValue(ctx, "configRepo", configRepo)
		ctx = context.WithValue(ctx, "dispatcherRegistry", registry)
		router.ServeHTTP(w, r.WithContext(ctx))
	})

	routes := []struct {
		method string
		path   string
		code   int
	}{
		{http.MethodGet, "/api/v1/config", http.StatusOK},
		{http.MethodGet, "/api/v1/sessions", http.StatusOK},
		{http.MethodGet, "/api/v1/connectors", http.StatusOK},
	}

	for _, rt := range routes {
		t.Run(rt.method+" "+rt.path, func(t *testing.T) {
			req, _ := http.NewRequest(rt.method, rt.path, nil)
			w := httptest.NewRecorder()
			wrappedRouter.ServeHTTP(w, req)
			assert.Equal(t, rt.code, w.Code)
		})
	}
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

	_, err = db.Exec(`
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

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS config_entries (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at DATETIME NOT NULL
		)
	`)
	require.NoError(t, err)

	return db
}

// mockAsynqmonHandler for testing.
type mockAsynqmonHandler struct{}

func (m *mockAsynqmonHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
