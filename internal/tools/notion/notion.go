// Package notion provides tools.Tool implementations for Notion read, create, and update operations.
package notion

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"bruce/internal/ai"
)

// ReadResponse is the JSON response returned by the notion_read tool.
type ReadResponse struct {
	PageID     string                 `json:"page_id"`
	Title      string                 `json:"title"`
	URL        string                 `json:"url"`
	Properties map[string]interface{} `json:"properties,omitempty"`
	Content    string                 `json:"content,omitempty"`
}

// CreateResponse is the JSON response returned by the notion_create tool.
type CreateResponse struct {
	PageID    string `json:"page_id"`
	Title     string `json:"title"`
	URL       string `json:"url"`
	CreatedAt string `json:"created_at"` // RFC3339
}

// UpdateResponse is the JSON response returned by the notion_update tool.
type UpdateResponse struct {
	PageID    string `json:"page_id"`
	UpdatedAt string `json:"updated_at"` // RFC3339
}

// NotionReadTool implements tools.Tool for reading a Notion page or querying a database.
type NotionReadTool struct {
	apiToken string
}

// NewReadTool creates a new NotionReadTool using the supplied integration token.
func NewReadTool(apiToken string) *NotionReadTool {
	return &NotionReadTool{apiToken: apiToken}
}

// Name returns the tool name.
func (t *NotionReadTool) Name() string { return "notion_read" }

// Definition returns the AI tool definition for notion_read.
func (t *NotionReadTool) Definition() ai.ToolDefinition {
	return ai.ToolDefinition{
		Name:        "notion_read",
		Description: "Read a Notion page or query a Notion database by ID",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"page_id": map[string]interface{}{
					"type":        "string",
					"description": "Notion page or database ID (UUID from the page URL)",
				},
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Optional search query — used when reading a database to filter results",
				},
			},
			"required": []string{"page_id"},
		},
	}
}

// Execute reads a Notion page or queries a Notion database and returns its content.
// If the page_id belongs to a database the tool falls back to QueryDatabase automatically.
func (t *NotionReadTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	log.Printf("[notion] notion_read: Execute called with input keys: %v", mapKeys(input))

	pageID, ok := input["page_id"].(string)
	if !ok || pageID == "" {
		return nil, fmt.Errorf("page_id is required and must be a non-empty string")
	}
	log.Printf("[notion] notion_read: page_id=%q", pageID)

	client := NewClient(t.apiToken)

	// Try to read as a page first.
	page, err := client.ReadPage(ctx, pageID)
	if err != nil {
		log.Printf("[notion] notion_read: ReadPage failed (%v) — trying QueryDatabase", err)
		// Fall back to database query.
		rows, dbErr := client.QueryDatabase(ctx, pageID, nil)
		if dbErr != nil {
			log.Printf("[notion] notion_read: QueryDatabase also failed: %v", dbErr)
			return nil, fmt.Errorf("notion_read: %w", err)
		}
		log.Printf("[notion] notion_read: QueryDatabase succeeded, rowCount=%d", len(rows))

		data, merr := json.Marshal(rows)
		if merr != nil {
			return nil, fmt.Errorf("notion_read: marshal database results: %w", merr)
		}
		return string(data), nil
	}

	props, _ := page["properties"].(map[string]interface{})
	resp := ReadResponse{
		PageID:     pageID,
		Title:      fmt.Sprintf("%v", page["title"]),
		URL:        fmt.Sprintf("%v", page["url"]),
		Properties: props,
		Content:    fmt.Sprintf("%v", page["content"]),
	}
	log.Printf("[notion] notion_read: page read OK — title=%q contentLen=%d", resp.Title, len(resp.Content))

	data, err := json.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("notion_read: marshal response: %w", err)
	}

	return string(data), nil
}

// NotionCreateTool implements tools.Tool for creating a new Notion page.
type NotionCreateTool struct {
	apiToken string
}

// NewCreateTool creates a new NotionCreateTool using the supplied integration token.
func NewCreateTool(apiToken string) *NotionCreateTool {
	return &NotionCreateTool{apiToken: apiToken}
}

// Name returns the tool name.
func (t *NotionCreateTool) Name() string { return "notion_create" }

