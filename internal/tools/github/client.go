// Package github provides a low-level GitHub REST API client authenticated via a Bearer token.
package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const (
	githubBaseURL    = "https://api.github.com"
	githubAPIVersion = "2022-11-28"
)

// Client wraps an HTTP client for making authenticated GitHub API requests.
type Client struct {
	token      string
	httpClient *http.Client
}

// NewClient creates a new GitHub API client using the supplied Bearer token.
func NewClient(token string) *Client {
	return &Client{
		token:      token,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

// ListBranches returns up to perPage branches for the given owner/repo.
func (c *Client) ListBranches(ctx context.Context, owner, repo string, perPage int) ([]map[string]interface{}, error) {
	log.Printf("[github] client: ListBranches — owner=%q repo=%q perPage=%d", owner, repo, perPage)

	url := fmt.Sprintf("%s/repos/%s/%s/branches?per_page=%d", githubBaseURL, owner, repo, perPage)
	result, err := c.doRequestArray(ctx, http.MethodGet, url, nil)
	if err != nil {
		log.Printf("[github] client: ListBranches failed: %v", err)
		return nil, err
	}

	log.Printf("[github] client: ListBranches OK — count=%d", len(result))
	return result, nil
}

// ListIssues returns up to perPage issues for the given owner/repo filtered by state.
// Pull requests are excluded via the pulls=false query parameter.
func (c *Client) ListIssues(ctx context.Context, owner, repo, state string, perPage int) ([]map[string]interface{}, error) {
	log.Printf("[github] client: ListIssues — owner=%q repo=%q state=%q perPage=%d", owner, repo, state, perPage)

	url := fmt.Sprintf("%s/repos/%s/%s/issues?state=%s&per_page=%d&pulls=false", githubBaseURL, owner, repo, state, perPage)
	result, err := c.doRequestArray(ctx, http.MethodGet, url, nil)
	if err != nil {
		log.Printf("[github] client: ListIssues failed: %v", err)
		return nil, err
	}

	log.Printf("[github] client: ListIssues OK — count=%d", len(result))
	return result, nil
}

// ListPRs returns all pull requests for the given owner/repo filtered by state.
func (c *Client) ListPRs(ctx context.Context, owner, repo, state string) ([]map[string]interface{}, error) {
	log.Printf("[github] client: ListPRs — owner=%q repo=%q state=%q", owner, repo, state)

	url := fmt.Sprintf("%s/repos/%s/%s/pulls?state=%s", githubBaseURL, owner, repo, state)
	result, err := c.doRequestArray(ctx, http.MethodGet, url, nil)
	if err != nil {
		log.Printf("[github] client: ListPRs failed: %v", err)
		return nil, err
	}

	log.Printf("[github] client: ListPRs OK — count=%d", len(result))
	return result, nil
}

// CreateIssue creates a new issue in the given owner/repo.
// Returns (number, url, error).
func (c *Client) CreateIssue(ctx context.Context, owner, repo, title, body string, labels []string) (int, string, error) {
	log.Printf("[github] client: CreateIssue — owner=%q repo=%q title=%q", owner, repo, title)

	reqBody := map[string]interface{}{
		"title": title,
	}
	if body != "" {
		reqBody["body"] = body
	}
	if len(labels) > 0 {
		reqBody["labels"] = labels
	}

	url := fmt.Sprintf("%s/repos/%s/%s/issues", githubBaseURL, owner, repo)
	resp, err := c.doRequest(ctx, http.MethodPost, url, reqBody)
	if err != nil {
		log.Printf("[github] client: CreateIssue failed: %v", err)
		return 0, "", err
	}

	numberFloat, _ := resp["number"].(float64)
	number := int(numberFloat)
	issueURL, _ := resp["html_url"].(string)

	log.Printf("[github] client: CreateIssue OK — number=%d url=%q", number, issueURL)
	return number, issueURL, nil
}

// doRequest performs an authenticated HTTP request against the GitHub API
// and returns the parsed JSON response as a map.
func (c *Client) doRequest(ctx context.Context, method, url string, body interface{}) (map[string]interface{}, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("github: marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("github: build request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", githubAPIVersion)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github: http request: %w", err)
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("github: read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("[github] client: HTTP %d — url=%s body=%s", resp.StatusCode, url, string(respData))
		return nil, mapGithubError(resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respData, &result); err != nil {
		return nil, fmt.Errorf("github: parse response: %w", err)
	}

	return result, nil
}

// doRequestArray performs an authenticated HTTP request against the GitHub API
// and returns the parsed JSON response as a slice of maps.
func (c *Client) doRequestArray(ctx context.Context, method, url string, body interface{}) ([]map[string]interface{}, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("github: marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("github: build request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", githubAPIVersion)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github: http request: %w", err)
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("github: read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("[github] client: HTTP %d — url=%s body=%s", resp.StatusCode, url, string(respData))
		return nil, mapGithubError(resp.StatusCode)
	}

	var result []map[string]interface{}
	if err := json.Unmarshal(respData, &result); err != nil {
		return nil, fmt.Errorf("github: parse response: %w", err)
	}

	return result, nil
}

// mapGithubError converts well-known GitHub HTTP status codes into descriptive errors.
func mapGithubError(statusCode int) error {
	switch statusCode {
	case http.StatusUnauthorized:
		return fmt.Errorf("github auth failed — check tools.github.token in config")
	case http.StatusForbidden:
		return fmt.Errorf("github permission denied — check token scope (needs repo, read:user)")
	case http.StatusNotFound:
		return fmt.Errorf("repository not found — check owner/repo")
	case http.StatusUnprocessableEntity:
		return fmt.Errorf("invalid request — check parameters")
	case http.StatusTooManyRequests:
		return fmt.Errorf("GitHub API rate limit exceeded")
	default:
		return fmt.Errorf("github api error: HTTP %d", statusCode)
	}
}
