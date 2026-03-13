# Spec 14: Gmail Tool [BACKEND]

## Overview

Implement `email_read`, `email_send`, and `email_search` tools that interact with Gmail via Google Workspace API. All tools use OAuth tokens from Spec 13. `email_read` retrieves a single email by ID; `email_search` finds emails by subject/from/to; `email_send` composes and sends an email. Tools validate recipient addresses and prevent sending to untrusted domains.

## Phase

**Phase 1** (Weeks 1–6)

## Prerequisites

- Tool registry exists (Spec 12)
- Google OAuth flow complete (Spec 13)
- Gmail API enabled in OAuth scopes

## Deliverables

**Files to Create:**
- `internal/tools/gmail/gmail.go` — Tool implementations (read, send, search)
- `internal/tools/gmail/client.go` — Gmail API wrapper using google.golang.org/api/gmail/v1

**Files to Modify:**
- `internal/tools/registry.go` — register all three tools at startup
- `cmd/bruce/main.go` — instantiate Gmail tool with OAuth client
- `config.example.yml` — document Gmail tool section

## Acceptance Criteria

- [ ] Tool `email_read` exists; accepts: `message_id` (string)
- [ ] Tool `email_search` exists; accepts: `query` (string, Gmail search syntax)
- [ ] Tool `email_send` exists; accepts: `to`, `subject`, `body` (text or HTML); optional `cc`, `bcc`
- [ ] `email_read` returns: message ID, from, to, subject, body (plain text), sent timestamp
- [ ] `email_search` returns: list of message IDs, subjects, from addresses (max 10 results)
- [ ] `email_send` validates recipient addresses (rejects invalid formats)
- [ ] `email_send` returns: sent message ID, timestamp, confirmation
- [ ] Tool handles 401 (expired token) by refreshing via Spec 13
- [ ] Tool handles 429 (rate limit) gracefully with clear error
- [ ] All three tools are thread-safe (use mutex for token refresh if needed)
- [ ] Latency: read <2s p95, search <3s p95, send <5s p95

## API / Component Contract

**`email_read` Schema**:
```json
{
	"name": "email_read",
	"description": "Read a specific email message",
	"input_schema": {
		"type": "object",
		"properties": {
			"message_id": {
				"type": "string",
				"description": "Gmail message ID"
			}
		},
		"required": ["message_id"]
	}
}
```

**`email_search` Schema**:
```json
{
	"name": "email_search",
	"description": "Search for emails using Gmail search syntax",
	"input_schema": {
		"type": "object",
		"properties": {
			"query": {
				"type": "string",
				"description": "Gmail search query (e.g., 'from:user@example.com subject:meeting')"
			},
			"max_results": {
				"type": "integer",
				"description": "Max results to return (default 10, max 50)"
			}
		},
		"required": ["query"]
	}
}
```

**`email_send` Schema**:
```json
{
	"name": "email_send",
	"description": "Send an email",
	"input_schema": {
		"type": "object",
		"properties": {
			"to": {
				"type": "array",
				"items": { "type": "string" },
				"description": "Recipient email addresses"
			},
			"subject": { "type": "string" },
			"body": { "type": "string" },
			"cc": {
				"type": "array",
				"items": { "type": "string" },
				"description": "CC addresses (optional)"
			},
			"bcc": {
				"type": "array",
				"items": { "type": "string" },
				"description": "BCC addresses (optional)"
			}
		},
		"required": ["to", "subject", "body"]
	}
}
```

**`internal/tools/gmail/client.go`**:
```go
type Client struct {
	svc   *gmail.Service
	token *oauth2.Token
}

func (c *Client) ReadMessage(ctx context.Context, messageID string) (*Message, error)
func (c *Client) SearchMessages(ctx context.Context, query string, maxResults int) ([]MessageSummary, error)
func (c *Client) SendMessage(ctx context.Context, to, cc, bcc []string, subject, body string) (string, error)

type Message struct {
	ID        string
	From      string
	To        []string
	Subject   string
	Body      string
	Timestamp int64
}

type MessageSummary struct {
	ID      string
	From    string
	Subject string
}
```

## Out of Scope

- Email attachments (Phase 2+)
- HTML email rendering (plain text only)
- Multi-account support (single Gmail account per MVP)
- Email forwarding or auto-reply
- Label/folder operations (inbox only for Phase 1)

