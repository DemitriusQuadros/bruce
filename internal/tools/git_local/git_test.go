package git_local

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"bruce/internal/config"
)

func TestValidateRepoPath(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "git-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	subDir := filepath.Join(tempDir, "my-repo")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create sub dir: %v", err)
	}

	t.Run("empty or dot path resolves to homeDir", func(t *testing.T) {
		got, err := validateRepoPath(tempDir, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected, _ := filepath.EvalSymlinks(tempDir)
		if got != expected {
			t.Errorf("got %q, expected %q", got, expected)
		}

		gotDot, err := validateRepoPath(tempDir, ".")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if gotDot != expected {
			t.Errorf("got %q, expected %q", gotDot, expected)
		}
	})

	t.Run("relative subpath", func(t *testing.T) {
		got, err := validateRepoPath(tempDir, "my-repo")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected, _ := filepath.EvalSymlinks(subDir)
		if got != expected {
			t.Errorf("got %q, expected %q", got, expected)
		}
	})

	t.Run("absolute path within homeDir", func(t *testing.T) {
		got, err := validateRepoPath(tempDir, subDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected, _ := filepath.EvalSymlinks(subDir)
		if got != expected {
			t.Errorf("got %q, expected %q", got, expected)
		}
	})

	t.Run("path traversal rejected", func(t *testing.T) {
		_, err := validateRepoPath(tempDir, "../escape")
		if err == nil {
			t.Errorf("expected error on traversal path")
		}
	})

	t.Run("absolute path outside rejected", func(t *testing.T) {
		_, err := validateRepoPath(tempDir, "/etc")
		if err == nil {
			t.Errorf("expected error on outside path")
		}
	})
}

func TestResolveConfig(t *testing.T) {
	cfg := &config.Config{
		Tools: config.ToolsConfig{
			GitLocal: config.GitLocalConfig{
				HomeDir:        "/custom/git/path",
				TimeoutSeconds: 45,
			},
		},
	}

	home, timeout := resolveConfig(cfg)
	if home != "/custom/git/path" {
		t.Errorf("got home %q, expected /custom/git/path", home)
	}
	if timeout != 45*time.Second {
		t.Errorf("got timeout %v, expected 45s", timeout)
	}

	// Nil config fallback
	homeNil, timeoutNil := resolveConfig(nil)
	if homeNil != os.Getenv("HOME") {
		t.Errorf("got %q, expected $HOME", homeNil)
	}
	if timeoutNil != 30*time.Second {
		t.Errorf("got %v, expected 30s", timeoutNil)
	}
}

func TestGitStatusTool_Execute(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "git-status-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := &config.Config{
		Tools: config.ToolsConfig{
			GitLocal: config.GitLocalConfig{
				HomeDir:        tempDir,
				TimeoutSeconds: 5,
			},
		},
	}

	tool := NewStatusTool(cfg)
	if tool.Name() != "git_status" {
		t.Errorf("tool name mismatch: got %q", tool.Name())
	}

	// Not a git repo should fail gracefully with an error
	_, err = tool.Execute(context.Background(), map[string]interface{}{})
	if err == nil {
		t.Errorf("expected error when running in non-git repo")
	}
}
