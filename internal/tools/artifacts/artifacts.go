package artifacts

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"bruce/internal/ai"
	"bruce/internal/config"
	"bruce/internal/logging"
)

const defaultArtifactsDir = "./data/artifacts"

// ArtifactInfo holds metadata about a saved artifact.
type ArtifactInfo struct {
	Filename    string    `json:"filename"`
	URL         string    `json:"url"`
	AbsoluteURL string    `json:"absolute_url,omitempty"`
	SizeBytes   int64     `json:"size_bytes"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// resolveArtifactsDir returns the canonical artifacts storage directory.
func resolveArtifactsDir(cfg *config.Config) string {
	if cfg != nil && cfg.Tools.Artifacts.Dir != "" {
		return cfg.Tools.Artifacts.Dir
	}
	return defaultArtifactsDir
}

// resolveBaseURL returns the base URL configured for the app or artifacts.
func resolveBaseURL(cfg *config.Config) string {
	if cfg == nil {
		return ""
	}
	if cfg.Tools.Artifacts.BaseURL != "" {
		return strings.TrimRight(cfg.Tools.Artifacts.BaseURL, "/")
	}
	if cfg.App.BaseURL != "" {
		return strings.TrimRight(cfg.App.BaseURL, "/")
	}
	return ""
}

// resolveBaseURLWithContext inspects configRepo in ctx first for dynamic app.base_url.
func resolveBaseURLWithContext(ctx context.Context, cfg *config.Config) string {
	baseURL := resolveBaseURL(cfg)
	if ctx != nil {
		if repo, ok := ctx.Value("configRepo").(interface {
			Get(string) (string, error)
		}); ok && repo != nil {
			if dbURL, err := repo.Get("app.base_url"); err == nil && dbURL != "" {
				baseURL = strings.TrimRight(dbURL, "/")
			}
		}
	}
	return baseURL
}

// validateFilename ensures the filename cannot escape the artifacts directory.
func validateFilename(filename string) (string, error) {
	filename = strings.TrimSpace(filename)
	if filename == "" {
		return "", fmt.Errorf("filename is required")
	}

	// Clean and normalize
	cleaned := filepath.Clean(filename)
	if strings.HasPrefix(cleaned, "..") || strings.HasPrefix(cleaned, "/") || strings.HasPrefix(cleaned, "\\") {
		return "", fmt.Errorf("path traversal forbidden: filename cannot start with '..' or '/'")
	}
	if strings.Contains(cleaned, "../") || strings.Contains(cleaned, "..\\") {
		return "", fmt.Errorf("path traversal forbidden: filename cannot escape storage")
	}

	return cleaned, nil
}

// --- ArtifactSaveTool ---

type ArtifactSaveTool struct {
	cfg *config.Config
}

func NewArtifactSaveTool(cfg *config.Config) *ArtifactSaveTool {
	return &ArtifactSaveTool{cfg: cfg}
}

func (t *ArtifactSaveTool) Name() string { return "artifact_save" }

func (t *ArtifactSaveTool) Definition() ai.ToolDefinition {
	return ai.ToolDefinition{
		Name:        "artifact_save",
		Description: "Save a generated document, report, dashboard, table, or web page as an artifact. HTML files will be immediately rendered as static web pages at /artifacts/<filename>. Always use this tool when the user asks you to create, export, or generate an HTML page or report.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"filename": map[string]interface{}{
					"type":        "string",
					"description": "Filename for the artifact (e.g. 'doc.html', 'market_analysis.html', 'report.md')",
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "Full content to write into the artifact file",
				},
				"title": map[string]interface{}{
					"type":        "string",
					"description": "Optional human-readable title describing the artifact",
				},
			},
			"required": []string{"filename", "content"},
		},
	}
}

type SaveResponse struct {
	Filename    string `json:"filename"`
	URL         string `json:"url"`
	AbsoluteURL string `json:"absolute_url,omitempty"`
	SizeBytes   int    `json:"size_bytes"`
	Message     string `json:"message"`
}

func (t *ArtifactSaveTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	rawFilename, ok := input["filename"].(string)
	if !ok || rawFilename == "" {
		return nil, fmt.Errorf("filename is required")
	}

	content, ok := input["content"].(string)
	if !ok {
		return nil, fmt.Errorf("content is required")
	}

	cleanFilename, err := validateFilename(rawFilename)
	if err != nil {
		return nil, err
	}

	dir := resolveArtifactsDir(t.cfg)
	targetPath := filepath.Join(dir, cleanFilename)

	// Ensure parent directories exist
	parentDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		return nil, fmt.Errorf("create directory: %w", err)
	}

	// Atomic write
	tmpPath := targetPath + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(content), 0o644); err != nil {
		return nil, fmt.Errorf("write artifact temp: %w", err)
	}
	if err := os.Rename(tmpPath, targetPath); err != nil {
		_ = os.Remove(tmpPath)
		return nil, fmt.Errorf("finalize artifact: %w", err)
	}

	relURL := "/artifacts/" + filepath.ToSlash(cleanFilename)
	baseURL := resolveBaseURLWithContext(ctx, t.cfg)
	var absURL string
	if baseURL != "" {
		absURL = baseURL + relURL
	}

	msg := fmt.Sprintf("Artifact saved successfully. Accessible at %s", relURL)
	if absURL != "" {
		msg = fmt.Sprintf("Artifact saved successfully. Accessible at %s (or %s)", absURL, relURL)
	}

	logging.Infof("artifact saved: %s (%d bytes) -> %s", cleanFilename, len(content), relURL)

	resp := SaveResponse{
		Filename:    cleanFilename,
		URL:         relURL,
		AbsoluteURL: absURL,
		SizeBytes:   len(content),
		Message:     msg,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		return nil, err
	}
	return string(data), nil
}

// --- ArtifactListTool ---

type ArtifactListTool struct {
	cfg *config.Config
}

func NewArtifactListTool(cfg *config.Config) *ArtifactListTool {
	return &ArtifactListTool{cfg: cfg}
}

func (t *ArtifactListTool) Name() string { return "artifact_list" }

func (t *ArtifactListTool) Definition() ai.ToolDefinition {
	return ai.ToolDefinition{
		Name:        "artifact_list",
		Description: "List all artifacts currently saved in the project artifacts directory with their filenames, sizes, URLs, and modification timestamps.",
		InputSchema: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
	}
}

func (t *ArtifactListTool) Execute(ctx context.Context, _ map[string]interface{}) (interface{}, error) {
	dir := resolveArtifactsDir(t.cfg)
	baseURL := resolveBaseURLWithContext(ctx, t.cfg)

	var items []ArtifactInfo

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if info.IsDir() {
			return nil
		}
		if strings.HasSuffix(info.Name(), ".tmp") {
			return nil
		}

		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return nil
		}
		slashPath := filepath.ToSlash(relPath)
		relURL := "/artifacts/" + slashPath
		var absURL string
		if baseURL != "" {
			absURL = baseURL + relURL
		}

		items = append(items, ArtifactInfo{
			Filename:    slashPath,
			URL:         relURL,
			AbsoluteURL: absURL,
			SizeBytes:   info.Size(),
			UpdatedAt:   info.ModTime().UTC(),
		})
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("list artifacts: %w", err)
	}

	if items == nil {
		items = []ArtifactInfo{}
	}

	data, err := json.Marshal(items)
	if err != nil {
		return nil, err
	}
	return string(data), nil
}

// --- ArtifactReadTool ---

type ArtifactReadTool struct {
	cfg *config.Config
}

func NewArtifactReadTool(cfg *config.Config) *ArtifactReadTool {
	return &ArtifactReadTool{cfg: cfg}
}

func (t *ArtifactReadTool) Name() string { return "artifact_read" }

func (t *ArtifactReadTool) Definition() ai.ToolDefinition {
	return ai.ToolDefinition{
		Name:        "artifact_read",
		Description: "Read the full text content of an existing artifact by filename.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"filename": map[string]interface{}{
					"type":        "string",
					"description": "Filename of the artifact to read (e.g. 'doc.html')",
				},
			},
			"required": []string{"filename"},
		},
	}
}

func (t *ArtifactReadTool) Execute(_ context.Context, input map[string]interface{}) (interface{}, error) {
	rawFilename, ok := input["filename"].(string)
	if !ok || rawFilename == "" {
		return nil, fmt.Errorf("filename is required")
	}

	cleanFilename, err := validateFilename(rawFilename)
	if err != nil {
		return nil, err
	}

	dir := resolveArtifactsDir(t.cfg)
	targetPath := filepath.Join(dir, cleanFilename)

	data, err := os.ReadFile(targetPath)
	if err != nil {
		return nil, fmt.Errorf("read artifact %q: %w", cleanFilename, err)
	}

	return string(data), nil
}
