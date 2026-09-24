package websearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"

	"bruce/internal/ai"
	"bruce/internal/config"
	"bruce/internal/logging"
)

// SearchResult represents a single search result.
type SearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

// SearchTool implements the tools.Tool interface for internet searching.
type SearchTool struct {
	cfg        *config.Config
	httpClient *http.Client
}

// NewSearchTool creates a new SearchTool.
func NewSearchTool(cfg *config.Config) *SearchTool {
	return &SearchTool{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

// Name returns the tool name.
func (t *SearchTool) Name() string { return "web_search" }

// Definition returns the tool definition for the AI model.
func (t *SearchTool) Definition() ai.ToolDefinition {
	return ai.ToolDefinition{
		Name:        "web_search",
		Description: "Search the web for up-to-date information, news, technical documentation, or real-world facts. Returns a list of search results with titles, snippets, and source URLs.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "The search query to look up on the internet",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of search results to return (default 5, max 10)",
				},
			},
			"required": []string{"query"},
		},
	}
}

// Execute performs the web search.
func (t *SearchTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	query, ok := input["query"].(string)
	if !ok || strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("query is required")
	}
	query = strings.TrimSpace(query)

	limit := 5
	if l, ok := input["limit"]; ok {
		switch n := l.(type) {
		case float64:
			limit = int(n)
		case int:
			limit = n
		case int64:
			limit = int(n)
		}
	}
	if limit <= 0 {
		limit = 5
	}
	if limit > 10 {
		limit = 10
	}

	provider := "duckduckgo"
	if t.cfg != nil && t.cfg.Tools.WebSearch.Provider != "" {
		provider = strings.ToLower(t.cfg.Tools.WebSearch.Provider)
	}

	var results []SearchResult
	var err error

	switch provider {
	case "brave":
		apiKey := ""
		if t.cfg != nil {
			apiKey = t.cfg.Tools.WebSearch.APIKey
		}
		if apiKey != "" {
			results, err = t.searchBrave(ctx, query, limit, apiKey)
		} else {
			logging.Warn("Brave provider configured without api_key, falling back to DuckDuckGo")
			results, err = t.searchDuckDuckGo(ctx, query, limit)
		}
	case "tavily":
		apiKey := ""
		if t.cfg != nil {
			apiKey = t.cfg.Tools.WebSearch.APIKey
		}
		if apiKey != "" {
			results, err = t.searchTavily(ctx, query, limit, apiKey)
		} else {
			logging.Warn("Tavily provider configured without api_key, falling back to DuckDuckGo")
			results, err = t.searchDuckDuckGo(ctx, query, limit)
		}
	default:
		results, err = t.searchDuckDuckGo(ctx, query, limit)
	}

	if err != nil {
		logging.Errorf("web_search failed for %q: %v", query, err)
		return nil, fmt.Errorf("search failed: %w", err)
	}

	data, err := json.Marshal(results)
	if err != nil {
		return nil, fmt.Errorf("marshal results: %w", err)
	}

	return string(data), nil
}

// searchDuckDuckGo performs a search via DuckDuckGo Lite with fallback to Instant Answer API.
func (t *SearchTool) searchDuckDuckGo(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	// 1. Try DuckDuckGo Lite POST
	formData := url.Values{}
	formData.Set("q", query)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://lite.duckduckgo.com/lite/", strings.NewReader(formData.Encode()))
	if err == nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
		req.Header.Set("Referer", "https://lite.duckduckgo.com/")

		resp, err := t.httpClient.Do(req)
		if err == nil {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				bodyBytes, _ := io.ReadAll(resp.Body)
				results := parseDuckDuckGoLite(bodyBytes, limit)
				if len(results) > 0 {
					return results, nil
				}
			}
		}
	}

	// 2. Fallback: DuckDuckGo Instant Answer API
	return t.searchDuckDuckGoInstant(ctx, query, limit)
}

