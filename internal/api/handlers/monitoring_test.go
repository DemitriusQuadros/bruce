package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"bruce/internal/database"
	"bruce/internal/repository"
)

// TestLogsHandlerWithValidRepo tests that LogsHandler correctly retrieves the monitoring repository from context.
func TestLogsHandlerWithValidRepo(t *testing.T) {
	// Create in-memory SQLite database
	db, err := database.NewSQLiteDB(":memory:")
	require.NoError(t, err, "failed to create database")
	defer db.Close()

	// Run migrations
	err = database.RunMigrations(db)
	require.NoError(t, err, "failed to run migrations")

	// Create monitoring repository
	monitoringRepo := repository.NewMonitoringRepository(db)

	// Create a test request with the repository in context
	req := httptest.NewRequest(http.MethodGet, "/api/v1/logs", nil)
	ctx := context.WithValue(req.Context(), "monitoringRepo", monitoringRepo)
	req = req.WithContext(ctx)

	// Create a response recorder
	w := httptest.NewRecorder()

	// Call the handler
	LogsHandler()(w, req)

	// Check that the response is successful
	assert.Equal(t, http.StatusOK, w.Code)

	// Unmarshal the response
	var resp logsResponse
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.IsType(t, logsResponse{}, resp)
}

// TestLogsHandlerWithoutRepo tests that LogsHandler returns an error when the monitoring repository is missing.
func TestLogsHandlerWithoutRepo(t *testing.T) {
	// Create a test request WITHOUT the repository in context
	req := httptest.NewRequest(http.MethodGet, "/api/v1/logs", nil)

	// Create a response recorder
	w := httptest.NewRecorder()

	// Call the handler
	LogsHandler()(w, req)

	// Check that the response is a 500 error
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "monitoring repository not available")
}

// TestMetricsHandlerWithValidRepo tests that MetricsHandler correctly retrieves the monitoring repository from context.
func TestMetricsHandlerWithValidRepo(t *testing.T) {
	// Create in-memory SQLite database
	db, err := database.NewSQLiteDB(":memory:")
	require.NoError(t, err, "failed to create database")
	defer db.Close()

	// Run migrations
	err = database.RunMigrations(db)
	require.NoError(t, err, "failed to run migrations")

	// Create monitoring repository
	monitoringRepo := repository.NewMonitoringRepository(db)

	// Create a test request with the repository in context
	req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics", nil)
	ctx := context.WithValue(req.Context(), "monitoringRepo", monitoringRepo)
	req = req.WithContext(ctx)

	// Create a response recorder
	w := httptest.NewRecorder()

	// Call the handler
	MetricsHandler()(w, req)

	// Check that the response is successful
	assert.Equal(t, http.StatusOK, w.Code)

	// Unmarshal the response
	var resp metricsResponse
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.IsType(t, metricsResponse{}, resp)
}

// TestMetricsHandlerWithoutRepo tests that MetricsHandler returns an error when the monitoring repository is missing.
func TestMetricsHandlerWithoutRepo(t *testing.T) {
	// Create a test request WITHOUT the repository in context
	req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics", nil)

	// Create a response recorder
	w := httptest.NewRecorder()

	// Call the handler
	MetricsHandler()(w, req)

	// Check that the response is a 500 error
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "monitoring repository not available")
}

// TestMonitoringConfigGetHandlerWithValidRepo tests that MonitoringConfigGetHandler correctly retrieves the monitoring repository from context.
func TestMonitoringConfigGetHandlerWithValidRepo(t *testing.T) {
	// Create in-memory SQLite database
	db, err := database.NewSQLiteDB(":memory:")
	require.NoError(t, err, "failed to create database")
	defer db.Close()

	// Run migrations
	err = database.RunMigrations(db)
	require.NoError(t, err, "failed to run migrations")

	// Create monitoring repository
	monitoringRepo := repository.NewMonitoringRepository(db)

	// Create a test request with the repository in context
	req := httptest.NewRequest(http.MethodGet, "/api/v1/monitoring/config", nil)
	ctx := context.WithValue(req.Context(), "monitoringRepo", monitoringRepo)
	req = req.WithContext(ctx)

	// Create a response recorder
	w := httptest.NewRecorder()

	// Call the handler
	MonitoringConfigGetHandler()(w, req)

	// Check that the response is successful
	assert.Equal(t, http.StatusOK, w.Code)

	// Unmarshal the response
	var resp monitoringConfigResponse
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.IsType(t, monitoringConfigResponse{}, resp)
	// Check default values
	assert.True(t, resp.LogEnabled)
	assert.True(t, resp.MetricsEnabled)
	assert.Equal(t, 30, resp.RetentionDays)
}

// TestMonitoringConfigGetHandlerWithoutRepo tests that MonitoringConfigGetHandler returns an error when the monitoring repository is missing.
func TestMonitoringConfigGetHandlerWithoutRepo(t *testing.T) {
	// Create a test request WITHOUT the repository in context
	req := httptest.NewRequest(http.MethodGet, "/api/v1/monitoring/config", nil)

	// Create a response recorder
	w := httptest.NewRecorder()

	// Call the handler
	MonitoringConfigGetHandler()(w, req)

	// Check that the response is a 500 error
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "monitoring repository not available")
}

// TestMonitoringConfigPatchHandlerWithValidRepo tests that MonitoringConfigPatchHandler correctly retrieves the monitoring repository from context.
func TestMonitoringConfigPatchHandlerWithValidRepo(t *testing.T) {
	// Create in-memory SQLite database
	db, err := database.NewSQLiteDB(":memory:")
	require.NoError(t, err, "failed to create database")
	defer db.Close()

	// Run migrations
	err = database.RunMigrations(db)
	require.NoError(t, err, "failed to run migrations")

	// Create monitoring repository
	monitoringRepo := repository.NewMonitoringRepository(db)

	// Create a test request with the repository in context
	body := `{"log_enabled": false, "metrics_enabled": false, "retention_days": 60}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/monitoring/config", nil)
	req.Body = http.NoBody
	// Manually set body from string
	req.ContentLength = int64(len(body))
	req.Header.Set("Content-Type", "application/json")

	// Recreate request with proper body
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/monitoring/config", http.NoBody)
	ctx := context.WithValue(req.Context(), "monitoringRepo", monitoringRepo)
	req = req.WithContext(ctx)

	// Create a response recorder
	w := httptest.NewRecorder()

	// Call the handler
	MonitoringConfigPatchHandler()(w, req)

	// Should fail due to invalid JSON body, but should NOT fail due to missing repo
	assert.NotEqual(t, http.StatusInternalServerError, w.Code, "should not get 'monitoring repository not available' error")
}

// TestMonitoringConfigPatchHandlerWithoutRepo tests that MonitoringConfigPatchHandler returns an error when the monitoring repository is missing.
func TestMonitoringConfigPatchHandlerWithoutRepo(t *testing.T) {
	// Create a test request WITHOUT the repository in context
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/monitoring/config", nil)

	// Create a response recorder
	w := httptest.NewRecorder()

	// Call the handler
	MonitoringConfigPatchHandler()(w, req)

	// Check that the response is a 500 error
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "monitoring repository not available")
}
