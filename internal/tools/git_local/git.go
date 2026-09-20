// Package git_local provides tools.Tool implementations for local Git operations via CLI subprocess.
package git_local

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"bruce/internal/ai"
	"bruce/internal/config"
)

// resolveConfig resolves the home directory and command timeout from *config.Config.
func resolveConfig(cfg *config.Config) (string, time.Duration) {
	home := ""
	if cfg != nil && cfg.Tools.GitLocal.HomeDir != "" {
		home = cfg.Tools.GitLocal.HomeDir
	}
	if home == "" {
		home = os.Getenv("HOME")
	}

	timeout := 30 * time.Second
	if cfg != nil && cfg.Tools.GitLocal.TimeoutSeconds > 0 {
		timeout = time.Duration(cfg.Tools.GitLocal.TimeoutSeconds) * time.Second
	}
	return home, timeout
}

// StatusResponse is the JSON response returned by the git_status tool.
type StatusResponse struct {
	RepoPath  string   `json:"repo_path"`
	Branch    string   `json:"branch"`
	Staged    []string `json:"staged"`
	Unstaged  []string `json:"unstaged"`
	Untracked []string `json:"untracked"`
}

// CommitResponse is the JSON response returned by the git_commit tool.
type CommitResponse struct {
	RepoPath    string `json:"repo_path"`
	Hash        string `json:"hash"`
	Message     string `json:"message"`
	CommittedAt string `json:"committed_at"` // RFC3339
}

// PushResponse is the JSON response returned by the git_push tool.
type PushResponse struct {
	RepoPath string `json:"repo_path"`
	Branch   string `json:"branch"`
	PushedAt string `json:"pushed_at"` // RFC3339
}

// BranchResponse is the JSON response returned by the git_branch tool.
type BranchResponse struct {
	RepoPath   string   `json:"repo_path"`
	Action     string   `json:"action"`
	Branches   []string `json:"branches,omitempty"`
	BranchName string   `json:"branch_name,omitempty"`
}

// GitStatusTool implements tools.Tool for running git status.
type GitStatusTool struct {
	cfg *config.Config
}

// NewStatusTool creates a new GitStatusTool.
func NewStatusTool(cfg *config.Config) *GitStatusTool {
	return &GitStatusTool{cfg: cfg}
}

// Name returns the tool name.
func (t *GitStatusTool) Name() string { return "git_status" }

// Definition returns the AI tool definition for git_status.
func (t *GitStatusTool) Definition() ai.ToolDefinition {
	return ai.ToolDefinition{
		Name:        "git_status",
		Description: "Get the git status of a local repository (branch, staged, unstaged, untracked files)",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"repo_path": map[string]interface{}{
					"type":        "string",
					"description": "Relative or absolute path to the git repository (defaults to \".\" for configured home_dir)",
				},
			},
		},
	}
}

// Execute runs git status on the given repository and returns the result as JSON.
func (t *GitStatusTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	log.Printf("[git_local] git_status: Execute called")

	repoPathRaw, _ := input["repo_path"].(string)
	if repoPathRaw == "" {
		repoPathRaw = "."
	}

	homeDir, timeout := resolveConfig(t.cfg)
	resolvedPath, err := validateRepoPath(homeDir, repoPathRaw)
	if err != nil {
		return nil, err
	}

	log.Printf("[git_local] git_status: repo=%q", resolvedPath)

	cmd := NewCommander(resolvedPath, timeout)
	status, err := cmd.Status(ctx)
	if err != nil {
		log.Printf("[git_local] git_status: Status failed: %v", err)
		return nil, fmt.Errorf("git_status: %w", err)
	}

	resp := StatusResponse{
		RepoPath:  repoPathRaw,
		Branch:    status.Branch,
		Staged:    status.Staged,
		Unstaged:  status.Unstaged,
		Untracked: status.Untracked,
	}
	log.Printf("[git_local] git_status: OK — branch=%q staged=%d unstaged=%d untracked=%d",
		resp.Branch, len(resp.Staged), len(resp.Unstaged), len(resp.Untracked))

	data, err := json.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("git_status: marshal response: %w", err)
	}
	return string(data), nil
}

// GitCommitTool implements tools.Tool for staging all changes and committing.
type GitCommitTool struct {
	cfg *config.Config
}

// NewCommitTool creates a new GitCommitTool.
func NewCommitTool(cfg *config.Config) *GitCommitTool {
	return &GitCommitTool{cfg: cfg}
}

// Name returns the tool name.
func (t *GitCommitTool) Name() string { return "git_commit" }

// Definition returns the AI tool definition for git_commit.
func (t *GitCommitTool) Definition() ai.ToolDefinition {
	return ai.ToolDefinition{
		Name:        "git_commit",
		Description: "Stage all changes (git add -A) and create a commit in a local repository",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"repo_path": map[string]interface{}{
					"type":        "string",
					"description": "Relative or absolute path to the git repository (defaults to \".\" for configured home_dir)",
				},
				"message": map[string]interface{}{
					"type":        "string",
					"description": "Commit message",
				},
			},
			"required": []string{"message"},
		},
	}
}

