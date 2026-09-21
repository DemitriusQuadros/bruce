package websearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"bruce/internal/ai"
	"bruce/internal/config"
	"bruce/internal/logging"
)

// FetchTool implements tools.Tool for fetching and reading web page content.
type FetchTool struct {
	cfg        *config.Config
	httpClient *http.Client
}

// NewFetchTool creates a new FetchTool.
func NewFetchTool(cfg *config.Config) *FetchTool {
	return &FetchTool{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: 25 * time.Second},
	}
}

// Name returns the tool name.
func (t *FetchTool) Name() string { return "web_fetch" }

// Definition returns the tool definition for the AI model.
func (t *FetchTool) Definition() ai.ToolDefinition {
	return ai.ToolDefinition{
		Name:        "web_fetch",
		Description: "Fetch and read the main content of a public web page or document. Strips ads, scripts, navigation menus, and boilerplate HTML, returning clean structured text. Use this after web_search to inspect source articles or documentation.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"url": map[string]interface{}{
					"type":        "string",
					"description": "Full HTTP or HTTPS URL to fetch and read",
				},
				"max_chars": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of characters of text to return (default 15000, max 50000)",
				},
			},
			"required": []string{"url"},
		},
	}
}

// FetchResponse represents the structured response returned by web_fetch.
type FetchResponse struct {
	URL         string `json:"url"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Content     string `json:"content"`
	Length      int    `json:"length"`
	Truncated   bool   `json:"truncated"`
}

// Execute fetches the page and returns clean extracted text.
func (t *FetchTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	rawURL, ok := input["url"].(string)
	if !ok || strings.TrimSpace(rawURL) == "" {
		return nil, fmt.Errorf("url is required")
	}
	rawURL = strings.TrimSpace(rawURL)

	parsedURL, err := url.Parse(rawURL)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return nil, fmt.Errorf("invalid URL scheme: must be http or https")
	}

	maxChars := 15000
	if m, ok := input["max_chars"]; ok {
		switch n := m.(type) {
		case float64:
			maxChars = int(n)
		case int:
			maxChars = n
		case int64:
			maxChars = int(n)
		}
	}
	if maxChars <= 0 {
		maxChars = 15000
	}
	if maxChars > 50000 {
		maxChars = 50000
	}

	// SSRF Protection: Prevent accessing internal private networks
	if err := validatePublicHost(parsedURL.Hostname()); err != nil {
		return nil, fmt.Errorf("SSRF blocked: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 BruceAI/1.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,text/plain;q=0.8,*/*;q=0.5")

	resp, err := t.httpClient.Do(req)
	if err != nil {
		logging.Warnf("web_fetch error calling %s: %v", rawURL, err)
		return nil, fmt.Errorf("fetch page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch error: HTTP %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	// Read up to 2MB raw response
	limitedReader := io.LimitReader(resp.Body, 2*1024*1024)
	bodyBytes, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	contentType := strings.ToLower(resp.Header.Get("Content-Type"))

	var result *FetchResponse
	if strings.Contains(contentType, "html") || bytes.Contains(bodyBytes[:min(512, len(bodyBytes))], []byte("<html")) {
		page, err := CleanHTML(bodyBytes, maxChars)
		if err != nil {
			return nil, fmt.Errorf("clean html: %w", err)
		}
		result = &FetchResponse{
			URL:         rawURL,
			Title:       page.Title,
			Description: page.Description,
			Content:     page.Content,
			Length:      page.Length,
			Truncated:   page.Truncated,
		}
	} else {
		// Plain text / json / markdown
		text := string(bodyBytes)
		truncated := false
		if len(text) > maxChars {
			text = text[:maxChars]
			truncated = true
		}
		result = &FetchResponse{
			URL:       rawURL,
			Content:   strings.TrimSpace(text),
			Length:    len(text),
			Truncated: truncated,
		}
	}

	data, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("marshal response: %w", err)
	}

	return string(data), nil
}

var privateCIDRs []*net.IPNet

func init() {
	blocks := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
		"::1/128",
		"169.254.0.0/16",
		"fd00::/8",
	}
	for _, cidr := range blocks {
		_, network, err := net.ParseCIDR(cidr)
		if err == nil {
			privateCIDRs = append(privateCIDRs, network)
		}
	}
}

func validatePublicHost(host string) error {
	ip := net.ParseIP(host)
	if ip != nil {
		for _, block := range privateCIDRs {
			if block.Contains(ip) {
				return fmt.Errorf("ip %s is private/loopback", host)
			}
		}
		return nil
	}

	ips, err := net.LookupHost(host)
	if err != nil {
		return fmt.Errorf("dns lookup failed: %w", err)
	}
	for _, ipStr := range ips {
		parsed := net.ParseIP(ipStr)
		if parsed != nil {
			for _, block := range privateCIDRs {
				if block.Contains(parsed) {
					return fmt.Errorf("resolved host %s (%s) is private/loopback", host, ipStr)
				}
			}
		}
	}
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
