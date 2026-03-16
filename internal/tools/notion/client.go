// Package notion provides a low-level Notion API client backed by an integration token.
package notion

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	notionBaseURL = "https://api.notion.com/v1"
	notionVersion = "2022-06-28"
)

// Client wraps an HTTP client for making authenticated Notion API requests.
type Client struct {
	apiToken   string
	httpClient *http.Client
}

// NewClient creates a new Notion API client using the supplied integration token.
func NewClient(apiToken string) *Client {
	return &Client{
		apiToken:   apiToken,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

// ReadPage retrieves a Notion page by ID and its block children (body text).
// Returns a merged map with keys: id, title, url, properties, content.
func (c *Client) ReadPage(ctx context.Context, pageID string) (map[string]interface{}, error) {
	log.Printf("[notion] client: ReadPage — pageID=%q", pageID)

	page, err := c.doRequest(ctx, http.MethodGet, "/pages/"+pageID, nil)
	if err != nil {
		log.Printf("[notion] client: ReadPage GET /pages/%s failed: %v", pageID, err)
		return nil, err
	}
	log.Printf("[notion] client: ReadPage page fetched OK")

	blocksResp, err := c.doRequest(ctx, http.MethodGet, "/blocks/"+pageID+"/children", nil)
	if err != nil {
		log.Printf("[notion] client: ReadPage GET /blocks/%s/children failed: %v", pageID, err)
		return nil, err
	}

	var blocks []interface{}
	if results, ok := blocksResp["results"].([]interface{}); ok {
		blocks = results
	}

	title := extractPageTitle(page)
	url, _ := page["url"].(string)
	properties, _ := page["properties"].(map[string]interface{})

	result := map[string]interface{}{
		"id":         pageID,
		"title":      title,
		"url":        url,
		"properties": properties,
		"content":    extractTextFromBlocks(blocks),
	}
	log.Printf("[notion] client: ReadPage OK — title=%q blockCount=%d", title, len(blocks))
	return result, nil
}

// QueryDatabase queries a Notion database by ID with an optional filter.
// Follows pagination up to 3 pages of results.
func (c *Client) QueryDatabase(ctx context.Context, databaseID string, filter map[string]interface{}) ([]map[string]interface{}, error) {
	log.Printf("[notion] client: QueryDatabase — databaseID=%q", databaseID)

	var allResults []map[string]interface{}
	var cursor string
	const maxPages = 3

	for page := 0; page < maxPages; page++ {
		body := map[string]interface{}{}
		if filter != nil {
			body["filter"] = filter
		}
		if cursor != "" {
			body["start_cursor"] = cursor
		}

		resp, err := c.doRequest(ctx, http.MethodPost, "/databases/"+databaseID+"/query", body)
		if err != nil {
			log.Printf("[notion] client: QueryDatabase POST /databases/%s/query failed: %v", databaseID, err)
			return nil, err
		}

		results, _ := resp["results"].([]interface{})
		for _, r := range results {
			if row, ok := r.(map[string]interface{}); ok {
				allResults = append(allResults, row)
			}
		}
		log.Printf("[notion] client: QueryDatabase page=%d results=%d", page+1, len(results))

		hasMore, _ := resp["has_more"].(bool)
		if !hasMore {
			break
		}
		nextCursor, _ := resp["next_cursor"].(string)
		if nextCursor == "" {
			break
		}
		cursor = nextCursor
	}

	log.Printf("[notion] client: QueryDatabase OK — total results=%d", len(allResults))
	return allResults, nil
}

// CreatePage creates a new Notion page under the given parent page ID.
// Returns (pageID, url, error).
func (c *Client) CreatePage(ctx context.Context, parentID, title string, properties map[string]interface{}) (string, string, error) {
	log.Printf("[notion] client: CreatePage — parentID=%q title=%q", parentID, title)

	titleProp := map[string]interface{}{
		"title": []interface{}{
			map[string]interface{}{
				"text": map[string]interface{}{
					"content": title,
				},
			},
		},
	}

	mergedProps := map[string]interface{}{}
	for k, v := range properties {
		mergedProps[k] = v
	}
	mergedProps["title"] = titleProp["title"]

	body := map[string]interface{}{
		"parent": map[string]interface{}{
			"page_id": parentID,
		},
		"properties": mergedProps,
	}

	resp, err := c.doRequest(ctx, http.MethodPost, "/pages", body)
	if err != nil {
		log.Printf("[notion] client: CreatePage POST /pages failed: %v", err)
		return "", "", err
	}

	pageID, _ := resp["id"].(string)
	url, _ := resp["url"].(string)
	log.Printf("[notion] client: CreatePage OK — pageID=%q url=%q", pageID, url)
	return pageID, url, nil
}

// UpdatePage updates properties of an existing Notion page.
func (c *Client) UpdatePage(ctx context.Context, pageID string, properties map[string]interface{}) error {
	log.Printf("[notion] client: UpdatePage — pageID=%q propertyCount=%d", pageID, len(properties))

	body := map[string]interface{}{
		"properties": properties,
	}

	_, err := c.doRequest(ctx, http.MethodPatch, "/pages/"+pageID, body)
	if err != nil {
		log.Printf("[notion] client: UpdatePage PATCH /pages/%s failed: %v", pageID, err)
		return err
	}

	log.Printf("[notion] client: UpdatePage OK")
	return nil
}

// doRequest performs an authenticated HTTP request against the Notion API.
// It marshals body (if non-nil), sets required headers, checks status codes,
// and returns the parsed JSON response or a mapped error.
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) (map[string]interface{}, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("notion: marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, notionBaseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("notion: build request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	req.Header.Set("Notion-Version", notionVersion)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("notion: http request: %w", err)
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("notion: read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("[notion] client: HTTP %d — path=%s body=%s", resp.StatusCode, path, string(respData))
		return nil, mapNotionError(resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respData, &result); err != nil {
		return nil, fmt.Errorf("notion: parse response: %w", err)
	}

	return result, nil
}

// mapNotionError converts well-known Notion HTTP status codes into descriptive errors.
func mapNotionError(statusCode int) error {
	switch statusCode {
	case http.StatusBadRequest:
		return fmt.Errorf("invalid request — check parameters")
	case http.StatusUnauthorized:
		return fmt.Errorf("notion token invalid or revoked — check api_token in config")
	case http.StatusForbidden:
		return fmt.Errorf("permission denied — integration may not have access to this page")
	case http.StatusNotFound:
		return fmt.Errorf("page or database not found — check the ID")
	case http.StatusTooManyRequests:
		return fmt.Errorf("Notion API rate limit exceeded — try again later")
	default:
		return fmt.Errorf("notion api error: HTTP %d", statusCode)
	}
}

// extractPageTitle pulls the plain-text title from a Notion page object.
func extractPageTitle(page map[string]interface{}) string {
	props, ok := page["properties"].(map[string]interface{})
	if !ok {
		return ""
	}

	// Try common title property names.
	for _, key := range []string{"title", "Title", "Name", "name"} {
		prop, ok := props[key].(map[string]interface{})
		if !ok {
			continue
		}
		titleArr, ok := prop["title"].([]interface{})
		if !ok {
			continue
		}
		var parts []string
		for _, t := range titleArr {
			if obj, ok := t.(map[string]interface{}); ok {
				if plain, ok := obj["plain_text"].(string); ok {
					parts = append(parts, plain)
				}
			}
		}
		if len(parts) > 0 {
			return strings.Join(parts, "")
		}
	}
	return ""
}

// extractTextFromBlocks converts a slice of Notion block objects to plain text.
// Supported block types: paragraph, heading_1, heading_2, heading_3,
// bulleted_list_item, numbered_list_item.
func extractTextFromBlocks(blocks []interface{}) string {
	var sb strings.Builder

	for _, raw := range blocks {
		block, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}

		blockType, _ := block["type"].(string)

		prefix := ""
		switch blockType {
		case "heading_1":
			prefix = "# "
		case "heading_2":
			prefix = "## "
		case "heading_3":
			prefix = "### "
		case "bulleted_list_item":
			prefix = "- "
		case "numbered_list_item":
			prefix = "- "
		case "paragraph":
			// no prefix
		default:
			continue
		}

		typeData, ok := block[blockType].(map[string]interface{})
		if !ok {
			continue
		}

		richText, ok := typeData["rich_text"].([]interface{})
		if !ok {
			continue
		}

		var parts []string
		for _, rt := range richText {
			if obj, ok := rt.(map[string]interface{}); ok {
				if plain, ok := obj["plain_text"].(string); ok {
					parts = append(parts, plain)
				}
			}
		}

		text := strings.Join(parts, "")
		if text == "" {
			continue
		}

		sb.WriteString(prefix)
		sb.WriteString(text)
		sb.WriteString("\n\n")
	}

	return strings.TrimSpace(sb.String())
}
