# Spec 24: Google Docs Tool [BACKEND]

## Overview

Implement `docs_read`, `docs_create`, and `docs_append` tools that interact with Google Docs via Google Workspace API. All tools use OAuth tokens from Spec 13. `docs_read` retrieves document content as markdown; `docs_create` creates a new document with initial content; `docs_append` adds text to the end of an existing document. Tools validate document IDs and prevent unauthorized access.

## Phase

**Phase 1** (Weeks 1–6)

## Prerequisites

- Tool registry exists (Spec 12)
- Google OAuth flow complete (Spec 13)
- Google Docs API enabled in OAuth scopes

## Deliverables

**Files to Create:**
- `internal/tools/docs/docs.go` — Tool implementations (read, create, append)
- `internal/tools/docs/client.go` — Google Docs API wrapper using google.golang.org/api/docs/v1

**Files to Modify:**
- `internal/tools/registry.go` — register all three tools at startup
- `cmd/bruce/main.go` — instantiate Docs tool with OAuth client
- `config.example.yml` — document Docs tool section

## Acceptance Criteria

- [ ] Tool `docs_read` accepts: `document_id` (string, from Google Docs URL)
- [ ] Tool `docs_create` accepts: `title` (string), `content` (markdown-formatted string)
- [ ] Tool `docs_append` accepts: `document_id`, `text` (string)
- [ ] `docs_read` returns: document title, content as markdown, last modified timestamp, owner email
- [ ] `docs_create` returns: new document ID, shareable URL, creation timestamp
- [ ] `docs_append` appends text to end of document and returns updated content length
- [ ] Tools convert between markdown (user-friendly) and Google Docs format (bold, headings, lists)
- [ ] Tool handles 401 (expired token) by refreshing via Spec 13
- [ ] Tool handles 404 (document not found) with clear error message
- [ ] Tool handles 429 (rate limit) gracefully
- [ ] All three tools are thread-safe
- [ ] Latency: read <2s p95, create <3s p95, append <2s p95

## API / Component Contract

**`docs_read` Schema**:
```json
{
	"name": "docs_read",
	"description": "Read content from a Google Doc",
	"input_schema": {
		"type": "object",
		"properties": {
			"document_id": {
				"type": "string",
				"description": "Google Docs document ID (from URL: docs.google.com/document/d/{id}/)"
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
	"description": "Create a new Google Doc with initial content",
	"input_schema": {
		"type": "object",
		"properties": {
			"title": { "type": "string", "description": "Document title" },
			"content": {
				"type": "string",
				"description": "Initial content (markdown format)"
			}
		},
		"required": ["title", "content"]
	}
}
```

**`docs_append` Schema**:
```json
{
	"name": "docs_append",
	"description": "Append text to the end of a Google Doc",
	"input_schema": {
		"type": "object",
		"properties": {
			"document_id": { "type": "string" },
			"text": { "type": "string", "description": "Text to append" }
		},
		"required": ["document_id", "text"]
	}
}
```

**Response Format for `docs_read`**:
```json
{
	"document_id": "1abc23def456ghi789",
	"title": "My Document",
	"content": "# Heading\n\nParagraph text here.\n\n- List item 1\n- List item 2",
	"last_modified": "2024-03-13T14:02:00Z",
	"owner_email": "user@example.com",
	"url": "https://docs.google.com/document/d/1abc23def456ghi789/edit"
}
```

**Response Format for `docs_create`**:
```json
{
	"document_id": "1xyz98def456jkl789",
	"title": "My Document",
	"url": "https://docs.google.com/document/d/1xyz98def456jkl789/edit",
	"created_at": "2024-03-13T14:02:00Z"
}
```

**`internal/tools/docs/client.go`**:
```go
type Client struct {
	svc   *docs.Service
	token *oauth2.Token
}

func (c *Client) ReadDocument(ctx context.Context, documentID string) (*Document, error)
func (c *Client) CreateDocument(ctx context.Context, title, content string) (string, error)
func (c *Client) AppendText(ctx context.Context, documentID, text string) (int64, error)

type Document struct {
	ID           string
	Title        string
	Content      string
	LastModified time.Time
	OwnerEmail   string
	URL          string
}
```

## Out of Scope

- Collaborative editing or access control changes
- Comments or suggestions mode
- Drawing/image insertion
- Table formatting (basic lists only)
- Version history or revision tracking
- Sharing settings modification
- Real-time collaborative sync

