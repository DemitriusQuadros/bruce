// Package gmail provides tools.Tool implementations for Gmail read, search, and send operations.
package gmail

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/mail"

	"bruce/internal/ai"
	"bruce/internal/auth"
)

// ReadEmailTool implements tools.Tool for reading a Gmail message by ID.
type ReadEmailTool struct {
	googleAuth *auth.GoogleHandler
}

// NewReadTool creates a new ReadEmailTool backed by the supplied GoogleHandler.
func NewReadTool(googleAuth *auth.GoogleHandler) *ReadEmailTool {
	return &ReadEmailTool{googleAuth: googleAuth}
}

// Name returns the tool name.
func (t *ReadEmailTool) Name() string { return "email_read" }

// Definition returns the AI tool definition for email_read.
func (t *ReadEmailTool) Definition() ai.ToolDefinition {
	return ai.ToolDefinition{
		Name:        "email_read",
		Description: "Read a specific Gmail message by ID",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"message_id": map[string]interface{}{
					"type":        "string",
					"description": "The Gmail message ID to retrieve",
				},
			},
			"required": []string{"message_id"},
		},
	}
}

// Execute retrieves the Gmail message identified by message_id.
func (t *ReadEmailTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	log.Printf("[gmail] email_read: Execute called with input keys: %v", mapKeys(input))

	messageID, ok := input["message_id"].(string)
	if !ok || messageID == "" {
		return nil, fmt.Errorf("message_id is required and must be a non-empty string")
	}
	log.Printf("[gmail] email_read: reading message_id=%q", messageID)

	log.Printf("[gmail] email_read: calling GetToken for user=default")
	token, err := t.googleAuth.GetToken(ctx, "default")
	if err != nil {
		log.Printf("[gmail] email_read: GetToken failed: %v", err)
		return nil, fmt.Errorf("email_read: get token: %w", err)
	}
	log.Printf("[gmail] email_read: GetToken succeeded (token_type=%q, has_refresh_token=%v)", token.TokenType, token.RefreshToken != "")

	log.Printf("[gmail] email_read: building Gmail API client")
	client, err := NewClient(ctx, token, t.googleAuth.OAuthConfig())
	if err != nil {
		log.Printf("[gmail] email_read: create client failed: %v", err)
		return nil, fmt.Errorf("email_read: create client: %w", err)
	}
	log.Printf("[gmail] email_read: Gmail API client built successfully")

	log.Printf("[gmail] email_read: calling Messages.Get for message_id=%q", messageID)
	msg, err := client.ReadMessage(ctx, messageID)
	if err != nil {
		log.Printf("[gmail] email_read: Messages.Get failed: %v", err)
		return nil, fmt.Errorf("email_read: %w", err)
	}
	log.Printf("[gmail] email_read: Messages.Get succeeded (subject=%q, from=%q)", msg.Subject, msg.From)

	data, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("email_read: marshal response: %w", err)
	}

	return string(data), nil
}

// SearchEmailTool implements tools.Tool for searching Gmail messages.
type SearchEmailTool struct {
	googleAuth *auth.GoogleHandler
}

// NewSearchTool creates a new SearchEmailTool backed by the supplied GoogleHandler.
func NewSearchTool(googleAuth *auth.GoogleHandler) *SearchEmailTool {
	return &SearchEmailTool{googleAuth: googleAuth}
}

// Name returns the tool name.
func (t *SearchEmailTool) Name() string { return "email_search" }

// Definition returns the AI tool definition for email_search.
func (t *SearchEmailTool) Definition() ai.ToolDefinition {
	return ai.ToolDefinition{
		Name:        "email_search",
		Description: "Search Gmail messages",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Gmail search query (same syntax as the Gmail search box)",
				},
				"max_results": map[string]interface{}{
					"type":        "number",
					"description": "Maximum number of messages to return (default 10, max 100)",
				},
			},
			"required": []string{"query"},
		},
	}
}

// Execute searches Gmail using the provided query.
func (t *SearchEmailTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	log.Printf("[gmail] email_search: Execute called with input keys: %v", mapKeys(input))

	query, ok := input["query"].(string)
	if !ok || query == "" {
		return nil, fmt.Errorf("query is required and must be a non-empty string")
	}

	maxResults := 10
	if v, ok := input["max_results"].(float64); ok && v > 0 {
		maxResults = int(v)
	}
	log.Printf("[gmail] email_search: query=%q max_results=%d", query, maxResults)

	log.Printf("[gmail] email_search: calling GetToken for user=default")
	token, err := t.googleAuth.GetToken(ctx, "default")
	if err != nil {
		log.Printf("[gmail] email_search: GetToken failed: %v", err)
		return nil, fmt.Errorf("email_search: get token: %w", err)
	}
	log.Printf("[gmail] email_search: GetToken succeeded (token_type=%q, has_refresh_token=%v)", token.TokenType, token.RefreshToken != "")

	log.Printf("[gmail] email_search: building Gmail API client")
	client, err := NewClient(ctx, token, t.googleAuth.OAuthConfig())
	if err != nil {
		log.Printf("[gmail] email_search: create client failed: %v", err)
		return nil, fmt.Errorf("email_search: create client: %w", err)
	}
	log.Printf("[gmail] email_search: Gmail API client built successfully")

	log.Printf("[gmail] email_search: calling Messages.List with query=%q", query)
	summaries, err := client.SearchMessages(ctx, query, maxResults)
	if err != nil {
		log.Printf("[gmail] email_search: Messages.List failed: %v", err)
		return nil, fmt.Errorf("email_search: %w", err)
	}
	log.Printf("[gmail] email_search: Messages.List succeeded, returned %d messages", len(summaries))

	data, err := json.Marshal(summaries)
	if err != nil {
		return nil, fmt.Errorf("email_search: marshal response: %w", err)
	}

	return string(data), nil
}

