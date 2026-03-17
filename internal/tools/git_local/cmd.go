// Package git_local provides a subprocess wrapper for local Git operations.
package git_local

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// StatusResult holds the parsed output of git status --porcelain=v1 -b.
type StatusResult struct {
	Branch    string
	Staged    []string
	Unstaged  []string
	Untracked []string
}

// Commander wraps git subprocess execution for a specific repository directory.
type Commander struct {
	repoPath string
	timeout  time.Duration
}

// NewCommander creates a new Commander for the given repository path with the specified timeout.
func NewCommander(repoPath string, timeout time.Duration) *Commander {
	return &Commander{
		repoPath: repoPath,
		timeout:  timeout,
	}
}

// runGit executes a git command with the given arguments in the repository directory.
// Returns combined stdout+stderr output as a string, or an error.
func (c *Commander) runGit(ctx context.Context, args ...string) (string, error) {
	execCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	cmd := exec.CommandContext(execCtx, "git", args...)
	cmd.Dir = c.repoPath
	cmd.Stdin = nil
	cmd.Env = []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + os.Getenv("HOME"),
		"GIT_TERMINAL_PROMPT=0",
	}

	var outBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &outBuf

	err := cmd.Run()
	output := outBuf.String()

	if err != nil {
		if execCtx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("git command timed out")
		}
		if _, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("git %s failed: %s", strings.Join(args, " "), strings.TrimSpace(output))
		}
		return "", fmt.Errorf("git command error: %w", err)
	}

	return strings.TrimRight(output, "\n"), nil
}

// Status runs git status --porcelain=v1 -b and returns a parsed StatusResult.
func (c *Commander) Status(ctx context.Context) (*StatusResult, error) {
	output, err := c.runGit(ctx, "status", "--porcelain=v1", "-b")
	if err != nil {
		return nil, err
	}

	result := &StatusResult{
		Staged:    []string{},
		Unstaged:  []string{},
		Untracked: []string{},
	}

	for _, line := range strings.Split(output, "\n") {
		if len(line) == 0 {
			continue
		}

		if strings.HasPrefix(line, "## ") {
			// Branch header: ## main...origin/main or ## HEAD (no branch)
			branchInfo := strings.TrimPrefix(line, "## ")
			// Strip tracking info (e.g. "main...origin/main" -> "main")
			if idx := strings.Index(branchInfo, "..."); idx != -1 {
				branchInfo = branchInfo[:idx]
			}
			// Handle "No commits yet on <branch>"
			if strings.HasPrefix(branchInfo, "No commits yet on ") {
				branchInfo = strings.TrimPrefix(branchInfo, "No commits yet on ")
			}
			result.Branch = branchInfo
			continue
		}

		if len(line) < 3 {
			continue
		}

		x := line[0] // index status
		y := line[1] // working tree status
		path := line[3:]

		switch {
		case x == '?' && y == '?':
			result.Untracked = append(result.Untracked, path)
		case x != ' ' && x != '?':
			result.Staged = append(result.Staged, path)
			if y != ' ' && y != '?' {
				result.Unstaged = append(result.Unstaged, path)
			}
		case y != ' ' && y != '?':
			result.Unstaged = append(result.Unstaged, path)
		}
	}

	return result, nil
}

// Commit runs git add -A followed by git commit -m {message} and returns the commit hash.
func (c *Commander) Commit(ctx context.Context, message string) (string, error) {
	if _, err := c.runGit(ctx, "add", "-A"); err != nil {
		return "", fmt.Errorf("git add -A: %w", err)
	}

	output, err := c.runGit(ctx, "commit", "-m", message)
	if err != nil {
		return "", err
	}

	// Extract commit hash from output like: [main abc1234] message
	hash := extractCommitHash(output)
	return hash, nil
}

// Push runs git push origin {branch}. If branch is empty, runs git push.
func (c *Commander) Push(ctx context.Context, branch string) error {
	var args []string
	if branch != "" {
		args = []string{"push", "origin", branch}
	} else {
		args = []string{"push"}
	}

	if _, err := c.runGit(ctx, args...); err != nil {
		return err
	}
	return nil
}

// Branch dispatches to the appropriate git branch subcommand based on action.
// Supported actions: list, create, delete.
// Returns a slice of branch names for "list", or the branch name for "create"/"delete".
func (c *Commander) Branch(ctx context.Context, action, name string) (interface{}, error) {
	switch action {
	case "list":
		output, err := c.runGit(ctx, "branch")
		if err != nil {
			return nil, err
		}
		var branches []string
		for _, line := range strings.Split(output, "\n") {
			line = strings.TrimSpace(line)
			line = strings.TrimPrefix(line, "* ")
			if line != "" {
				branches = append(branches, line)
			}
		}
		if branches == nil {
			branches = []string{}
		}
		return branches, nil

	case "create":
		if name == "" {
			return nil, fmt.Errorf("branch_name is required for action=create")
		}
		if _, err := c.runGit(ctx, "checkout", "-b", name); err != nil {
			return nil, err
		}
		return name, nil

	case "delete":
		if name == "" {
			return nil, fmt.Errorf("branch_name is required for action=delete")
		}
		if _, err := c.runGit(ctx, "branch", "-d", name); err != nil {
			return nil, err
		}
		return name, nil

	default:
		return nil, fmt.Errorf("unsupported action %q — must be one of: list, create, delete", action)
	}
}

// extractCommitHash pulls the abbreviated commit hash from git commit output.
// git commit output: [branch abc1234] commit message
var commitHashRe = regexp.MustCompile(`\[[\w/.-]+ ([0-9a-f]+)\]`)

func extractCommitHash(output string) string {
	matches := commitHashRe.FindStringSubmatch(output)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

// validateRepoPath validates and resolves a relative repository path against homeDir.
// Rejects paths containing ".." and absolute paths.
// Returns the resolved absolute path or an error if the path escapes homeDir.
func validateRepoPath(homeDir, repoPath string) (string, error) {
	if strings.Contains(repoPath, "..") {
		return "", fmt.Errorf("validation failed: repo_path contains .. (path traversal)")
	}

	if filepath.IsAbs(repoPath) {
		return "", fmt.Errorf("validation failed: repo_path must be relative")
	}

	// Resolve the home directory to eliminate symlinks.
	homeDirResolved, err := filepath.EvalSymlinks(homeDir)
	if err != nil {
		homeDirResolved = filepath.Clean(homeDir)
	}

	// Build the full path by joining resolved home dir with relative path.
	fullPath := filepath.Clean(filepath.Join(homeDirResolved, repoPath))

	// Verify the resulting path is still under homeDir.
	rel, err := filepath.Rel(homeDirResolved, fullPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("validation failed: repo_path escapes home directory")
	}

	return fullPath, nil
}
