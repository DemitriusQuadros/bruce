package files_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"bruce/internal/config"
	"bruce/internal/tools/files"
)

func TestFileWriteAndReadTool_DynamicConfig(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	cfg := &config.Config{
		Tools: config.ToolsConfig{
			Files: config.FilesConfig{
				Enabled:     true,
				HomeDir:     dir1,
				MaxFileSize: 102400,
			},
		},
	}

	writeTool := files.NewFileWriteTool(cfg)
	readTool := files.NewFileReadTool(cfg)

	// Write to dir1
	writeResp, err := writeTool.Execute(context.Background(), map[string]interface{}{
		"path":    "hello.txt",
		"content": "Hello World from dir1!",
	})
	require.NoError(t, err)
	assert.NotNil(t, writeResp)

	// Read from dir1
	readResp, err := readTool.Execute(context.Background(), map[string]interface{}{
		"path": "hello.txt",
	})
	require.NoError(t, err)
	fileRead, ok := readResp.(*files.FileReadResponse)
	require.True(t, ok)
	assert.Equal(t, "Hello World from dir1!", fileRead.Content)

	// Dynamically update home_dir to dir2 (simulates PUT /api/v1/config)
	cfg.Tools.Files.HomeDir = dir2

	// Write to dir2
	writeResp2, err := writeTool.Execute(context.Background(), map[string]interface{}{
		"path":    "hello.txt",
		"content": "Hello World from dir2!",
	})
	require.NoError(t, err)
	assert.NotNil(t, writeResp2)

	// Read from dir2
	readResp2, err := readTool.Execute(context.Background(), map[string]interface{}{
		"path": "hello.txt",
	})
	require.NoError(t, err)
	fileRead2, ok := readResp2.(*files.FileReadResponse)
	require.True(t, ok)
	assert.Equal(t, "Hello World from dir2!", fileRead2.Content)
}
