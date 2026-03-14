// Package gmail provides a low-level Gmail API client backed by Google OAuth 2.0 tokens.
package gmail

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"mime"
	"strings"

	"golang.org/x/oauth2"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

// Message represents a full Gmail message with decoded body.
type Message struct {
	ID      string
	From    string
	To      string
	Subject string
	Date    string
	Body    string
	Snippet string
}

// MessageSummary is a lightweight view of a Gmail message returned by search.
type MessageSummary struct {
	ID      string
	From    string
	Subject string
	Date    string
	Snippet string
}

// Client wraps a Gmail API service.
type Client struct {
	svc *gmail.Service
}

// NewClient creates a new Gmail API client using the supplied OAuth token and config.
func NewClient(ctx context.Context, token *oauth2.Token, oauthCfg *oauth2.Config) (*Client, error) {
	log.Printf("[gmail] client: building service (token_type=%q, expiry=%s)", token.TokenType, token.Expiry.String())
	src := oauthCfg.TokenSource(ctx, token)
	svc, err := gmail.NewService(ctx, option.WithTokenSource(src))
	if err != nil {
		log.Printf("[gmail] client: gmail.NewService failed: %v", err)
		return nil, fmt.Errorf("gmail: create service: %w", err)
	}
	log.Printf("[gmail] client: service built successfully")
	return &Client{svc: svc}, nil
}

// ReadMessage retrieves a full Gmail message by its ID.
func (c *Client) ReadMessage(ctx context.Context, messageID string) (*Message, error) {
	msg, err := c.svc.Users.Messages.Get("me", messageID).Format("full").Context(ctx).Do()
	if err != nil {
		return nil, mapGmailError(err)
	}

	m := &Message{
		ID:      msg.Id,
		Snippet: msg.Snippet,
	}

	// Extract headers.
	for _, h := range msg.Payload.Headers {
		switch strings.ToLower(h.Name) {
		case "from":
			m.From = h.Value
		case "to":
			m.To = h.Value
		case "subject":
			m.Subject = decodeRFC2047(h.Value)
		case "date":
			m.Date = h.Value
		}
	}

	// Extract body.
	m.Body = extractBody(msg.Payload)

	return m, nil
}

// SearchMessages searches Gmail messages using the given query and returns summaries.
// maxResults caps the number of results (clamped to 1–100).
func (c *Client) SearchMessages(ctx context.Context, query string, maxResults int) ([]MessageSummary, error) {
	if maxResults < 1 {
		maxResults = 1
	}
	if maxResults > 100 {
		maxResults = 100
	}

	resp, err := c.svc.Users.Messages.List("me").
		Q(query).
		MaxResults(int64(maxResults)).
		Context(ctx).
		Do()
	if err != nil {
		return nil, mapGmailError(err)
	}

	summaries := make([]MessageSummary, 0, len(resp.Messages))
	for _, ref := range resp.Messages {
		msg, err := c.svc.Users.Messages.Get("me", ref.Id).
			Format("metadata").
			MetadataHeaders("From", "Subject", "Date").
			Context(ctx).
			Do()
		if err != nil {
			// Skip messages that cannot be fetched individually.
			continue
		}

		s := MessageSummary{
			ID:      msg.Id,
			Snippet: msg.Snippet,
		}
		for _, h := range msg.Payload.Headers {
			switch strings.ToLower(h.Name) {
			case "from":
				s.From = h.Value
			case "subject":
				s.Subject = decodeRFC2047(h.Value)
			case "date":
				s.Date = h.Value
			}
		}
		summaries = append(summaries, s)
	}

	return summaries, nil
}

// SendMessage sends an email via Gmail. Returns the sent message ID.
func (c *Client) SendMessage(ctx context.Context, to, cc, bcc []string, subject, body string) (string, error) {
	var sb strings.Builder

	// Build RFC 2822 headers.
	sb.WriteString("To: " + strings.Join(to, ", ") + "\r\n")
	if len(cc) > 0 {
		sb.WriteString("Cc: " + strings.Join(cc, ", ") + "\r\n")
	}
	if len(bcc) > 0 {
		sb.WriteString("Bcc: " + strings.Join(bcc, ", ") + "\r\n")
	}
	sb.WriteString("Subject: " + subject + "\r\n")
	sb.WriteString("MIME-Version: 1.0\r\n")
	sb.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	sb.WriteString("\r\n")
	sb.WriteString(body)

	// Encode as base64url.
	encoded := base64.URLEncoding.EncodeToString([]byte(sb.String()))

	msg := &gmail.Message{
		Raw: encoded,
	}

	sent, err := c.svc.Users.Messages.Send("me", msg).Context(ctx).Do()
	if err != nil {
		return "", mapGmailError(err)
	}

	return sent.Id, nil
}

// mapGmailError converts well-known Google API errors into descriptive errors.
func mapGmailError(err error) error {
	if apiErr, ok := err.(*googleapi.Error); ok {
		log.Printf("[gmail] client: Google API error HTTP %d — %s (details: %v)", apiErr.Code, apiErr.Message, apiErr.Details)
		switch apiErr.Code {
		case 401:
			return fmt.Errorf("token expired or revoked")
		case 429:
			return fmt.Errorf("gmail rate limit exceeded")
		}
		return fmt.Errorf("gmail api error %d: %s", apiErr.Code, apiErr.Message)
	}
	log.Printf("[gmail] client: non-API error: %v", err)
	return err
}

// extractBody walks the message payload parts to find the plain-text body.
func extractBody(payload *gmail.MessagePart) string {
	if payload == nil {
		return ""
	}

	// Direct body.
	if payload.Body != nil && payload.Body.Size > 0 {
		data, err := base64.URLEncoding.DecodeString(payload.Body.Data)
		if err == nil {
			return string(data)
		}
	}

	// Walk parts for text/plain.
	for _, part := range payload.Parts {
		if strings.HasPrefix(part.MimeType, "text/plain") {
			if part.Body != nil && part.Body.Size > 0 {
				data, err := base64.URLEncoding.DecodeString(part.Body.Data)
				if err == nil {
					return string(data)
				}
			}
		}
	}

	// Fall back to first part.
	for _, part := range payload.Parts {
		if body := extractBody(part); body != "" {
			return body
		}
	}

	return ""
}

// decodeRFC2047 attempts to decode an RFC 2047 encoded header value.
// Returns the original string if decoding fails.
func decodeRFC2047(s string) string {
	dec := new(mime.WordDecoder)
	decoded, err := dec.DecodeHeader(s)
	if err != nil {
		return s
	}
	return decoded
}
