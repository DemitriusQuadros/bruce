# Spec 16: URL Reader Tool [BACKEND]

## Overview

Implement a `url_read` tool that fetches and parses web page content. Accepts a URL, follows redirects (max 3), extracts main content (body text, headings, links) and strips boilerplate (navigation, ads, scripts). Returns normalized markdown. Implements safeguards: timeouts (10s), robots.txt respect, and rejects private IP ranges.

## Phase

**Phase 1** (Weeks 1–6)

## Prerequisites

- Tool registry exists (Spec 12)
- No external dependencies required beyond stdlib

## Deliverables

**Files to Create:**
- `internal/tools/url/reader.go` — URL fetching and parsing
- `internal/tools/url/content_extractor.go` — HTML content extraction logic

**Files to Modify:**
- `internal/tools/registry.go` — register URL reader tool
- `cmd/bruce/main.go` — instantiate URL reader tool
- `config.example.yml` — document URL reader section (optional config for timeout)

## Acceptance Criteria

- [ ] Tool `url_read` accepts: `url` (string)
- [ ] Follows HTTP redirects up to 3 hops (301, 302, 307, 308)
- [ ] Returns extracted content: main text, headings, links (URL + anchor text), image alt text
- [ ] Respects `robots.txt` and `User-Agent` rules
- [ ] Rejects requests to private IPs (127.0.0.1, 10.*, 172.16-31.*, 192.168.*)
- [ ] HTTP request timeout: 10 seconds total
- [ ] Maximum response size: 10 MB
- [ ] Returned content is formatted as markdown (headings → `#`, bold → `**`, links → `[text](url)`)
- [ ] Strips scripts, styles, and tracking pixels
- [ ] Returns error with clear message for invalid URLs, timeouts, or blocked content
- [ ] Latency: p95 <5s (varies by page size)

## API / Component Contract

**`url_read` Schema**:
```json
{
	"name": "url_read",
	"description": "Fetch and extract main content from a web page",
	"input_schema": {
		"type": "object",
		"properties": {
			"url": {
				"type": "string",
				"description": "Full URL (must start with http:// or https://)"
			}
		},
		"required": ["url"]
	}
}
```

**Response Format** (as markdown string):
```
# Page Title

Main heading content here.

## Section 1
Paragraph text extracted from page.

- List item 1
- [Link text](https://example.com)

## Section 2
More content...
```

**`internal/tools/url/reader.go`**:
```go
type Client struct {
	timeout    time.Duration
	maxSize    int64
	maxRetries int
}

func (c *Client) FetchAndParse(ctx context.Context, url string) (string, error) {
	// Fetch URL with timeout
	// Follow redirects (max 3)
	// Check robots.txt
	// Extract main content
	// Return as markdown
}

func (c *Client) isPrivateIP(host string) bool { /* ... */ }
func (c *Client) htmlToMarkdown(body string) string { /* ... */ }
```

## Out of Scope

- JavaScript execution (static content only)
- Form submission or interaction
- Authentication/cookies (public pages only)
- PDF or binary file handling
- Dynamic content (AJAX-loaded data)

