# Spec 27: Git Local Tool [BACKEND]

## Overview

Implement local Git tools that execute `git` commands on user-configured repository paths: `git_status`, `git_log`, `git_branch`. All commands are read-only for Phase 2. Executes shell `git` command via `os/exec` with timeout (10s default). Repository paths are configured in `config.yml` under `tools.git.repos` (map of friendly names to absolute paths). Prevents path traversal via validation.

## Phase

**Phase 2** (Weeks 7–14)

## Prerequisites

- Tool registry exists (Spec 12)
- Config system supports tool-specific config
- Shell execution is available in container/environment

## Deliverables

**Files to Create:**
- `internal/tools/git/git.go` — Git tool implementations (status, log, branch)
- `internal/tools/git/executor.go` — Git command executor with path validation

**Files to Modify:**
- `internal/tools/registry.go` — register Git tools at startup
- `cmd/bruce/main.go` — instantiate Git tool with configured repos
- `config.example.yml` — document `tools.git.repos` config section

## Acceptance Criteria

- [ ] Tool `git_status` accepts: `repo` (friendly name from config), returns current branch, uncommitted changes summary, untracked files count
- [ ] Tool `git_log` accepts: `repo`, `max_count` (optional, default 10); returns last N commits with hash, author, date, message (first 200 chars)
- [ ] Tool `git_branch` accepts: `repo`; returns list of branches with `*` marking current branch
- [ ] Repository paths are validated (must match configured path exactly; no symlink escapes)
- [ ] All commands timeout after 10s
- [ ] Command fails gracefully if repository not found or is not a Git repo
- [ ] `git` command is escaped to prevent shell injection (use `os/exec` array args, not shell string)
- [ ] Latency: <500ms p95 for status/branch, <1s p95 for log (mocked)
- [ ] Config includes list of allowed repos: `tools.git.repos: { "project-a": "/home/user/projects/project-a", "project-b": "/home/user/projects/project-b" }`

## API / Component Contract

**Config**:
```yaml
tools:
  git:
    repos:
      project-a: "/home/user/Projects/project-a"
      project-b: "/home/user/Projects/project-b"
```

**Tool Schemas**:
```json
{
	"name": "git_status",
	"input_schema": {
		"type": "object",
		"properties": {
			"repo": {
				"type": "string",
				"description": "Repository name from config (e.g., 'project-a')"
			}
		},
		"required": ["repo"]
	}
},
{
	"name": "git_log",
	"input_schema": {
		"type": "object",
		"properties": {
			"repo": { "type": "string" },
			"max_count": { "type": "integer", "minimum": 1, "default": 10 }
		},
		"required": ["repo"]
	}
},
{
	"name": "git_branch",
	"input_schema": {
		"type": "object",
		"properties": {
			"repo": { "type": "string" }
		},
		"required": ["repo"]
	}
}
```

**`internal/tools/git/executor.go`**:
```go
type Executor struct {
	repos map[string]string // friendly name -> absolute path
}

func (e *Executor) ValidatePath(repoName string) (string, error)
func (e *Executor) Status(ctx context.Context, repoPath string) (string, error)
func (e *Executor) Log(ctx context.Context, repoPath string, maxCount int) (string, error)
```

## Out of Scope

- Write operations (commit, push, branch creation — Phase 2+ with approval)
- Interactive rebase / merge conflict resolution
- Stash, bisect, blame operations
- Multiple remotes
