package bash

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateCommand_Blocklist(t *testing.T) {
	tests := []struct {
		name    string
		cmd     string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "rm -rf /",
			cmd:     "rm -rf /",
			wantErr: true,
			errMsg:  "rm -rf / is forbidden",
		},
		{
			name:    "rm -rf with path",
			cmd:     "rm -rf /important",
			wantErr: true,
			errMsg:  "rm -rf / is forbidden",
		},
		{
			name:    "pipe to bash",
			cmd:     "cat file | bash",
			wantErr: true,
			errMsg:  "pipe to shell is forbidden",
		},
		{
			name:    "pipe to sh",
			cmd:     "cat file | sh",
			wantErr: true,
			errMsg:  "pipe to shell is forbidden",
		},
		{
			name:    "eval injection",
			cmd:     "eval $(cat file)",
			wantErr: true,
			errMsg:  "eval injection is forbidden",
		},
		{
			name:    "backtick eval",
			cmd:     "eval `cat file`",
			wantErr: true,
			errMsg:  "eval injection is forbidden",
		},
		{
			name:    "device redirect stdout",
			cmd:     "echo test > /dev/sda",
			wantErr: true,
			errMsg:  "raw device redirect is forbidden",
		},
		{
			name:    "device redirect stdin",
			cmd:     "cat < /dev/zero",
			wantErr: true,
			errMsg:  "raw device redirect is forbidden",
		},
		{
			name:    "safe command",
			cmd:     "ls -la",
			wantErr: false,
		},
		{
			name:    "safe command with pipe to file",
			cmd:     "cat file | grep pattern > output.txt",
			wantErr: false,
		},
		{
			name:    "echo with special chars",
			cmd:     "echo 'hello world'",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCommand(tt.cmd, []string{})
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCommand() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if err.Error() != "validation failed: "+tt.errMsg {
					t.Errorf("ValidateCommand() error message = %q, want %q", err.Error(), "validation failed: "+tt.errMsg)
				}
			}
		})
	}
}

func TestValidateCommand_Allowlist(t *testing.T) {
	tests := []struct {
		name      string
		cmd       string
		allowlist []string
		wantErr   bool
	}{
		{
			name:      "command in allowlist",
			cmd:       "ls -la",
			allowlist: []string{"ls"},
			wantErr:   false,
		},
		{
			name:      "command not in allowlist",
			cmd:       "cat /etc/passwd",
			allowlist: []string{"ls", "echo"},
			wantErr:   true,
		},
		{
			name:      "path to command in allowlist",
			cmd:       "/bin/ls -la",
			allowlist: []string{"ls"},
			wantErr:   false,
		},
		{
			name:      "empty allowlist (no restriction)",
			cmd:       "rm -f file",
			allowlist: []string{},
			wantErr:   false, // No allowlist restriction, but blocklist still applies
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCommand(tt.cmd, tt.allowlist)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCommand() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateWorkingDir(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name    string
		baseDir string
		relDir  string
		wantErr bool
		checkFn func(string) bool // Optional check function instead of exact match
	}{
		{
			name:    "empty relative dir",
			baseDir: tmpDir,
			relDir:  "",
			wantErr: false,
			checkFn: func(got string) bool {
				// Should return something close to tmpDir (accounting for symlinks)
				return got != ""
			},
		},
		{
			name:    "relative subdir",
			baseDir: tmpDir,
			relDir:  "subdir",
			wantErr: false,
			checkFn: func(got string) bool {
				// Should end with "subdir"
				return filepath.Base(got) == "subdir"
			},
		},
		{
			name:    "path with ..",
			baseDir: tmpDir,
			relDir:  "../etc",
			wantErr: true,
		},
		{
			name:    "absolute path",
			baseDir: tmpDir,
			relDir:  "/etc",
			wantErr: true,
		},
		{
			name:    "nested relative path",
			baseDir: tmpDir,
			relDir:  "a/b/c",
			wantErr: false,
			checkFn: func(got string) bool {
				// Should end with "a/b/c" or "a\\b\\c" on Windows
				return filepath.Base(filepath.Dir(filepath.Dir(got))) == "a"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateWorkingDir(tt.baseDir, tt.relDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateWorkingDir() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && tt.checkFn != nil && !tt.checkFn(got) {
				t.Errorf("ValidateWorkingDir() = %q, check failed", got)
			}
		})
	}
}

func TestValidateWorkingDir_ComplexPath(t *testing.T) {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, "base")
	os.Mkdir(baseDir, 0o755)

	// Resolve baseDir to its canonical form (same as what ValidateWorkingDir does)
	baseDirResolved, _ := filepath.EvalSymlinks(baseDir)

	// Verify that a complex path with multiple components stays under base
	got, err := ValidateWorkingDir(baseDir, "a/b/c/d")
	if err != nil {
		t.Errorf("ValidateWorkingDir() error = %v, wantErr false", err)
	}

	// Check that the result is under baseDir (using canonical form)
	rel, err := filepath.Rel(baseDirResolved, got)
	if err != nil || strings.HasPrefix(rel, "..") {
		t.Errorf("ValidateWorkingDir() result escaped base directory: rel=%s, got=%s, base=%s", rel, got, baseDirResolved)
	}
}

func TestValidateCommand_EmptyCommand(t *testing.T) {
	err := ValidateCommand("", []string{})
	if err == nil {
		t.Error("ValidateCommand() should reject empty command")
	}
	if err.Error() != "validation failed: command is empty" {
		t.Errorf("ValidateCommand() error = %q, want 'validation failed: command is empty'", err.Error())
	}
}