func parseDuckDuckGoLite(body []byte, limit int) []SearchResult {
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return nil
	}

	var results []SearchResult
	var currentResult *SearchResult

	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n == nil || len(results) >= limit {
			return
		}

		if n.Type == html.ElementNode {
			// Check for <a class="result-link" href="...">Title</a>
			if n.Data == "a" {
				isResultLink := false
				var href string
				for _, attr := range n.Attr {
					if attr.Key == "class" && strings.Contains(attr.Val, "result-link") {
						isResultLink = true
					}
					if attr.Key == "href" {
						href = attr.Val
					}
				}
				if isResultLink && href != "" {
					title := strings.TrimSpace(getTextContent(n))
					if title != "" {
						currentResult = &SearchResult{
							Title: title,
							URL:   href,
						}
					}
				}
			}

			// Check for <td class="result-snippet">Snippet</td>
			if n.Data == "td" {
				for _, attr := range n.Attr {
					if attr.Key == "class" && strings.Contains(attr.Val, "result-snippet") {
						snippet := strings.TrimSpace(getTextContent(n))
						if currentResult != nil && currentResult.Title != "" {
							currentResult.Snippet = snippet
							results = append(results, *currentResult)
							currentResult = nil
						}
					}
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}

	traverse(doc)
	return results
}

func (t *SearchTool) searchDuckDuckGoInstant(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	apiURL := fmt.Sprintf("https://api.duckduckgo.com/?q=%s&format=json&no_html=1&skip_disambig=0", url.QueryEscape(query))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "BruceAI/1.0")

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("duckduckgo api status %d", resp.StatusCode)
	}

	var data struct {
		Heading       string `json:"Heading"`
		Abstract      string `json:"Abstract"`
		AbstractURL   string `json:"AbstractURL"`
		RelatedTopics []struct {
			Text     string `json:"Text"`
			FirstURL string `json:"FirstURL"`
			Topics   []struct {
				Text     string `json:"Text"`
				FirstURL string `json:"FirstURL"`
			} `json:"Topics"`
		} `json:"RelatedTopics"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	var results []SearchResult
	if data.Abstract != "" && data.AbstractURL != "" {
		results = append(results, SearchResult{
			Title:   data.Heading,
			URL:     data.AbstractURL,
			Snippet: data.Abstract,
		})
	}

	for _, rt := range data.RelatedTopics {
		if len(results) >= limit {
			break
		}
		if rt.Text != "" && rt.FirstURL != "" {
			results = append(results, SearchResult{
				Title:   rt.Text,
				URL:     rt.FirstURL,
				Snippet: rt.Text,
			})
		}
		for _, sub := range rt.Topics {
			if len(results) >= limit {
				break
			}
			if sub.Text != "" && sub.FirstURL != "" {
				results = append(results, SearchResult{
					Title:   sub.Text,
					URL:     sub.FirstURL,
					Snippet: sub.Text,
				})
			}
		}
	}

	return results, nil
}

func (t *SearchTool) searchBrave(ctx context.Context, query string, limit int, apiKey string) ([]SearchResult, error) {
	reqURL := fmt.Sprintf("https://api.search.brave.com/res/v1/web/search?q=%s&count=%d", url.QueryEscape(query), limit)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Subscription-Token", apiKey)

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("brave search error (status %d)", resp.StatusCode)
	}

	var data struct {
		Web struct {
			Results []struct {
				Title       string `json:"title"`
				URL         string `json:"url"`
				Description string `json:"description"`
			} `json:"results"`
		} `json:"web"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	results := make([]SearchResult, 0, len(data.Web.Results))
	for _, r := range data.Web.Results {
		results = append(results, SearchResult{
			Title:   r.Title,
			URL:     r.URL,
			Snippet: r.Description,
		})
	}
	return results, nil
}

func (t *SearchTool) searchTavily(ctx context.Context, query string, limit int, apiKey string) ([]SearchResult, error) {
	payload := map[string]interface{}{
		"api_key": apiKey,
		"query":   query,
		"max_results": limit,
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.tavily.com/search", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tavily search error (status %d)", resp.StatusCode)
	}

	var data struct {
		Results []struct {
			Title   string `json:"title"`
			URL     string `json:"url"`
			Content string `json:"content"`
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	results := make([]SearchResult, 0, len(data.Results))
	for _, r := range data.Results {
		results = append(results, SearchResult{
			Title:   r.Title,
			URL:     r.URL,
			Snippet: r.Content,
		})
	}
	return results, nil
}
