package artifacts

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"bruce/internal/config"
)

func TestArtifactSaveListRead(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "bruce_artifacts_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	cfg := &config.Config{
		App: config.AppConfig{
			BaseURL: "http://bruce.homeserver.local",
		},
		Tools: config.ToolsConfig{
			Artifacts: config.ArtifactsConfig{
				Dir: tempDir,
			},
		},
	}

	saveTool := NewArtifactSaveTool(cfg)
	listTool := NewArtifactListTool(cfg)
	readTool := NewArtifactReadTool(cfg)

	ctx := context.Background()

	// 1. Save HTML artifact
	saveRes, err := saveTool.Execute(ctx, map[string]interface{}{
		"filename": "doc.html",
		"content":  "<!DOCTYPE html><html><body><h1>Test Doc</h1></body></html>",
	})
	require.NoError(t, err)

	var saveObj SaveResponse
	err = json.Unmarshal([]byte(saveRes.(string)), &saveObj)
	require.NoError(t, err)
	assert.Equal(t, "doc.html", saveObj.Filename)
	assert.Equal(t, "/artifacts/doc.html", saveObj.URL)
	assert.Equal(t, "http://bruce.homeserver.local/artifacts/doc.html", saveObj.AbsoluteURL)

	// Verify file was written to disk
	savedBytes, err := os.ReadFile(filepath.Join(tempDir, "doc.html"))
	require.NoError(t, err)
	assert.Contains(t, string(savedBytes), "<h1>Test Doc</h1>")

	// 2. Read artifact
	readRes, err := readTool.Execute(ctx, map[string]interface{}{
		"filename": "doc.html",
	})
	require.NoError(t, err)
	assert.Contains(t, readRes.(string), "<h1>Test Doc</h1>")

	// 3. List artifacts
	listRes, err := listTool.Execute(ctx, map[string]interface{}{})
	require.NoError(t, err)

	var listObj []ArtifactInfo
	err = json.Unmarshal([]byte(listRes.(string)), &listObj)
	require.NoError(t, err)
	require.Len(t, listObj, 1)
	assert.Equal(t, "doc.html", listObj[0].Filename)
	assert.Equal(t, "/artifacts/doc.html", listObj[0].URL)
	assert.Equal(t, "http://bruce.homeserver.local/artifacts/doc.html", listObj[0].AbsoluteURL)
}

func TestArtifactSecurity(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "bruce_artifacts_sec_*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	cfg := &config.Config{
		Tools: config.ToolsConfig{
			Artifacts: config.ArtifactsConfig{
				Dir: tempDir,
			},
		},
	}

	saveTool := NewArtifactSaveTool(cfg)
	ctx := context.Background()

	// Path traversal attempt: ../etc/passwd
	_, err = saveTool.Execute(ctx, map[string]interface{}{
		"filename": "../etc/passwd",
		"content":  "hacked",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "path traversal forbidden")

	// Leading slash attempt: /etc/passwd
	_, err = saveTool.Execute(ctx, map[string]interface{}{
		"filename": "/etc/passwd",
		"content":  "hacked",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "path traversal forbidden")
}