// Definition returns the AI tool definition for notion_create.
func (t *NotionCreateTool) Definition() ai.ToolDefinition {
	return ai.ToolDefinition{
		Name:        "notion_create",
		Description: "Create a new Notion page under a parent page",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"parent_id": map[string]interface{}{
					"type":        "string",
					"description": "ID of the parent Notion page (the new page will be created as a child)",
				},
				"title": map[string]interface{}{
					"type":        "string",
					"description": "Title of the new page",
				},
				"properties": map[string]interface{}{
					"type":        "object",
					"description": "Optional additional Notion page properties (Notion property format)",
				},
			},
			"required": []string{"parent_id", "title"},
		},
	}
}

// Execute creates a new Notion page and returns its ID and URL.
func (t *NotionCreateTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	log.Printf("[notion] notion_create: Execute called with input keys: %v", mapKeys(input))

	parentID, ok := input["parent_id"].(string)
	if !ok || parentID == "" {
		return nil, fmt.Errorf("parent_id is required and must be a non-empty string")
	}

	title, ok := input["title"].(string)
	if !ok || title == "" {
		return nil, fmt.Errorf("title is required and must be a non-empty string")
	}

	var properties map[string]interface{}
	if raw, ok := input["properties"].(map[string]interface{}); ok {
		properties = raw
	}
	if properties == nil {
		properties = map[string]interface{}{}
	}
	log.Printf("[notion] notion_create: parent_id=%q title=%q extraProperties=%d", parentID, title, len(properties))

	client := NewClient(t.apiToken)

	pageID, url, err := client.CreatePage(ctx, parentID, title, properties)
	if err != nil {
		log.Printf("[notion] notion_create: CreatePage failed: %v", err)
		return nil, fmt.Errorf("notion_create: %w", err)
	}
	log.Printf("[notion] notion_create: CreatePage succeeded — pageID=%q url=%q", pageID, url)

	resp := CreateResponse{
		PageID:    pageID,
		Title:     title,
		URL:       url,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	data, err := json.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("notion_create: marshal response: %w", err)
	}

	return string(data), nil
}

// NotionUpdateTool implements tools.Tool for updating properties of an existing Notion page.
type NotionUpdateTool struct {
	apiToken string
}

// NewUpdateTool creates a new NotionUpdateTool using the supplied integration token.
func NewUpdateTool(apiToken string) *NotionUpdateTool {
	return &NotionUpdateTool{apiToken: apiToken}
}

// Name returns the tool name.
func (t *NotionUpdateTool) Name() string { return "notion_update" }

// Definition returns the AI tool definition for notion_update.
func (t *NotionUpdateTool) Definition() ai.ToolDefinition {
	return ai.ToolDefinition{
		Name:        "notion_update",
		Description: "Update properties of an existing Notion page",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"page_id": map[string]interface{}{
					"type":        "string",
					"description": "ID of the Notion page to update",
				},
				"properties": map[string]interface{}{
					"type":        "object",
					"description": "Notion page properties to update (Notion property format)",
				},
			},
			"required": []string{"page_id", "properties"},
		},
	}
}

// Execute updates the specified Notion page properties.
func (t *NotionUpdateTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	log.Printf("[notion] notion_update: Execute called with input keys: %v", mapKeys(input))

	pageID, ok := input["page_id"].(string)
	if !ok || pageID == "" {
		return nil, fmt.Errorf("page_id is required and must be a non-empty string")
	}

	properties, ok := input["properties"].(map[string]interface{})
	if !ok || len(properties) == 0 {
		return nil, fmt.Errorf("properties is required and must be a non-empty object")
	}
	log.Printf("[notion] notion_update: page_id=%q propertyCount=%d", pageID, len(properties))

	client := NewClient(t.apiToken)

	if err := client.UpdatePage(ctx, pageID, properties); err != nil {
		log.Printf("[notion] notion_update: UpdatePage failed: %v", err)
		return nil, fmt.Errorf("notion_update: %w", err)
	}
	log.Printf("[notion] notion_update: UpdatePage succeeded")

	resp := UpdateResponse{
		PageID:    pageID,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	data, err := json.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("notion_update: marshal response: %w", err)
	}

	return string(data), nil
}

// mapKeys returns a slice of keys from a map for debug logging.
func mapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
