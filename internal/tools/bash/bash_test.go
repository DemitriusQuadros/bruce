package bash

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"bruce/internal/config"
)

func TestBashTool_Execute_Echo(t *testing.T) {
	cfg := &config.Config{
		Tools: config.ToolsConfig{
			Bash: config.BashConfig{
				Enabled:        true,
				TimeoutSeconds: 5,
				MaxOutputBytes: 65536,
			},
		},
	}
	tool := New(cfg)

	res, err := tool.Execute(context.Background(), map[string]interface{}{
		"command": "echo 'hello from bash'",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resp, ok := res.(*BashResponse)
	if !ok {
		t.Fatalf("expected *BashResponse, got %T", res)
	}

	if strings.TrimSpace(resp.Stdout) != "hello from bash" {
		t.Errorf("stdout = %q, want 'hello from bash'", resp.Stdout)
	}
	if resp.ExitCode != 0 {
		t.Errorf("exit_code = %d, want 0", resp.ExitCode)
	}
}

func TestBashTool_Execute_NonZeroExit(t *testing.T) {
	tool := NewWithConfig(config.BashConfig{
		Enabled:        true,
		TimeoutSeconds: 5,
	})

	res, err := tool.Execute(context.Background(), map[string]interface{}{
		"command": "exit 42",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resp, ok := res.(*BashResponse)
	if !ok {
		t.Fatalf("expected *BashResponse, got %T", res)
	}

	if resp.ExitCode != 42 {
		t.Errorf("exit_code = %d, want 42", resp.ExitCode)
	}
}

func TestBashTool_Execute_WorkingDir(t *testing.T) {
	tempDir := t.TempDir()
	tempResolved, _ := filepath.EvalSymlinks(tempDir)

	cfg := &config.Config{
		Tools: config.ToolsConfig{
			Bash: config.BashConfig{
				Enabled:        true,
				WorkingDir:     tempDir,
				TimeoutSeconds: 5,
			},
		},
	}
	tool := New(cfg)

	res, err := tool.Execute(context.Background(), map[string]interface{}{
		"command": "pwd",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resp := res.(*BashResponse)
	gotPwd := strings.TrimSpace(resp.Stdout)
	gotPwdResolved, _ := filepath.EvalSymlinks(gotPwd)
	if gotPwdResolved != tempResolved {
		t.Errorf("pwd = %q (resolved %q), want %q", gotPwd, gotPwdResolved, tempResolved)
	}
}
