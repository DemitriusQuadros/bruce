# Spec 16: URL Reader Tool [BACKEND]

## Overview

Implement `url_read` tool that fetches any HTTP(S) URL, extracts readable text (strips navigation, ads, boilerplate), and returns clean content. Uses `net/http` for fetching and `golang.org/x/net/html` for parsing. Implements simple caching (1-hour TTL) to avoid repeated fetches. Tool is self-contained; no OAuth or API keys required.

## Phase

**Phase 1** (Weeks 1–6)

## Prerequisites

- Tool registry exists (Spec 12)
- Standard Go libraries available

## Deliverables

**Files to Create:**
- `internal/tools/web/web.go` — URL reader tool implementation
- `internal/tools/web/parser.go` — HTML parsing and text extraction
- `internal/tools/web/cache.go` — in-memory cache with TTL

**Files to Modify:**
- `internal/tools/registry.go` — register URL reader tool at startup
- `cmd/bruce/main.go` — instantiate web tool

## Acceptance Criteria

- [ ] Tool name is `url_read`; schema includes: `url` (string), `use_cache` (bool, optional, default true)
- [ ] Tool fetches URL with configurable timeout (10s default)
- [ ] Tool extracts main text content, removing: nav bars, sidebars, ads, scripts, styles, metadata
- [ ] Tool returns: title, meta description (if available), clean text (first 5000 chars max)
- [ ] Tool caches successful fetches for 1 hour (by URL)
- [ ] Tool handles 404, 403, 5xx errors gracefully with error message
- [ ] Tool handles non-HTML content (JSON, plain text) by returning as-is
- [ ] Tool respects `robots.txt` (check if any, log if blocked)
- [ ] Latency: cached hit <10ms, first fetch <10s p95
- [ ] Tool does not follow redirects beyond 3 hops (prevent redirect loops)

## API / Component Contract

**Tool Schema**:
```json
{
	"name": "url_read",
	"description": "Fetch and extract readable text from any URL",
	"input_schema": {
		"type": "object",
		"properties": {
			"url": {
				"type": "string",
				"format": "uri",
				"description": "HTTP(S) URL to fetch"
			},
			"use_cache": {
				"type": "boolean",
				"description": "Use cached content if available (1-hour TTL)",
				"default": true
			}
		},
		"required": ["url"]
	}
}
```

**`internal/tools/web/web.go`**:
```go
type URLReader struct {
	cache Cache
}

func (r *URLReader) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error)

type URLContent struct {
	URL         string
	Title       string
	Description string
	Text        string
	FetchedAt   time.Time
	FromCache   bool
}
```

## Out of Scope

- PDF extraction
- JavaScript rendering (static HTML only)
- Full-text search
- Link extraction (simple text is the goal)
- Proxy support
- Authentication-protected URLs (public content only)
