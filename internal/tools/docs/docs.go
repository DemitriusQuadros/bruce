// Package docs provides tools.Tool implementations for Google Docs read, create, and append operations.
package docs

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"bruce/internal/ai"
	"bruce/internal/auth"
)

// ReadResponse is the JSON response returned by the docs_read tool.
type ReadResponse struct {
	DocumentID   string `json:"document_id"`
	Title        string `json:"title"`
	Content      string `json:"content"`
	LastModified string `json:"last_modified"` // RFC3339
	OwnerEmail   string `json:"owner_email"`
	URL          string `json:"url"`
}

// CreateResponse is the JSON response returned by the docs_create tool.
type CreateResponse struct {
	DocumentID string `json:"document_id"`
	Title      string `json:"title"`
	URL        string `json:"url"`
	CreatedAt  string `json:"created_at"` // RFC3339
}

// AppendResponse is the JSON response returned by the docs_append tool.
type AppendResponse struct {
	DocumentID    string `json:"document_id"`
	ContentLength int64  `json:"content_length"`
}

// DocsReadTool implements tools.Tool for reading a Google Docs document.
type DocsReadTool struct {
	googleAuth *auth.GoogleHandler
}

// NewReadTool creates a new DocsReadTool backed by the supplied GoogleHandler.
func NewReadTool(googleAuth *auth.GoogleHandler) *DocsReadTool {
	return &DocsReadTool{googleAuth: googleAuth}
}

// Name returns the tool name.
func (t *DocsReadTool) Name() string { return "docs_read" }

// Definition returns the AI tool definition for docs_read.
func (t *DocsReadTool) Definition() ai.ToolDefinition {
	return ai.ToolDefinition{
		Name:        "docs_read",
		Description: "Read content from a Google Doc",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"document_id": map[string]interface{}{
					"type":        "string",
					"description": "Google Docs document ID (from URL: docs.google.com/document/d/{id}/)",
				},
			},
			"required": []string{"document_id"},
		},
	}
}

// Execute reads a Google Docs document and returns its content as markdown.
func (t *DocsReadTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	log.Printf("[docs] docs_read: Execute called with input keys: %v", mapKeys(input))

	documentID, ok := input["document_id"].(string)
	if !ok || documentID == "" {
		return nil, fmt.Errorf("document_id is required and must be a non-empty string")
	}
	log.Printf("[docs] docs_read: document_id=%q", documentID)

	log.Printf("[docs] docs_read: calling GetToken for user=default")
	token, err := t.googleAuth.GetToken(ctx, "default")
	if err != nil {
		log.Printf("[docs] docs_read: GetToken failed: %v", err)
		return nil, fmt.Errorf("docs_read: get token: %w", err)
	}
	log.Printf("[docs] docs_read: GetToken succeeded (token_type=%q, has_refresh_token=%v)", token.TokenType, token.RefreshToken != "")

	log.Printf("[docs] docs_read: building Docs API client")
	client, err := NewClient(ctx, token, t.googleAuth.OAuthConfig())
	if err != nil {
		log.Printf("[docs] docs_read: create client failed: %v", err)
		return nil, fmt.Errorf("docs_read: create client: %w", err)
	}
	log.Printf("[docs] docs_read: Docs API client built successfully")

	log.Printf("[docs] docs_read: calling ReadDocument document_id=%q", documentID)
	doc, err := client.ReadDocument(ctx, documentID)
	if err != nil {
		log.Printf("[docs] docs_read: ReadDocument failed: %v", err)
		return nil, fmt.Errorf("docs_read: %w", err)
	}
	log.Printf("[docs] docs_read: ReadDocument succeeded, title=%q contentLen=%d", doc.Title, len(doc.Content))

	resp := ReadResponse{
		DocumentID:   doc.ID,
		Title:        doc.Title,
		Content:      doc.Content,
		LastModified: doc.LastModified.Format(time.RFC3339),
		OwnerEmail:   doc.OwnerEmail,
		URL:          doc.URL,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("docs_read: marshal response: %w", err)
	}

	return string(data), nil
}

// DocsCreateTool implements tools.Tool for creating a new Google Docs document.
type DocsCreateTool struct {
	googleAuth *auth.GoogleHandler
}

// NewCreateTool creates a new DocsCreateTool backed by the supplied GoogleHandler.
func NewCreateTool(googleAuth *auth.GoogleHandler) *DocsCreateTool {
	return &DocsCreateTool{googleAuth: googleAuth}
}

// Name returns the tool name.
func (t *DocsCreateTool) Name() string { return "docs_create" }

// Definition returns the AI tool definition for docs_create.
func (t *DocsCreateTool) Definition() ai.ToolDefinition {
	return ai.ToolDefinition{
		Name:        "docs_create",
		Description: "Create a new Google Doc with initial content",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"title": map[string]interface{}{
					"type":        "string",
					"description": "Document title",
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "Initial content (markdown format)",
				},
			},
			"required": []string{"title"},
		},
	}
}