// SendEmailTool implements tools.Tool for sending an email via Gmail.
type SendEmailTool struct {
	googleAuth *auth.GoogleHandler
}

// NewSendTool creates a new SendEmailTool backed by the supplied GoogleHandler.
func NewSendTool(googleAuth *auth.GoogleHandler) *SendEmailTool {
	return &SendEmailTool{googleAuth: googleAuth}
}

// Name returns the tool name.
func (t *SendEmailTool) Name() string { return "email_send" }

// Definition returns the AI tool definition for email_send.
func (t *SendEmailTool) Definition() ai.ToolDefinition {
	return ai.ToolDefinition{
		Name:        "email_send",
		Description: "Send an email via Gmail",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"to": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "string"},
					"description": "Recipient email addresses",
				},
				"cc": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "string"},
					"description": "CC email addresses",
				},
				"bcc": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "string"},
					"description": "BCC email addresses",
				},
				"subject": map[string]interface{}{
					"type":        "string",
					"description": "Email subject line",
				},
				"body": map[string]interface{}{
					"type":        "string",
					"description": "Plain-text email body",
				},
			},
			"required": []string{"to", "subject", "body"},
		},
	}
}

// Execute sends the email. All address fields are validated with net/mail.
func (t *SendEmailTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	log.Printf("[gmail] email_send: Execute called with input keys: %v", mapKeys(input))

	to, err := extractAddresses(input, "to", true)
	if err != nil {
		return nil, err
	}

	cc, err := extractAddresses(input, "cc", false)
	if err != nil {
		return nil, err
	}

	bcc, err := extractAddresses(input, "bcc", false)
	if err != nil {
		return nil, err
	}

	subject, ok := input["subject"].(string)
	if !ok || subject == "" {
		return nil, fmt.Errorf("subject is required and must be a non-empty string")
	}

	body, ok := input["body"].(string)
	if !ok || body == "" {
		return nil, fmt.Errorf("body is required and must be a non-empty string")
	}
	log.Printf("[gmail] email_send: to=%v cc_count=%d bcc_count=%d subject=%q body_len=%d", to, len(cc), len(bcc), subject, len(body))

	log.Printf("[gmail] email_send: calling GetToken for user=default")
	token, err := t.googleAuth.GetToken(ctx, "default")
	if err != nil {
		log.Printf("[gmail] email_send: GetToken failed: %v", err)
		return nil, fmt.Errorf("email_send: get token: %w", err)
	}
	log.Printf("[gmail] email_send: GetToken succeeded (token_type=%q, has_refresh_token=%v)", token.TokenType, token.RefreshToken != "")

	log.Printf("[gmail] email_send: building Gmail API client")
	client, err := NewClient(ctx, token, t.googleAuth.OAuthConfig())
	if err != nil {
		log.Printf("[gmail] email_send: create client failed: %v", err)
		return nil, fmt.Errorf("email_send: create client: %w", err)
	}
	log.Printf("[gmail] email_send: Gmail API client built successfully")

	log.Printf("[gmail] email_send: calling Messages.Send to=%v subject=%q", to, subject)
	sentID, err := client.SendMessage(ctx, to, cc, bcc, subject, body)
	if err != nil {
		log.Printf("[gmail] email_send: Messages.Send failed: %v", err)
		return nil, fmt.Errorf("email_send: %w", err)
	}
	log.Printf("[gmail] email_send: Messages.Send succeeded, sent_message_id=%q", sentID)

	data, err := json.Marshal(map[string]string{"message_id": sentID, "status": "sent"})
	if err != nil {
		return nil, fmt.Errorf("email_send: marshal response: %w", err)
	}

	return string(data), nil
}

// extractAddresses pulls a slice of validated email address strings from the input map.
// If required is true and the key is missing or empty, an error is returned.
func extractAddresses(input map[string]interface{}, key string, required bool) ([]string, error) {
	raw, exists := input[key]
	if !exists || raw == nil {
		if required {
			return nil, fmt.Errorf("%s is required", key)
		}
		return nil, nil
	}

	slice, ok := raw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("%s must be an array of strings", key)
	}

	if required && len(slice) == 0 {
		return nil, fmt.Errorf("%s must contain at least one address", key)
	}

	addrs := make([]string, 0, len(slice))
	for i, v := range slice {
		addr, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("%s[%d] must be a string", key, i)
		}
		if _, err := mail.ParseAddress(addr); err != nil {
			return nil, fmt.Errorf("%s[%d] %q is not a valid email address: %w", key, i, addr, err)
		}
		addrs = append(addrs, addr)
	}

	return addrs, nil
}

// mapKeys returns a slice of keys from a map for debug logging.
func mapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
