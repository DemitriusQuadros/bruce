package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"bruce/internal/repository"
)

// TestConfigHandlerGET tests the GET /api/v1/config endpoint.
func TestConfigHandlerGET(t *testing.T) {
	// Set up in-memory SQLite database.
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Create config_entries table.
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS config_entries (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at DATETIME NOT NULL
		)
	`)
	require.NoError(t, err)

	// Create config repository.
	configRepo := repository.NewConfigRepository(db)

	// Insert a test config value.
	err = configRepo.Upsert("claude.api_key", "sk-test-123")
	require.NoError(t, err)

	// Create request with repository in context.
	handler := ConfigHandler()
	req, err := http.NewRequest(http.MethodGet, "/api/v1/config", nil)
	require.NoError(t, err)
	req = req.WithContext(context.WithValue(req.Context(), "configRepo", configRepo))

	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result []configEntry
	err = json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)

	// Verify that the response contains masked api_key.
	found := false
	for _, entry := range result {
		if entry.Key == "claude.api_key" {
			assert.Equal(t, "****", entry.Value)
			found = true
			break
		}
	}
	assert.True(t, found, "claude.api_key should be in response")
}

// TestConfigHandlerPUT tests the PUT /api/v1/config endpoint.
func TestConfigHandlerPUT(t *testing.T) {
	// Set up in-memory SQLite database.
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Create config_entries table.
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS config_entries (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at DATETIME NOT NULL
		)
	`)
	require.NoError(t, err)

	configRepo := repository.NewConfigRepository(db)

	handler := ConfigHandler()

	// Test valid config update.
	payload := struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}{
		Key:   "ui.default_system_prompt",
		Value: "You are Bruce.",
	}

	body, err := json.Marshal(payload)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPut, "/api/v1/config", bytes.NewReader(body))
	require.NoError(t, err)
	req = req.WithContext(context.WithValue(req.Context(), "configRepo", configRepo))

	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]bool
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.True(t, response["ok"])

	// Verify the value was stored.
	value, err := configRepo.Get("ui.default_system_prompt")
	require.NoError(t, err)
	assert.Equal(t, "You are Bruce.", value)
}

// TestConfigHandlerPUTValidation tests input validation for PUT.
func TestConfigHandlerPUTValidation(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS config_entries (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at DATETIME NOT NULL
		)
	`)
	require.NoError(t, err)

	configRepo := repository.NewConfigRepository(db)
	handler := ConfigHandler()

	// Test missing key.
	payload := struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}{
		Key:   "",
		Value: "test",
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/config", bytes.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), "configRepo", configRepo))

	w := httptest.NewRecorder()
	handler(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Test unknown key.
	payload.Key = "unknown.key"
	payload.Value = "value"
	body, _ = json.Marshal(payload)
	req, _ = http.NewRequest(http.MethodPut, "/api/v1/config", bytes.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), "configRepo", configRepo))

	w = httptest.NewRecorder()
	handler(w, req)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

// TestIsSensitiveKey tests the sensitive key detection.
func TestIsSensitiveKey(t *testing.T) {
	tests := []struct {
		key       string
		sensitive bool
	}{
		{"claude.api_key", true},
		{"claude.model", false},
		{"connectors.discord.bot_token", true},
		{"connectors.discord.enabled", false},
		{"some.secret", true},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			assert.Equal(t, tt.sensitive, isSensitiveKey(tt.key))
		})
	}
}
