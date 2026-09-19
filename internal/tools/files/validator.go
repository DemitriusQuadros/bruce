package files

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// expandHome expands a leading ~ or ~/ in a path to the user's home directory.
func expandHome(path string) string {
	if path == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
	} else if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

// ValidatePath validates a user-supplied path against a base directory.
// It rejects absolute paths, any path containing "..", and symlinks that escape the base.
// Returns the canonical absolute path on success.
func ValidatePath(basePath, userPath string) (string, error) {
	// Expand home directory if specified in basePath
	basePath = expandHome(basePath)

	// Cleanly handle leading ~/ or ./ in userPath relative to configured basePath
	userPath = strings.TrimPrefix(userPath, "~/")
	userPath = strings.TrimPrefix(userPath, "./")

	if filepath.IsAbs(userPath) {
		return "", fmt.Errorf("validation failed: path must be relative, got %q", userPath)
	}

	if strings.Contains(userPath, "..") {
		return "", fmt.Errorf("validation failed: path contains .. (traversal not allowed)")
	}

	// Clean the base path.
	cleanBase := filepath.Clean(basePath)

	// Build and clean the full candidate path.
	fullPath := filepath.Clean(filepath.Join(cleanBase, userPath))

	// Ensure the result is still under basePath.
	rel, err := filepath.Rel(cleanBase, fullPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("validation failed: path escapes base directory")
	}

	// If the file already exists, resolve symlinks and re-verify.
	resolved, err := filepath.EvalSymlinks(fullPath)
	if err == nil {
		// File exists — check the real path is still under base.
		resolvedBase, err2 := filepath.EvalSymlinks(cleanBase)
		if err2 != nil {
			resolvedBase = cleanBase
		}
		rel2, err2 := filepath.Rel(resolvedBase, resolved)
		if err2 != nil || strings.HasPrefix(rel2, "..") {
			return "", fmt.Errorf("validation failed: symlink target escapes base directory")
		}
		return resolved, nil
	}

	return fullPath, nil
}
