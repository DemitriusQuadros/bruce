package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"bruce/internal/worker"
)

// MockDispatcher implements worker.Dispatcher for testing.
type MockDispatcher struct{}

func (m *MockDispatcher) Send(channelID, message string) error {
	return nil
}

// TestConnectorsHandlerNoRegistry tests when dispatcher registry is not in context.
func TestConnectorsHandlerNoRegistry(t *testing.T) {
	handler := ConnectorsHandler()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/connectors", nil)
	req = req.WithContext(context.Background())

	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result []connectorResponse
	err := json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)

	// Should return all three connectors as disabled.
	assert.Equal(t, 3, len(result))
	assert.Equal(t, "whatsapp", result[0].Type)
	assert.False(t, result[0].Enabled)
	assert.Equal(t, "disabled", result[0].Status)
	assert.Equal(t, "discord", result[1].Type)
	assert.False(t, result[1].Enabled)
	assert.Equal(t, "disabled", result[1].Status)
	assert.Equal(t, "telegram", result[2].Type)
	assert.False(t, result[2].Enabled)
	assert.Equal(t, "disabled", result[2].Status)
}

// TestConnectorsHandlerWithRegistry tests with active connectors.
func TestConnectorsHandlerWithRegistry(t *testing.T) {
	registry := worker.NewDispatcherRegistry()

	// Register WhatsApp connector.
	registry.Register("whatsapp", &MockDispatcher{})

	handler := ConnectorsHandler()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/connectors", nil)
	req = req.WithContext(context.WithValue(req.Context(), "dispatcherRegistry", registry))

	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result []connectorResponse
	err := json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, 3, len(result))

	// WhatsApp should be enabled and connected.
	assert.Equal(t, "whatsapp", result[0].Type)
	assert.True(t, result[0].Enabled)
	assert.Equal(t, "connected", result[0].Status)

	// Discord should be disabled.
	assert.Equal(t, "discord", result[1].Type)
	assert.False(t, result[1].Enabled)
	assert.Equal(t, "disabled", result[1].Status)

	// Telegram should be disabled.
	assert.Equal(t, "telegram", result[2].Type)
	assert.False(t, result[2].Enabled)
	assert.Equal(t, "disabled", result[2].Status)
}

// TestConnectorsHandlerBothEnabled tests with both connectors enabled.
func TestConnectorsHandlerBothEnabled(t *testing.T) {
	registry := worker.NewDispatcherRegistry()

	// Register all three connectors.
	registry.Register("whatsapp", &MockDispatcher{})
	registry.Register("discord", &MockDispatcher{})
	registry.Register("telegram", &MockDispatcher{})

	handler := ConnectorsHandler()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/connectors", nil)
	req = req.WithContext(context.WithValue(req.Context(), "dispatcherRegistry", registry))

	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result []connectorResponse
	err := json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, 3, len(result))

	// All three should be enabled and connected.
	for _, c := range result {
		assert.True(t, c.Enabled)
		assert.Equal(t, "connected", c.Status)
	}
}

// TestGetStatus tests the getStatus helper function.
func TestGetStatus(t *testing.T) {
	assert.Equal(t, "connected", getStatus(true))
	assert.Equal(t, "disabled", getStatus(false))
}
