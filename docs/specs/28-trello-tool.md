# Spec 28: Notion Tool [BACKEND]

## Overview

Implement Notion API tools: `notion_read` (query pages/databases), `notion_create` (create pages), `notion_update` (update page properties) using Notion's official REST API. Requires Notion integration token (from user's Notion workspace). Phase 2 adds read + basic write; write operations require approval (Spec 20). Supports querying databases by ID and creating/updating child pages.

## Phase

**Phase 2** (Weeks 7–14)

## Prerequisites

- Tool registry exists (Spec 12)
- Confirmation system exists (Spec 20)
- Config system extended for Notion API key

## Deliverables

**Files to Create:**
- `internal/tools/notion/notion.go` — Tool implementations (read, create, update)
- `internal/tools/notion/client.go` — Notion API wrapper

**Files to Modify:**
- `internal/tools/registry.go` — register Notion tools at startup
- `cmd/bruce/main.go` — instantiate Notion tool with API token
- `internal/config/config.go` — add `tools.notion.api_token`, `tools.notion.enabled`
- `config.example.yml` — document Notion API token setup
- `internal/database/schema.sql` — `oauth_tokens` table already supports arbitrary provider names

## Acceptance Criteria

- [ ] Tool `notion_read` accepts: `page_id` or `database_id`; returns page/block content or database records (paginated)
- [ ] Tool `notion_create` accepts: `parent_id`, `title`, `properties` (optional JSON); creates child page; requires approval
- [ ] Tool `notion_update` accepts: `page_id`, `properties` (JSON patch); updates page properties; requires approval
- [ ] All tools handle 401 (invalid token), 404 (page not found), 429 (rate limit) gracefully
- [ ] Read operations auto-retry on 429 with backoff (Spec 24)
- [ ] Write operations are queued in confirmation system before execution
- [ ] Latency: read <2s p95, write <3s p95

## API / Component Contract

**Config**:
```yaml
tools:
  notion:
    enabled: true
    api_token: "secret_..."
```

**Tool Schemas**:
```json
{
	"name": "notion_read",
	"description": "Query a Notion page or database",
	"input_schema": {
		"type": "object",
		"properties": {
			"page_id": {
				"type": "string",
				"description": "Notion page or database ID (UUID format)"
			},
			"query": {
				"type": "string",
				"description": "Optional filter for database queries (Notion filter syntax)"
			}
		},
		"required": ["page_id"]
	}
},
{
	"name": "notion_create",
	"description": "Create a new Notion page (requires approval)",
	"input_schema": {
		"type": "object",
		"properties": {
			"parent_id": { "type": "string", "description": "Parent page or database ID" },
			"title": { "type": "string", "description": "Page title" },
			"properties": {
				"type": "object",
				"description": "Page properties (database-specific schema)"
			}
		},
		"required": ["parent_id", "title"]
	}
}
```

## Out of Scope

- Advanced Notion filtering (complex queries beyond simple equals/contains)
- Workspace-level operations (only page/database level)
- Notion Synced Databases
- Bulk operations (single page CRUD only)
