# Spec 29: Trello Tool [BACKEND]

## Overview

Implement Trello API tools: `trello_list_cards`, `trello_move_card`, `trello_create_card` using Trello's REST API. Requires Trello API key and token. Supports querying board/list state, moving cards between lists, and creating new cards. Read operations are immediate; write operations require approval (Spec 20).

## Phase

**Phase 2** (Weeks 7–14)

## Prerequisites

- Tool registry exists (Spec 12)
- Confirmation system exists (Spec 20)
- Config system extended for Trello API credentials

## Deliverables

**Files to Create:**
- `internal/tools/trello/trello.go` — Tool implementations
- `internal/tools/trello/client.go` — Trello API wrapper

**Files to Modify:**
- `internal/tools/registry.go` — register Trello tools at startup
- `cmd/bruce/main.go` — instantiate Trello tool with API credentials
- `internal/config/config.go` — add `tools.trello.api_key`, `tools.trello.api_token`, `tools.trello.board_id`
- `config.example.yml` — document Trello API setup

## Acceptance Criteria

- [ ] Tool `trello_list_cards` accepts: `board_id` or `list_id`; returns cards with ID, name, description, list, labels
- [ ] Tool `trello_move_card` accepts: `card_id`, `target_list_id`; moves card; requires approval
- [ ] Tool `trello_create_card` accepts: `list_id`, `name`, `description` (optional), `labels` (optional); requires approval
- [ ] All tools handle 401 (invalid key/token), 404 (card not found), 429 (rate limit) gracefully
- [ ] Write operations create confirmations and wait for approval before API call
- [ ] Latency: read <2s p95, write <3s p95
- [ ] Config supports optional default `board_id` for convenience

## API / Component Contract

**Config**:
```yaml
tools:
  trello:
    enabled: true
    api_key: "..."
    api_token: "..."
    board_id: "optional_default_board_id"
```

**Tool Schemas**:
```json
{
	"name": "trello_list_cards",
	"description": "List cards on a Trello board or list",
	"input_schema": {
		"type": "object",
		"properties": {
			"board_id": { "type": "string" },
			"list_id": { "type": "string" }
		}
	}
},
{
	"name": "trello_move_card",
	"description": "Move a card to a different list (requires approval)",
	"input_schema": {
		"type": "object",
		"properties": {
			"card_id": { "type": "string" },
			"target_list_id": { "type": "string" }
		},
		"required": ["card_id", "target_list_id"]
	}
},
{
	"name": "trello_create_card",
	"description": "Create a new Trello card (requires approval)",
	"input_schema": {
		"type": "object",
		"properties": {
			"list_id": { "type": "string" },
			"name": { "type": "string" },
			"description": { "type": "string" },
			"labels": { "type": "array", "items": { "type": "string" } }
		},
		"required": ["list_id", "name"]
	}
}
```

## Out of Scope

- Checklist management
- Comment/attachment operations
- Board/list creation
- Power-Ups integration