// Execute stages all changes and creates a commit, returning the commit hash as JSON.
func (t *GitCommitTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	log.Printf("[git_local] git_commit: Execute called")

	repoPathRaw, _ := input["repo_path"].(string)
	if repoPathRaw == "" {
		repoPathRaw = "."
	}

	message, ok := input["message"].(string)
	if !ok || message == "" {
		return nil, fmt.Errorf("message is required and must be a non-empty string")
	}

	homeDir, timeout := resolveConfig(t.cfg)
	resolvedPath, err := validateRepoPath(homeDir, repoPathRaw)
	if err != nil {
		return nil, err
	}

	log.Printf("[git_local] git_commit: repo=%q message=%q", resolvedPath, message)

	cmd := NewCommander(resolvedPath, timeout)
	hash, err := cmd.Commit(ctx, message)
	if err != nil {
		log.Printf("[git_local] git_commit: Commit failed: %v", err)
		return nil, fmt.Errorf("git_commit: %w", err)
	}

	resp := CommitResponse{
		RepoPath:    repoPathRaw,
		Hash:        hash,
		Message:     message,
		CommittedAt: time.Now().UTC().Format(time.RFC3339),
	}
	log.Printf("[git_local] git_commit: OK — hash=%q", hash)

	data, err := json.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("git_commit: marshal response: %w", err)
	}
	return string(data), nil
}

// GitPushTool implements tools.Tool for pushing commits to a remote.
type GitPushTool struct {
	cfg *config.Config
}

// NewPushTool creates a new GitPushTool.
func NewPushTool(cfg *config.Config) *GitPushTool {
	return &GitPushTool{cfg: cfg}
}

// Name returns the tool name.
func (t *GitPushTool) Name() string { return "git_push" }

// Definition returns the AI tool definition for git_push.
func (t *GitPushTool) Definition() ai.ToolDefinition {
	return ai.ToolDefinition{
		Name:        "git_push",
		Description: "Push commits from a local repository to its remote origin",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"repo_path": map[string]interface{}{
					"type":        "string",
					"description": "Relative or absolute path to the git repository (defaults to \".\" for configured home_dir)",
				},
				"branch": map[string]interface{}{
					"type":        "string",
					"description": "Branch to push (e.g. main). If omitted, runs git push with no branch argument.",
				},
			},
		},
	}
}

// Execute pushes commits to the remote origin and returns confirmation as JSON.
func (t *GitPushTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	log.Printf("[git_local] git_push: Execute called")

	repoPathRaw, _ := input["repo_path"].(string)
	if repoPathRaw == "" {
		repoPathRaw = "."
	}

	branch, _ := input["branch"].(string)

	homeDir, timeout := resolveConfig(t.cfg)
	resolvedPath, err := validateRepoPath(homeDir, repoPathRaw)
	if err != nil {
		return nil, err
	}

	log.Printf("[git_local] git_push: repo=%q branch=%q", resolvedPath, branch)

	cmd := NewCommander(resolvedPath, timeout)
	if err := cmd.Push(ctx, branch); err != nil {
		log.Printf("[git_local] git_push: Push failed: %v", err)
		return nil, fmt.Errorf("git_push: %w", err)
	}

	resp := PushResponse{
		RepoPath: repoPathRaw,
		Branch:   branch,
		PushedAt: time.Now().UTC().Format(time.RFC3339),
	}
	log.Printf("[git_local] git_push: OK")

	data, err := json.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("git_push: marshal response: %w", err)
	}
	return string(data), nil
}

// GitBranchTool implements tools.Tool for branch management operations.
type GitBranchTool struct {
	cfg *config.Config
}

// NewBranchTool creates a new GitBranchTool.
func NewBranchTool(cfg *config.Config) *GitBranchTool {
	return &GitBranchTool{cfg: cfg}
}

// Name returns the tool name.
func (t *GitBranchTool) Name() string { return "git_branch" }

// Definition returns the AI tool definition for git_branch.
func (t *GitBranchTool) Definition() ai.ToolDefinition {
	return ai.ToolDefinition{
		Name:        "git_branch",
		Description: "Manage branches in a local git repository (list, create, or delete)",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"repo_path": map[string]interface{}{
					"type":        "string",
					"description": "Relative or absolute path to the git repository (defaults to \".\" for configured home_dir)",
				},
				"action": map[string]interface{}{
					"type":        "string",
					"description": "Branch action to perform: list, create, or delete",
					"enum":        []string{"list", "create", "delete"},
				},
				"branch_name": map[string]interface{}{
					"type":        "string",
					"description": "Branch name (required for create and delete actions)",
				},
			},
			"required": []string{"action"},
		},
	}
}

// Execute performs the requested branch operation and returns the result as JSON.
func (t *GitBranchTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	log.Printf("[git_local] git_branch: Execute called")

	repoPathRaw, _ := input["repo_path"].(string)
	if repoPathRaw == "" {
		repoPathRaw = "."
	}

	action, ok := input["action"].(string)
	if !ok || action == "" {
		return nil, fmt.Errorf("action is required and must be one of: list, create, delete")
	}

	branchName, _ := input["branch_name"].(string)

	homeDir, timeout := resolveConfig(t.cfg)
	resolvedPath, err := validateRepoPath(homeDir, repoPathRaw)
	if err != nil {
		return nil, err
	}

	log.Printf("[git_local] git_branch: repo=%q action=%q branchName=%q", resolvedPath, action, branchName)

	cmd := NewCommander(resolvedPath, timeout)
	result, err := cmd.Branch(ctx, action, branchName)
	if err != nil {
		log.Printf("[git_local] git_branch: Branch failed: %v", err)
		return nil, fmt.Errorf("git_branch: %w", err)
	}

	resp := BranchResponse{
		RepoPath:   repoPathRaw,
		Action:     action,
		BranchName: branchName,
	}

	switch v := result.(type) {
	case []string:
		resp.Branches = v
	case string:
		resp.BranchName = v
	}

	log.Printf("[git_local] git_branch: OK — action=%q", action)

	data, err := json.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("git_branch: marshal response: %w", err)
	}
	return string(data), nil
}
