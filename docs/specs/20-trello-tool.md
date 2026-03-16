# Spec 28: Trello Tool [BACKEND]

## Overview

Implement `trello_board_list`, `trello_card_create`, `trello_card_move`, and `trello_card_get` tools that interact with Trello API. Requires OAuth via Spec 13 or Trello API key. Tools allow listing boards, creating cards with labels/due dates, moving cards between lists, and retrieving card details. Useful for task management workflows.

## Phase

**Phase 2** (Weeks 7–14)

## Prerequisites

- Tool registry exists (Spec 12)
- OAuth flow or API key management (Spec 13 or Spec 22)
- Trello API credentials configured

## Deliverables

**Files to Create:**
- `internal/tools/trello/trello.go` — Tool implementations
- `internal/tools/trello/client.go` — Trello API wrapper

**Files to Modify:**
- `internal/tools/registry.go` — register all tools
- `cmd/bruce/main.go` — instantiate Trello tool
- `config.example.yml` — document Trello API key configuration

## Acceptance Criteria

- [ ] Tool `trello_board_list` returns: list of boards (id, name, description)
- [ ] Tool `trello_card_create` accepts: `board_id`, `list_id`, `name`, optional: `description`, `due_date`, `labels`
- [ ] Tool `trello_card_move` accepts: `card_id`, `list_id` (new list)
- [ ] Tool `trello_card_get` accepts: `card_id`; returns: full card details (name, description, due date, labels, list, board)
- [ ] Tool handles Trello API authentication (API key + token)
- [ ] Tool validates board/list/card IDs before operations
- [ ] Tool handles 401 (auth failure) with clear error
- [ ] Tool handles 404 (not found) gracefully
- [ ] Tool handles 429 (rate limit) with retry logic (Spec 23)
- [ ] Latency: list <2s p95, create <3s p95, move <2s p95

## API / Component Contract

**`trello_board_list` Schema**:
```json
{
	"name": "trello_board_list",
	"description": "List all Trello boards",
	"input_schema": {
		"type": "object",
		"properties": {}
	}
}
```

**`trello_card_create` Schema**:
```json
{
	"name": "trello_card_create",
	"description": "Create a new card in a Trello list",
	"input_schema": {
		"type": "object",
		"properties": {
			"board_id": { "type": "string" },
			"list_id": { "type": "string" },
			"name": { "type": "string" },
			"description": { "type": "string" },
			"due_date": { "type": "string", "format": "date" },
			"labels": { "type": "array", "items": { "type": "string" } }
		},
		"required": ["board_id", "list_id", "name"]
	}
}
```

**Response** (for card_create):
```json
{
	"card_id": "abc123def456",
	"url": "https://trello.com/c/abc123def456",
	"created_at": "2024-03-13T14:02:00Z"
}
```

## Out of Scope

- Checklist management
- Comment operations
- Attachment operations
- Board/list creation
- Custom field support
- Power-up integration

