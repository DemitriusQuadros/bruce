package bash

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// ValidateCommand checks if the command is allowed based on allowlist and blocklist.
// Returns an error if the command is blocked.
func ValidateCommand(cmd string, allowlist []string) error {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return fmt.Errorf("validation failed: command is empty")
	}

	// Check blocklist (always active).
	if err := checkBlocklist(cmd); err != nil {
		return err
	}

	// Check allowlist (only if non-empty).
	if len(allowlist) > 0 {
		if err := checkAllowlist(cmd, allowlist); err != nil {
			return err
		}
	}

	return nil
}

// checkBlocklist rejects dangerous patterns.
func checkBlocklist(cmd string) error {
	// Patterns that are always blocked:
	blockedPatterns := []struct {
		pattern string
		reason  string
	}{
		// Destructive operations
		{pattern: `rm\s+-[rf]*f.*/$`, reason: "rm -rf / is forbidden"},
		{pattern: `rm\s+-[rf]*f\s+/`, reason: "rm -rf / is forbidden"},

		// Fork bombs
		{pattern: `:\(\)\s*{\s*:\|:\&`, reason: "fork bomb detected"},
		{pattern: `\(\)\s*{\s*\(|\.\&`, reason: "fork bomb detected"},

		// Pipe to shell
		{pattern: `\|\s*bash`, reason: "pipe to shell is forbidden"},
		{pattern: `\|\s*sh`, reason: "pipe to shell is forbidden"},
		{pattern: `\|bash`, reason: "pipe to shell is forbidden"},
		{pattern: `\|sh`, reason: "pipe to shell is forbidden"},

		// Device redirects
		{pattern: `>\s*/dev/`, reason: "raw device redirect is forbidden"},
		{pattern: `<\s*/dev/`, reason: "raw device redirect is forbidden"},

		// Eval injection
		{pattern: `eval\s*\$\(`, reason: "eval injection is forbidden"},
		{pattern: `eval\s*` + "`", reason: "eval injection is forbidden"},
	}

	for _, bp := range blockedPatterns {
		re := regexp.MustCompile(bp.pattern)
		if re.MatchString(cmd) {
			return fmt.Errorf("validation failed: %s", bp.reason)
		}
	}

	return nil
}

// checkAllowlist ensures the binary name is in the allowed list.
func checkAllowlist(cmd string, allowlist []string) error {
	// Extract the binary name (first word).
	fields := strings.Fields(cmd)
	if len(fields) == 0 {
		return fmt.Errorf("validation failed: no command found")
	}

	binary := filepath.Base(fields[0]) // Get just the binary name, not the path.

	// Check if binary is in allowlist.
	for _, allowed := range allowlist {
		if binary == allowed || binary == filepath.Base(allowed) {
			return nil
		}
	}

	return fmt.Errorf("validation failed: command %q not in allowed list", binary)
}

// ValidateWorkingDir validates the working directory and returns the absolute path.
func ValidateWorkingDir(baseDir, relDir string) (string, error) {
	if relDir == "" {
		// No relative dir specified; use base dir.
		return baseDir, nil
	}

	// Reject paths containing ..
	if strings.Contains(relDir, "..") {
		return "", fmt.Errorf("validation failed: working_dir contains .. (path traversal)")
	}

	// Reject absolute paths.
	if filepath.IsAbs(relDir) {
		return "", fmt.Errorf("validation failed: working_dir must be relative")
	}

	// Resolve the base directory to eliminate symlinks.
	baseDirResolved, err := filepath.EvalSymlinks(baseDir)
	if err != nil {
		// Base dir doesn't exist or can't be evaluated; use as-is
		baseDirResolved = filepath.Clean(baseDir)
	}

	// Build the full path by joining resolved base with relative path.
	fullPath := filepath.Clean(filepath.Join(baseDirResolved, relDir))

	// Verify that the resulting path is still under the base directory.
	rel, err := filepath.Rel(baseDirResolved, fullPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("validation failed: working_dir escapes base directory")
	}

	return fullPath, nil
}
