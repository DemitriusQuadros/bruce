package websearch

import (
	"bytes"
	"fmt"
	"strings"

	"golang.org/x/net/html"
)

// ExtractedPage holds the cleaned content of an HTML page.
type ExtractedPage struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Content     string `json:"content"`
	Length      int    `json:"length"`
	Truncated   bool   `json:"truncated"`
}

var ignoredTags = map[string]bool{
	"script":   true,
	"style":    true,
	"noscript": true,
	"iframe":   true,
	"svg":      true,
	"canvas":   true,
	"nav":      true,
	"footer":   true,
	"aside":    true,
	"header":   true,
	"head":     false, // We need head to extract title and meta description
}

// CleanHTML parses raw HTML, extracts metadata, and converts the main content into clean text.
func CleanHTML(rawHTML []byte, maxChars int) (*ExtractedPage, error) {
	if maxChars <= 0 {
		maxChars = 15000
	}

	doc, err := html.Parse(bytes.NewReader(rawHTML))
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	page := &ExtractedPage{}
	var textBuilder strings.Builder

	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n == nil {
			return
		}

		if n.Type == html.ElementNode {
			tag := strings.ToLower(n.Data)

			// Extract title
			if tag == "title" && page.Title == "" {
				page.Title = strings.TrimSpace(getTextContent(n))
				return
			}

			// Extract meta description
			if tag == "meta" {
				var name, content string
				for _, attr := range n.Attr {
					k := strings.ToLower(attr.Key)
					v := strings.ToLower(attr.Val)
					if k == "name" && (v == "description" || v == "twitter:description") {
						name = "desc"
					} else if k == "property" && v == "og:description" {
						name = "desc"
					} else if k == "content" {
						content = attr.Val
					}
				}
				if name == "desc" && content != "" && page.Description == "" {
					page.Description = strings.TrimSpace(content)
				}
				return
			}

			// Skip ignored tags
			if ignored, ok := ignoredTags[tag]; ok && ignored {
				return
			}

			// Add markdown-like header prefixes
			switch tag {
			case "h1":
				textBuilder.WriteString("\n\n# ")
			case "h2":
				textBuilder.WriteString("\n\n## ")
			case "h3":
				textBuilder.WriteString("\n\n### ")
			case "h4", "h5", "h6":
				textBuilder.WriteString("\n\n#### ")
			case "p", "div", "article", "section", "blockquote":
				textBuilder.WriteString("\n")
			case "li":
				textBuilder.WriteString("\n- ")
			case "tr":
				textBuilder.WriteString("\n")
			case "br":
				textBuilder.WriteString("\n")
			}
		}

		if n.Type == html.TextNode {
			text := strings.TrimSpace(n.Data)
			if text != "" {
				textBuilder.WriteString(text + " ")
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}

		if n.Type == html.ElementNode {
			tag := strings.ToLower(n.Data)
			switch tag {
			case "p", "h1", "h2", "h3", "h4", "h5", "h6", "div", "article", "section", "blockquote", "tr":
				textBuilder.WriteString("\n")
			}
		}
	}

	traverse(doc)

	// Clean up consecutive blank lines
	cleanedText := collapseWhitespace(textBuilder.String())

	if len(cleanedText) > maxChars {
		cleanedText = cleanedText[:maxChars]
		page.Truncated = true
	}

	page.Content = cleanedText
	page.Length = len(cleanedText)

	return page, nil
}

func getTextContent(n *html.Node) string {
	var b strings.Builder
	var f func(*html.Node)
	f = func(node *html.Node) {
		if node.Type == html.TextNode {
			b.WriteString(node.Data)
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
	return b.String()
}

func collapseWhitespace(s string) string {
	lines := strings.Split(s, "\n")
	var result []string
	consecutiveBlank := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			consecutiveBlank++
			if consecutiveBlank <= 1 && len(result) > 0 {
				result = append(result, "")
			}
		} else {
			consecutiveBlank = 0
			result = append(result, trimmed)
		}
	}

	return strings.TrimSpace(strings.Join(result, "\n"))
}
