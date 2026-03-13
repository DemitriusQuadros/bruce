# Spec 25: Google Docs Tool [BACKEND]

## Overview

Implement `docs_read` and `docs_create` tools using Google Docs API v1. `docs_read` fetches document content and metadata by document ID or URL. `docs_create` creates a new blank document with title, returns document ID and share link. Uses the same OAuth token infrastructure as Gmail (Spec 13). Both tools support text-only operations for Phase 2.

## Phase

**Phase 2** (Weeks 7–14)

## Prerequisites

- Tool registry exists (Spec 12)
- Google OAuth flow complete (Spec 13)
- Google Docs API enabled in OAuth scopes

## Deliverables

**Files to Create:**
- `internal/tools/google/docs.go` — Tool implementations (read + create)
- `internal/tools/google/docs_client.go` — Google Docs API wrapper

**Files to Modify:**
- `internal/tools/registry.go` — register both tools at startup
- `cmd/bruce/main.go` — instantiate Docs tool with OAuth client
- `config.example.yml` — document Docs tool section

## Acceptance Criteria

- [ ] Tool `docs_read` accepts: `document_id` (string)
- [ ] Tool `docs_create` accepts: `title` (string), `sharing` (optional: "private" or "shareable_link")
- [ ] `docs_read` fetches document via API, extracts plain text, returns title + body (max 10KB text)
- [ ] `docs_read` handles 404 (document not found), 403 (permission denied) gracefully
- [ ] `docs_create` creates new blank document, sets title, returns document ID and share link (if `sharing: "shareable_link"`)
- [ ] Tool handles OAuth token expiry by refreshing (Spec 13)
- [ ] Tool handles rate limits (429) with backoff (Spec 24)
- [ ] Latency: read <2s p95, create <3s p95
- [ ] Tool ignores formatting, comments, suggestions; returns clean text

## API / Component Contract

**`docs_read` Schema**:
```json
{
	"name": "docs_read",
	"description": "Read the content of a Google Doc",
	"input_schema": {
		"type": "object",
		"properties": {
			"document_id": {
				"type": "string",
				"description": "Google Docs document ID"
			}
		},
		"required": ["document_id"]
	}
}
```

**`docs_create` Schema**:
```json
{
	"name": "docs_create",
	"description": "Create a new Google Doc with a title",
	"input_schema": {
		"type": "object",
		"properties": {
			"title": {
				"type": "string",
				"description": "Title of the new document"
			},
			"sharing": {
				"type": "string",
				"enum": ["private", "shareable_link"],
				"description": "Sharing mode; 'private' = owner only, 'shareable_link' = get public link"
			}
		},
		"required": ["title"]
	}
}
```

## Out of Scope

- Update document content (only create + read for Phase 2)
- Comments, suggestions, revision history
- Image/embedding support
- Collaborative editing notifications
