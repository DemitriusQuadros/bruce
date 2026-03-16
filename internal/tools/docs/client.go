// Package docs provides a low-level Google Docs and Drive API client backed by OAuth 2.0 tokens.
package docs

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"google.golang.org/api/docs/v1"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

// Document represents a Google Docs document with its content rendered as markdown.
type Document struct {
	ID           string
	Title        string
	Content      string    // markdown-formatted
	LastModified time.Time
	OwnerEmail   string
	URL          string
}

// Client wraps a Google Docs API service and a Drive API service.
type Client struct {
	svc      *docs.Service
	driveSvc *drive.Service
}

// NewClient creates a new Docs API client using the supplied OAuth token and config.
func NewClient(ctx context.Context, token *oauth2.Token, oauthCfg *oauth2.Config) (*Client, error) {
	log.Printf("[docs] client: building service (token_type=%q, expiry=%s)", token.TokenType, token.Expiry.String())
	src := oauthCfg.TokenSource(ctx, token)

	svc, err := docs.NewService(ctx, option.WithTokenSource(src))
	if err != nil {
		log.Printf("[docs] client: docs.NewService failed: %v", err)
		return nil, fmt.Errorf("docs: create docs service: %w", err)
	}

	driveSvc, err := drive.NewService(ctx, option.WithTokenSource(src))
	if err != nil {
		log.Printf("[docs] client: drive.NewService failed: %v", err)
		return nil, fmt.Errorf("docs: create drive service: %w", err)
	}

	log.Printf("[docs] client: services built successfully")
	return &Client{svc: svc, driveSvc: driveSvc}, nil
}

// ReadDocument retrieves a Google Docs document and returns it with content rendered as markdown.
func (c *Client) ReadDocument(ctx context.Context, documentID string) (*Document, error) {
	log.Printf("[docs] client: Documents.Get — documentID=%q", documentID)

	doc, err := c.svc.Documents.Get(documentID).Context(ctx).Do()
	if err != nil {
		log.Printf("[docs] client: Documents.Get raw error: %T — %v", err, err)
		return nil, mapDocsError(err)
	}
	log.Printf("[docs] client: Documents.Get OK — title=%q", doc.Title)

	// Fetch metadata (modified time and owner) from Drive API.
	log.Printf("[docs] client: Drive.Files.Get — documentID=%q", documentID)
	meta, err := c.driveSvc.Files.Get(documentID).Fields("modifiedTime,owners").SupportsAllDrives(true).Context(ctx).Do()
	if err != nil {
		log.Printf("[docs] client: Drive.Files.Get raw error: %T — %v", err, err)
		return nil, mapDocsError(err)
	}

	var lastModified time.Time
	if meta.ModifiedTime != "" {
		lastModified, _ = time.Parse(time.RFC3339, meta.ModifiedTime)
	}

	ownerEmail := ""
	if len(meta.Owners) > 0 {
		ownerEmail = meta.Owners[0].EmailAddress
	}

	content := docsToMarkdown(doc.Body.Content)

	return &Document{
		ID:           documentID,
		Title:        doc.Title,
		Content:      content,
		LastModified: lastModified,
		OwnerEmail:   ownerEmail,
		URL:          "https://docs.google.com/document/d/" + documentID,
	}, nil
}

// CreateDocument creates a new Google Doc with the given title and optional markdown content.
// Returns the new document ID.
func (c *Client) CreateDocument(ctx context.Context, title, content string) (string, error) {
	log.Printf("[docs] client: Documents.Create — title=%q contentLen=%d", title, len(content))

	created, err := c.svc.Documents.Create(&docs.Document{Title: title}).Context(ctx).Do()
	if err != nil {
		log.Printf("[docs] client: Documents.Create raw error: %T — %v", err, err)
		return "", mapDocsError(err)
	}
	docID := created.DocumentId
	log.Printf("[docs] client: Documents.Create OK — docID=%q", docID)

	if content == "" {
		return docID, nil
	}

	// Build BatchUpdate requests to insert content and apply heading styles.
	requests := buildInsertRequests(content)
	if len(requests) == 0 {
		return docID, nil
	}

	log.Printf("[docs] client: Documents.BatchUpdate — docID=%q requestCount=%d", docID, len(requests))
	_, err = c.svc.Documents.BatchUpdate(docID, &docs.BatchUpdateDocumentRequest{
		Requests: requests,
	}).Context(ctx).Do()
	if err != nil {
		log.Printf("[docs] client: Documents.BatchUpdate raw error: %T — %v", err, err)
		return "", mapDocsError(err)
	}
	log.Printf("[docs] client: Documents.BatchUpdate OK")

	return docID, nil
}

