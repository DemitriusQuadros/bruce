package bash

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"

	"bruce/internal/ai"
	"bruce/internal/config"
)

// BashTool implements the tools.Tool interface for bash command execution.
type BashTool struct {
	cfg          *config.Config
	staticConfig config.BashConfig
}

// New creates a new BashTool with dynamic configuration.
func New(cfg *config.Config) *BashTool {
	return &BashTool{cfg: cfg}
}

// NewWithConfig creates a new BashTool with static configuration.
func NewWithConfig(cfg config.BashConfig) *BashTool {
	return &BashTool{staticConfig: cfg}
}

func (bt *BashTool) getConfig() config.BashConfig {
	var c config.BashConfig
	if bt.cfg != nil {
		c = bt.cfg.Tools.Bash
	} else {
		c = bt.staticConfig
	}
	if c.MaxOutputBytes <= 0 {
		c.MaxOutputBytes = 65536 // 64KB default
	}
	if c.TimeoutSeconds <= 0 {
		c.TimeoutSeconds = 30 // 30s default
	}
	if c.TimeoutSeconds > 60 {
		c.TimeoutSeconds = 60 // 60s hard cap
	}
	return c
}

// Name returns the tool name.
func (bt *BashTool) Name() string {
	return "bash_exec"
}

// Definition returns the AI tool definition for the AI model.
func (bt *BashTool) Definition() ai.ToolDefinition {
	return ai.ToolDefinition{
		Name:        "bash_exec",
		Description: "Execute a bash command with restrictions and timeout. Non-zero exit codes are reported in the response, not as errors.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"command": map[string]interface{}{
					"type":        "string",
					"description": "Bash command to execute (passed to bash -c)",
				},
				"working_dir": map[string]interface{}{
					"type":        "string",
					"description": "Subdirectory under configured base (or absolute path within base, defaults to base dir)",
				},
				"timeout_seconds": map[string]interface{}{
					"type":        "integer",
					"description": "Command timeout in seconds (capped at config max, hard cap 60)",
				},
			},
			"required": []string{"command"},
		},
	}
}

// BashResponse is the response structure for bash_exec tool.
type BashResponse struct {
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	ExitCode   int    `json:"exit_code"`
	Truncated  bool   `json:"truncated"`
	WorkingDir string `json:"working_dir"`
}

// Execute runs the bash command with validation and security checks.
func (bt *BashTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	// Extract inputs.
	command, ok := input["command"].(string)
	if !ok || command == "" {
		return nil, fmt.Errorf("command is required and must be a string")
	}

	workingDir, _ := input["working_dir"].(string)
	timeoutSeconds, _ := input["timeout_seconds"].(float64)

	conf := bt.getConfig()

	// Validate command.
	if err := ValidateCommand(command, conf.AllowedCommands); err != nil {
		return nil, err
	}

	// Validate and resolve working directory.
	resolvedDir, err := ValidateWorkingDir(conf.WorkingDir, workingDir)
	if err != nil {
		return nil, err
	}

	// Ensure working directory exists.
	if err := os.MkdirAll(resolvedDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create working directory %q: %w", resolvedDir, err)
	}

	// Determine timeout: use request timeout if provided and valid, otherwise config default.
	timeout := time.Duration(conf.TimeoutSeconds) * time.Second
	if timeoutSeconds > 0 {
		requestTimeout := int(timeoutSeconds)
		if requestTimeout > 60 {
			requestTimeout = 60 // Hard cap at 60s
		}
		timeout = time.Duration(requestTimeout) * time.Second
	}

	// Execute command with timeout.
	resp, err := bt.executeWithTimeout(ctx, command, resolvedDir, timeout, conf.MaxOutputBytes)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// executeWithTimeout runs the command with timeout enforcement.
func (bt *BashTool) executeWithTimeout(ctx context.Context, command, workingDir string, timeout time.Duration, maxOutputBytes int) (*BashResponse, error) {
	// Create a context with timeout.
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Create the command.
	cmd := exec.CommandContext(execCtx, "bash", "-c", command)
	cmd.Dir = workingDir
	cmd.Stdin = nil // No stdin

	// Restrict environment: only PATH, HOME, USER, LANG, TERM, PAGER.
	cmd.Env = []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + os.Getenv("HOME"),
		"USER=" + os.Getenv("USER"),
		"LANG=" + os.Getenv("LANG"),
		"TERM=dumb",
		"PAGER=cat",
	}

	// Capture stdout and stderr.
	var stdoutBuf, stderrBuf []byte
	cmd.Stdout = &limitedWriter{data: &stdoutBuf, max: maxOutputBytes}
	cmd.Stderr = &limitedWriter{data: &stderrBuf, max: maxOutputBytes}

	// Run the command.
	err := cmd.Run()

	// Extract exit code (0 if no error, or the actual exit code if command failed).
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else if execCtx.Err() == context.DeadlineExceeded {
			// Timeout: kill the process with SIGKILL after grace period.
			if cmd.Process != nil {
				_ = cmd.Process.Signal(syscall.SIGTERM)
				time.Sleep(100 * time.Millisecond)
				_ = cmd.Process.Kill()
			}
			return nil, fmt.Errorf("tool execution timed out")
		} else {
			// Other error (spawn failure, etc.).
			return nil, fmt.Errorf("command execution failed: %w", err)
		}
	}

	// Check if output was truncated.
	truncated := len(stdoutBuf)+len(stderrBuf) >= maxOutputBytes

	return &BashResponse{
		Stdout:     string(stdoutBuf),
		Stderr:     string(stderrBuf),
		ExitCode:   exitCode,
		Truncated:  truncated,
		WorkingDir: workingDir,
	}, nil
}

// limitedWriter writes to a byte slice with a size limit.
type limitedWriter struct {
	data    *[]byte
	max     int
	written int
}

func (lw *limitedWriter) Write(p []byte) (int, error) {
	remaining := lw.max - lw.written
	if remaining <= 0 {
		return len(p), nil // Silently drop data if limit reached.
	}

	if len(p) > remaining {
		*lw.data = append(*lw.data, p[:remaining]...)
		lw.written += remaining
		return len(p), nil
	}

	*lw.data = append(*lw.data, p...)
	lw.written += len(p)
	return len(p), nil
}
