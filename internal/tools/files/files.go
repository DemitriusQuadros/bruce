// Package files provides file_read and file_write tools for the Bruce AI assistant.
package files

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"bruce/internal/ai"
	"bruce/internal/config"
)

const defaultMaxFileSize = 102400 // 100KB

// FileReadTool implements the tools.Tool interface for reading local files.
type FileReadTool struct {
	cfg config.FilesConfig
}

// NewFileReadTool creates a new FileReadTool.
func NewFileReadTool(cfg config.FilesConfig) *FileReadTool {
	if cfg.MaxFileSize <= 0 {
		cfg.MaxFileSize = defaultMaxFileSize
	}
	return &FileReadTool{cfg: cfg}
}

// Name returns the tool name.
func (t *FileReadTool) Name() string { return "file_read" }

// Definition returns the tool definition for the AI model.
func (t *FileReadTool) Definition() ai.ToolDefinition {
	return ai.ToolDefinition{
		Name:        "file_read",
		Description: "Read the contents of a file from the configured home directory. Paths must be relative and cannot escape the home directory.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Relative path to the file within the home directory",
				},
			},
			"required": []string{"path"},
		},
	}
}

// FileReadResponse is the response structure for file_read.
type FileReadResponse struct {
	Content   string `json:"content"`
	Path      string `json:"path"`
	SizeBytes int    `json:"size_bytes"`
	Truncated bool   `json:"truncated"`
}

// Execute reads the specified file.
func (t *FileReadTool) Execute(_ context.Context, input map[string]interface{}) (interface{}, error) {
	path, ok := input["path"].(string)
	if !ok || path == "" {
		return nil, fmt.Errorf("path is required and must be a string")
	}

	canonical, err := ValidatePath(t.cfg.HomeDir, path)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(canonical)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %q: %w", path, err)
	}
	defer f.Close()

	limited := io.LimitReader(f, int64(t.cfg.MaxFileSize)+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %q: %w", path, err)
	}

	truncated := false
	if len(data) > t.cfg.MaxFileSize {
		data = data[:t.cfg.MaxFileSize]
		truncated = true
	}

	return &FileReadResponse{
		Content:   string(data),
		Path:      canonical,
		SizeBytes: len(data),
		Truncated: truncated,
	}, nil
}

// FileWriteTool implements the tools.Tool interface for writing local files.
type FileWriteTool struct {
	cfg config.FilesConfig
}

// NewFileWriteTool creates a new FileWriteTool.
func NewFileWriteTool(cfg config.FilesConfig) *FileWriteTool {
	if cfg.MaxFileSize <= 0 {
		cfg.MaxFileSize = defaultMaxFileSize
	}
	return &FileWriteTool{cfg: cfg}
}

// Name returns the tool name.
func (t *FileWriteTool) Name() string { return "file_write" }

// Definition returns the tool definition for the AI model.
func (t *FileWriteTool) Definition() ai.ToolDefinition {
	return ai.ToolDefinition{
		Name:        "file_write",
		Description: "Write content to a file in the configured home directory. Creates the file and any missing parent directories. Overwrites existing files. Always writes atomically. Paths must be relative and cannot escape the home directory.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Relative path to the file within the home directory",
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "Content to write to the file",
				},
			},
			"required": []string{"path", "content"},
		},
	}
}

// FileWriteResponse is the response structure for file_write.
type FileWriteResponse struct {
	Path              string `json:"path"`
	SizeBytes         int    `json:"size_bytes"`
	Created           bool   `json:"created"`
	Overwritten       bool   `json:"overwritten"`
	ParentDirsCreated bool   `json:"parent_dirs_created"`
}

// Execute writes content to the specified file atomically.
func (t *FileWriteTool) Execute(_ context.Context, input map[string]interface{}) (interface{}, error) {
	path, ok := input["path"].(string)
	if !ok || path == "" {
		return nil, fmt.Errorf("path is required and must be a string")
	}
	content, _ := input["content"].(string)

	canonical, err := ValidatePath(t.cfg.HomeDir, path)
	if err != nil {
		return nil, err
	}

	// Check if file already exists.
	_, statErr := os.Stat(canonical)
	overwritten := statErr == nil

	// Create parent directories if needed.
	dir := filepath.Dir(canonical)
	parentDirsCreated := false
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("failed to create parent directories for %q: %w", path, err)
		}
		parentDirsCreated = true
	}

	// Atomic write: write to temp file then rename.
	tmpPath := canonical + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(content), 0o644); err != nil {
		return nil, fmt.Errorf("failed to write file %q: %w", path, err)
	}
	if err := os.Rename(tmpPath, canonical); err != nil {
		_ = os.Remove(tmpPath)
		return nil, fmt.Errorf("failed to finalize file %q: %w", path, err)
	}

	return &FileWriteResponse{
		Path:              canonical,
		SizeBytes:         len(content),
		Created:           !overwritten,
		Overwritten:       overwritten,
		ParentDirsCreated: parentDirsCreated,
	}, nil
}