// AppendText appends the given text to the end of an existing Google Doc.
// Returns the approximate new content length in characters.
func (c *Client) AppendText(ctx context.Context, documentID, text string) (int64, error) {
	log.Printf("[docs] client: AppendText — documentID=%q textLen=%d", documentID, len(text))

	// Fetch current document to find the end index.
	doc, err := c.svc.Documents.Get(documentID).Context(ctx).Do()
	if err != nil {
		log.Printf("[docs] client: Documents.Get (for append) raw error: %T — %v", err, err)
		return 0, mapDocsError(err)
	}

	// The end index of the last element minus 1 is the insertion point.
	endIndex := int64(1)
	if len(doc.Body.Content) > 0 {
		last := doc.Body.Content[len(doc.Body.Content)-1]
		if last.EndIndex > 1 {
			endIndex = last.EndIndex - 1
		}
	}
	log.Printf("[docs] client: Documents.Get OK — endIndex=%d", endIndex)

	requests := []*docs.Request{
		{
			InsertText: &docs.InsertTextRequest{
				Text: text,
				Location: &docs.Location{
					Index: endIndex,
				},
			},
		},
	}

	log.Printf("[docs] client: Documents.BatchUpdate (append) — documentID=%q insertAt=%d", documentID, endIndex)
	_, err = c.svc.Documents.BatchUpdate(documentID, &docs.BatchUpdateDocumentRequest{
		Requests: requests,
	}).Context(ctx).Do()
	if err != nil {
		log.Printf("[docs] client: Documents.BatchUpdate (append) raw error: %T — %v", err, err)
		return 0, mapDocsError(err)
	}

	newLength := endIndex + int64(len(text))
	log.Printf("[docs] client: AppendText OK — newLength=%d", newLength)
	return newLength, nil
}

// docsToMarkdown converts a slice of Google Docs structural elements to markdown text.
func docsToMarkdown(content []*docs.StructuralElement) string {
	var sb strings.Builder

	for _, elem := range content {
		if elem.Paragraph == nil {
			continue
		}
		para := elem.Paragraph

		// Determine heading prefix.
		headingPrefix := ""
		if para.ParagraphStyle != nil {
			switch para.ParagraphStyle.NamedStyleType {
			case "HEADING_1":
				headingPrefix = "# "
			case "HEADING_2":
				headingPrefix = "## "
			case "HEADING_3":
				headingPrefix = "### "
			case "HEADING_4":
				headingPrefix = "#### "
			case "HEADING_5":
				headingPrefix = "##### "
			case "HEADING_6":
				headingPrefix = "###### "
			}
		}

		// Accumulate text from paragraph elements.
		var paraText strings.Builder
		for _, pe := range para.Elements {
			if pe.TextRun == nil {
				continue
			}
			run := pe.TextRun
			text := run.Content

			// Apply bold and italic formatting.
			if run.TextStyle != nil {
				bold := run.TextStyle.Bold
				italic := run.TextStyle.Italic
				switch {
				case bold && italic:
					text = "***" + text + "***"
				case bold:
					text = "**" + text + "**"
				case italic:
					text = "*" + text + "*"
				}
			}
			paraText.WriteString(text)
		}

		line := strings.TrimRight(paraText.String(), "\n")
		if line == "" {
			continue
		}

		sb.WriteString(headingPrefix)
		sb.WriteString(line)
		sb.WriteString("\n\n")
	}

	return strings.TrimSpace(sb.String())
}

// buildInsertRequests converts markdown content into a slice of Google Docs API requests
// that insert the text and apply heading styles where applicable.
func buildInsertRequests(content string) []*docs.Request {
	lines := strings.Split(content, "\n")
	var requests []*docs.Request

	// Collect all text first as a single insertion at index 1.
	requests = append(requests, &docs.Request{
		InsertText: &docs.InsertTextRequest{
			Text: content,
			Location: &docs.Location{
				Index: 1,
			},
		},
	})

	// Apply heading styles by scanning lines for markdown headings.
	// We need to calculate byte offsets after the insertion.
	index := int64(1)
	for _, line := range lines {
		lineLen := int64(len(line)) + 1 // +1 for \n

		namedStyle := ""
		switch {
		case strings.HasPrefix(line, "###### "):
			namedStyle = "HEADING_6"
		case strings.HasPrefix(line, "##### "):
			namedStyle = "HEADING_5"
		case strings.HasPrefix(line, "#### "):
			namedStyle = "HEADING_4"
		case strings.HasPrefix(line, "### "):
			namedStyle = "HEADING_3"
		case strings.HasPrefix(line, "## "):
			namedStyle = "HEADING_2"
		case strings.HasPrefix(line, "# "):
			namedStyle = "HEADING_1"
		}

		if namedStyle != "" {
			requests = append(requests, &docs.Request{
				UpdateParagraphStyle: &docs.UpdateParagraphStyleRequest{
					Range: &docs.Range{
						StartIndex: index,
						EndIndex:   index + lineLen,
					},
					ParagraphStyle: &docs.ParagraphStyle{
						NamedStyleType: namedStyle,
					},
					Fields: "namedStyleType",
				},
			})
		}

		index += lineLen
	}

	return requests
}

// mapDocsError converts well-known Google API errors into descriptive errors.
func mapDocsError(err error) error {
	if apiErr, ok := err.(*googleapi.Error); ok {
		log.Printf("[docs] client: Google API error HTTP %d — %s (details: %v)", apiErr.Code, apiErr.Message, apiErr.Details)
		switch apiErr.Code {
		case 400:
			return fmt.Errorf("unsupported document type — the ID must point to a native Google Doc, not a Sheet, Slide, PDF, or uploaded .docx file")
		case 401:
			return fmt.Errorf("google token expired or revoked — re-authenticate via Settings")
		case 403:
			return fmt.Errorf("permission denied — document may not be shared with this account")
		case 404:
			return fmt.Errorf("document not found — check the document ID")
		case 429:
			return fmt.Errorf("Google Docs API rate limit exceeded — try again later")
		}
		return fmt.Errorf("docs api error %d: %s", apiErr.Code, apiErr.Message)
	}
	log.Printf("[docs] client: non-API error: %v", err)
	return err
}
