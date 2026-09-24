package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"bruce/internal/config"
)

func TestArtifactsStaticFileServer(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "bruce_handler_art_*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	htmlContent := `<!DOCTYPE html><html><body><h1>Doc Page</h1></body></html>`
	err = os.WriteFile(filepath.Join(tempDir, "doc.html"), []byte(htmlContent), 0o644)
	require.NoError(t, err)

	server := ArtifactsFileServer(tempDir)

	// 1. Existing file
	req := httptest.NewRequest("GET", "/doc.html", nil)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Header().Get("Content-Type"), "text/html")
	assert.Equal(t, htmlContent, rr.Body.String())

	// 2. Non-existent file -> 404
	req404 := httptest.NewRequest("GET", "/missing.html", nil)
	rr404 := httptest.NewRecorder()
	server.ServeHTTP(rr404, req404)
	assert.Equal(t, http.StatusNotFound, rr404.Code)

	// 3. Traversal attack
	reqSec := httptest.NewRequest("GET", "/../secret.txt", nil)
	rrSec := httptest.NewRecorder()
	server.ServeHTTP(rrSec, reqSec)
	assert.True(t, rrSec.Code == http.StatusForbidden || rrSec.Code == http.StatusNotFound)
}

func TestArtifactsRESTAPI(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "bruce_api_art_*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	err = os.WriteFile(filepath.Join(tempDir, "report.html"), []byte("<h1>Report</h1>"), 0o644)
	require.NoError(t, err)

	cfg := &config.Config{
		App: config.AppConfig{BaseURL: "http://bruce.homeserver.local"},
	}

	// 1. GET /api/v1/artifacts
	listHandler := ListArtifactsHandler(tempDir, cfg)
	req := httptest.NewRequest("GET", "/api/v1/artifacts", nil)
	rr := httptest.NewRecorder()
	listHandler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var items []ArtifactItem
	err = json.Unmarshal(rr.Body.Bytes(), &items)
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "report.html", items[0].Filename)
	assert.Equal(t, "/artifacts/report.html", items[0].URL)
	assert.Equal(t, "http://bruce.homeserver.local/artifacts/report.html", items[0].AbsoluteURL)

	// 2. DELETE /api/v1/artifacts/{filename}
	r := mux.NewRouter()
	r.HandleFunc("/api/v1/artifacts/{filename:.*}", DeleteArtifactHandler(tempDir)).Methods("DELETE")

	delReq := httptest.NewRequest("DELETE", "/api/v1/artifacts/report.html", nil)
	delRR := httptest.NewRecorder()
	r.ServeHTTP(delRR, delReq)
	assert.Equal(t, http.StatusNoContent, delRR.Code)

	// Verify file is gone
	_, statErr := os.Stat(filepath.Join(tempDir, "report.html"))
	assert.True(t, os.IsNotExist(statErr))
}
