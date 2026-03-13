# Spec 14: Gmail Tool [BACKEND]

## Overview

Implement `gmail_read` tool that lists recent messages from the user's Gmail inbox and fetches full content of a specific thread. Uses Google Gmail API v1 with stored OAuth token (from Spec 13). Tool schema exposes `action` (list|fetch), `max_results`, `thread_id`. Responses include sender, subject, snippet, and full message body for fetch operations.

## Phase

**Phase 1** (Weeks 1–6)

## Prerequisites

- Tool registry exists (Spec 12)
- Google OAuth flow complete (Spec 13)
- Google Gmail API enabled in OAuth scopes

## Deliverables

**Files to Create:**
- `internal/tools/gmail/gmail.go` — Tool implementation
- `internal/tools/gmail/client.go` — Gmail API wrapper (list, fetch methods)

**Files to Modify:**
- `internal/tools/registry.go` — register Gmail tool at startup
- `cmd/bruce/main.go` — instantiate Gmail tool with OAuth client
- `config.example.yml` — document Gmail tool section

## Acceptance Criteria

- [ ] Tool name is `gmail_read` (registered in registry)
- [ ] Tool schema includes: `action` (enum: "list" | "fetch"), `max_results` (int, 1–20), `thread_id` (string, optional)
- [ ] `action=list` returns up to `max_results` messages from inbox with: sender, subject, snippet, timestamp
- [ ] `action=fetch` fetches full thread content by `thread_id` and returns all messages in order
- [ ] Tool gracefully handles 401 (token expired) by refreshing via Spec 13
- [ ] Tool gracefully handles 429 (rate limit) by returning error with retry guidance
- [ ] Tool gracefully handles network errors by returning error (worker retries via Spec 11)
- [ ] API response is formatted as JSON with clear labels (from, to, subject, body, date)
- [ ] Tool execution latency is <3s p95 for list (mocked) and <5s p95 for fetch (real API)

## API / Component Contract

**Tool Schema**:
```json
{
	"name": "gmail_read",
	"description": "List or fetch Gmail messages from your inbox",
	"input_schema": {
		"type": "object",
		"properties": {
			"action": {
				"type": "string",
				"enum": ["list", "fetch"],
				"description": "list: show recent messages; fetch: get full thread content"
			},
			"max_results": {
				"type": "integer",
				"minimum": 1,
				"maximum": 20,
				"description": "For list action: number of messages to return"
			},
			"thread_id": {
				"type": "string",
				"description": "For fetch action: thread ID to fetch full content"
			}
		},
		"required": ["action"]
	}
}
```

**`internal/tools/gmail/client.go`**:
```go
type Client struct {
	svc   *gmail.Service
	token *oauth2.Token
}

func (c *Client) ListMessages(ctx context.Context, maxResults int) ([]Message, error)
func (c *Client) FetchThread(ctx context.Context, threadID string) ([]Message, error)

type Message struct {
	ID        string
	ThreadID  string
	From      string
	To        string
	Subject   string
	Snippet   string
	Body      string
	Timestamp time.Time
}
```

## Out of Scope

- Gmail send/draft (Spec 21 handles send via `gmail_send` tool)
- Gmail label/filter management
- Gmail search with complex queries (simple inbox list for Phase 1)
- Multi-account Gmail support
