# Spec 26: Git Local Tool [BACKEND]

## Overview

Implement `git_status`, `git_commit`, `git_push`, and `git_branch` tools for local Git repository operations. Tools work on the local filesystem (no GitHub API required, unlike Spec 25). Accept repo path (relative to home dir), validate with path restrictions. Useful for automating commits, branch management, and status checks during agent workflows.

## Phase

**Phase 1** (Weeks 1–6)

## Prerequisites

- Tool registry exists (Spec 12)
- File I/O tool validation patterns exist (Spec 17)
- Git CLI available in environment (git binary)

## Deliverables

**Files to Create:**
- `internal/tools/git_local/git.go` — Tool implementations (status, commit, push, branch)
- `internal/tools/git_local/cmd.go` — subprocess wrapper for git CLI

**Files to Modify:**
- `internal/tools/registry.go` — register all four tools
- `cmd/bruce/main.go` — instantiate Git tool
- `config.example.yml` — document Git tool section (allowed repo paths)

## Acceptance Criteria

- [ ] Tool `git_status` accepts: `repo_path` (relative to home)
- [ ] Tool `git_commit` accepts: `repo_path`, `message` (string)
- [ ] Tool `git_push` accepts: `repo_path`, `branch` (optional, defaults to current)
- [ ] Tool `git_branch` accepts: `repo_path`, `action` (list/create/delete), `branch_name`
- [ ] `git_status` returns: branch name, staged/unstaged changes, untracked files
- [ ] `git_commit` stages all changes and commits; returns commit hash
- [ ] `git_push` pushes to remote; handles auth (assumes SSH keys configured)
- [ ] `git_branch` creates/deletes/lists branches; validates branch names
- [ ] Path validation: reject paths containing `..`, outside home dir, or symlinks
- [ ] Command execution timeout: 30 seconds
- [ ] Stdout/stderr from git commands are captured and returned in response
- [ ] Handles git errors (merge conflicts, auth failures) with clear messages
- [ ] All operations are logged (repo path, command, success/failure)
- [ ] Latency: p95 <5s (varies by repo size)

## API / Component Contract

**`git_status` Schema**:
```json
{
	"name": "git_status",
	"description": "Show Git status of local repository",
	"input_schema": {
		"type": "object",
		"properties": {
			"repo_path": {
				"type": "string",
				"description": "Repository path relative to home directory"
			}
		},
		"required": ["repo_path"]
	}
}
```

**`git_commit` Schema**:
```json
{
	"name": "git_commit",
	"description": "Stage all changes and commit with message",
	"input_schema": {
		"type": "object",
		"properties": {
			"repo_path": { "type": "string" },
			"message": { "type": "string", "description": "Commit message" }
		},
		"required": ["repo_path", "message"]
	}
}
```

**`git_push` Schema**:
```json
{
	"name": "git_push",
	"description": "Push commits to remote repository",
	"input_schema": {
		"type": "object",
		"properties": {
			"repo_path": { "type": "string" },
			"branch": { "type": "string", "description": "Branch to push (default: current branch)" }
		},
		"required": ["repo_path"]
	}
}
```

**`git_branch` Schema**:
```json
{
	"name": "git_branch",
	"description": "Manage Git branches (list, create, delete)",
	"input_schema": {
		"type": "object",
		"properties": {
			"repo_path": { "type": "string" },
			"action": {
				"type": "string",
				"enum": ["list", "create", "delete"],
				"description": "Branch operation"
			},
			"branch_name": {
				"type": "string",
				"description": "Branch name (required for create/delete)"
			}
		},
		"required": ["repo_path", "action"]
	}
}
```

**Response Format**:
```json
{
	"repo_path": "my-project",
	"branch": "main",
	"status": "On branch main\nnothing to commit, working tree clean",
	"staged": [],
	"unstaged": ["file.txt"],
	"untracked": []
}
```

**`internal/tools/git_local/cmd.go`**:
```go
type Commander struct {
	repoPath string
	timeout  time.Duration
}

func (c *Commander) Status(ctx context.Context) (*StatusResult, error)
func (c *Commander) Commit(ctx context.Context, message string) (string, error) // returns commit hash
func (c *Commander) Push(ctx context.Context, branch string) error
func (c *Commander) Branch(ctx context.Context, action, name string) (interface{}, error)

func (c *Commander) runGit(ctx context.Context, args ...string) (string, error) {
	// Execute git CLI with args
	// Timeout after c.timeout
	// Return stdout or error
}
```

## Out of Scope

- Merge/rebase operations (too complex for Phase 1)
- Cherry-pick or patch operations
- Stash management
- Submodule operations
- GPG signing enforcement
- GitHub-specific operations (see Spec 25 for API-based GitHub)