// Execute creates a new Google Docs document with the supplied title and optional content.
func (t *DocsCreateTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	log.Printf("[docs] docs_create: Execute called with input keys: %v", mapKeys(input))

	title, ok := input["title"].(string)
	if !ok || title == "" {
		return nil, fmt.Errorf("title is required and must be a non-empty string")
	}

	content := ""
	if v, ok := input["content"].(string); ok {
		content = v
	}
	log.Printf("[docs] docs_create: title=%q contentLen=%d", title, len(content))

	log.Printf("[docs] docs_create: calling GetToken for user=default")
	token, err := t.googleAuth.GetToken(ctx, "default")
	if err != nil {
		log.Printf("[docs] docs_create: GetToken failed: %v", err)
		return nil, fmt.Errorf("docs_create: get token: %w", err)
	}
	log.Printf("[docs] docs_create: GetToken succeeded (token_type=%q, has_refresh_token=%v)", token.TokenType, token.RefreshToken != "")

	log.Printf("[docs] docs_create: building Docs API client")
	client, err := NewClient(ctx, token, t.googleAuth.OAuthConfig())
	if err != nil {
		log.Printf("[docs] docs_create: create client failed: %v", err)
		return nil, fmt.Errorf("docs_create: create client: %w", err)
	}
	log.Printf("[docs] docs_create: Docs API client built successfully")

	log.Printf("[docs] docs_create: calling CreateDocument title=%q", title)
	docID, err := client.CreateDocument(ctx, title, content)
	if err != nil {
		log.Printf("[docs] docs_create: CreateDocument failed: %v", err)
		return nil, fmt.Errorf("docs_create: %w", err)
	}
	log.Printf("[docs] docs_create: CreateDocument succeeded, docID=%q", docID)

	resp := CreateResponse{
		DocumentID: docID,
		Title:      title,
		URL:        "https://docs.google.com/document/d/" + docID,
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
	}

	data, err := json.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("docs_create: marshal response: %w", err)
	}

	return string(data), nil
}

// DocsAppendTool implements tools.Tool for appending text to an existing Google Docs document.
type DocsAppendTool struct {
	googleAuth *auth.GoogleHandler
}

// NewAppendTool creates a new DocsAppendTool backed by the supplied GoogleHandler.
func NewAppendTool(googleAuth *auth.GoogleHandler) *DocsAppendTool {
	return &DocsAppendTool{googleAuth: googleAuth}
}

// Name returns the tool name.
func (t *DocsAppendTool) Name() string { return "docs_append" }

// Definition returns the AI tool definition for docs_append.
func (t *DocsAppendTool) Definition() ai.ToolDefinition {
	return ai.ToolDefinition{
		Name:        "docs_append",
		Description: "Append text to the end of a Google Doc",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"document_id": map[string]interface{}{
					"type":        "string",
					"description": "Google Docs document ID (from URL: docs.google.com/document/d/{id}/)",
				},
				"text": map[string]interface{}{
					"type":        "string",
					"description": "Text to append",
				},
			},
			"required": []string{"document_id", "text"},
		},
	}
}

// Execute appends text to the end of a Google Docs document and returns the new content length.
func (t *DocsAppendTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	log.Printf("[docs] docs_append: Execute called with input keys: %v", mapKeys(input))

	documentID, ok := input["document_id"].(string)
	if !ok || documentID == "" {
		return nil, fmt.Errorf("document_id is required and must be a non-empty string")
	}

	text, ok := input["text"].(string)
	if !ok || text == "" {
		return nil, fmt.Errorf("text is required and must be a non-empty string")
	}
	log.Printf("[docs] docs_append: document_id=%q textLen=%d", documentID, len(text))

	log.Printf("[docs] docs_append: calling GetToken for user=default")
	token, err := t.googleAuth.GetToken(ctx, "default")
	if err != nil {
		log.Printf("[docs] docs_append: GetToken failed: %v", err)
		return nil, fmt.Errorf("docs_append: get token: %w", err)
	}
	log.Printf("[docs] docs_append: GetToken succeeded (token_type=%q, has_refresh_token=%v)", token.TokenType, token.RefreshToken != "")

	log.Printf("[docs] docs_append: building Docs API client")
	client, err := NewClient(ctx, token, t.googleAuth.OAuthConfig())
	if err != nil {
		log.Printf("[docs] docs_append: create client failed: %v", err)
		return nil, fmt.Errorf("docs_append: create client: %w", err)
	}
	log.Printf("[docs] docs_append: Docs API client built successfully")

	log.Printf("[docs] docs_append: calling AppendText document_id=%q", documentID)
	newLength, err := client.AppendText(ctx, documentID, text)
	if err != nil {
		log.Printf("[docs] docs_append: AppendText failed: %v", err)
		return nil, fmt.Errorf("docs_append: %w", err)
	}
	log.Printf("[docs] docs_append: AppendText succeeded, newLength=%d", newLength)

	resp := AppendResponse{
		DocumentID:    documentID,
		ContentLength: newLength,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("docs_append: marshal response: %w", err)
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
