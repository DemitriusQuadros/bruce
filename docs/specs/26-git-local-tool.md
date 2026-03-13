# Spec 26: GitHub Tool [BACKEND]

## Overview

Implement GitHub API tools: `github_list_branches`, `github_list_prs`, `github_list_issues`, `github_create_issue` for repository management. Uses GitHub REST API v3 with OAuth token (scope: `repo`, `read:user`). Supports single owner/repo or user-configured repository from config. Phase 2 adds reads; Phase 2+ adds writes. Issue creation requires approval (Spec 20).

## Phase

**Phase 2** (Weeks 7–14)

## Prerequisites

- Tool registry exists (Spec 12)
- GitHub OAuth flow (separate from Spec 13, but uses same pattern)
- Confirmation system exists (Spec 20)

## Deliverables

**Files to Create:**
- `internal/tools/github/github.go` — Tool implementations
- `internal/tools/github/client.go` — GitHub API wrapper
- `internal/auth/github.go` — GitHub OAuth flow (similar to Spec 13)

**Files to Modify:**
- `internal/api/router.go` — add `/auth/github/start`, `/auth/github/callback` routes
- `internal/tools/registry.go` — register GitHub tools at startup
- `cmd/bruce/main.go` — instantiate GitHub tool with OAuth client
- `config.example.yml` — document GitHub repository config and OAuth setup
- `internal/database/schema.sql` — ensure `oauth_tokens` table supports GitHub provider

## Acceptance Criteria

- [ ] Tool `github_list_branches` accepts: `owner`, `repo`, `per_page` (optional); returns branch names and protection status
- [ ] Tool `github_list_issues` accepts: `owner`, `repo`, `state` (open|closed|all), `per_page` (optional); returns issues with title, state, labels, number
- [ ] Tool `github_list_prs` accepts: `owner`, `repo`, `state` (open|closed|all); returns PR title, state, base/head branches
- [ ] Tool `github_create_issue` requires approval; accepts: `owner`, `repo`, `title`, `body` (optional), `labels` (optional array); returns issue number and URL
- [ ] All tools handle 401 (expired token) by refreshing OAuth
- [ ] All tools handle 404 (repo not found), 403 (permission denied) gracefully
- [ ] Rate limit (429) errors include reset time in response
- [ ] Latency: list ops <2s p95, create <3s p95
- [ ] `owner` and `repo` can be omitted if set in config as defaults

## API / Component Contract

**Tool Schemas**:
```json
{
	"name": "github_list_branches",
	"input_schema": {
		"type": "object",
		"properties": {
			"owner": { "type": "string", "description": "Repository owner (optional if config default set)" },
			"repo": { "type": "string", "description": "Repository name" },
			"per_page": { "type": "integer", "minimum": 1, "maximum": 100 }
		},
		"required": ["repo"]
	}
},
{
	"name": "github_list_issues",
	"input_schema": {
		"type": "object",
		"properties": {
			"owner": { "type": "string" },
			"repo": { "type": "string" },
			"state": { "type": "string", "enum": ["open", "closed", "all"] },
			"per_page": { "type": "integer" }
		},
		"required": ["repo"]
	}
},
{
	"name": "github_create_issue",
	"description": "Create a new GitHub issue (requires approval)",
	"input_schema": {
		"type": "object",
		"properties": {
			"owner": { "type": "string" },
			"repo": { "type": "string" },
			"title": { "type": "string" },
			"body": { "type": "string" },
			"labels": { "type": "array", "items": { "type": "string" } }
		},
		"required": ["repo", "title"]
	}
}
```

## Out of Scope

- Pull request creation (Phase 2+, write ops require maturation)
- PR merge/approval (Phase 3+)
- GitHub Actions workflow management
- Branch protection rule modification
- GitLab support (GitHub only for Phase 2)
