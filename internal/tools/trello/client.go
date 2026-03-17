// Package trello provides a low-level Trello API client authenticated via query params.
package trello

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

const trelloBaseURL = "https://api.trello.com/1"

// Client wraps an HTTP client for making authenticated Trello API requests.
// Auth is passed as query parameters on every request.
type Client struct {
	apiKey     string
	apiToken   string
	httpClient *http.Client
}

// NewClient creates a new Trello API client using the supplied API key and token.
func NewClient(apiKey, apiToken string) *Client {
	return &Client{
		apiKey:     apiKey,
		apiToken:   apiToken,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

// ListBoards returns the authenticated user's open boards.
// Each entry contains: id, name, desc, url.
func (c *Client) ListBoards(ctx context.Context) ([]map[string]interface{}, error) {
	log.Printf("[trello] client: ListBoards")

	url := trelloBaseURL + "/members/me/boards?fields=id,name,desc,url&filter=open"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("trello: build request: %w", err)
	}
	c.appendAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("trello: http request: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("trello: read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("[trello] client: ListBoards HTTP %d — body=%s", resp.StatusCode, string(data))
		return nil, mapTrelloError(resp.StatusCode)
	}

	var boards []map[string]interface{}
	if err := json.Unmarshal(data, &boards); err != nil {
		return nil, fmt.Errorf("trello: parse response: %w", err)
	}

	log.Printf("[trello] client: ListBoards OK — count=%d", len(boards))
	return boards, nil
}

// GetCard retrieves a card by ID and enriches the result with the list name.
// Returns a map with keys: id, name, description, due_date, labels, list_id, list_name, board_id, url.
func (c *Client) GetCard(ctx context.Context, cardID string) (map[string]interface{}, error) {
	log.Printf("[trello] client: GetCard — cardID=%q", cardID)

	raw, err := c.doRequest(ctx, http.MethodGet, "/cards/"+cardID+"?fields=id,name,desc,due,labels,idList,idBoard,url", nil)
	if err != nil {
		log.Printf("[trello] client: GetCard GET /cards/%s failed: %v", cardID, err)
		return nil, err
	}

	listID, _ := raw["idList"].(string)
	boardID, _ := raw["idBoard"].(string)
	cardURL, _ := raw["url"].(string)
	name, _ := raw["name"].(string)
	desc, _ := raw["desc"].(string)
	due, _ := raw["due"].(string)

	// Extract label names.
	var labelNames []string
	if labelsRaw, ok := raw["labels"].([]interface{}); ok {
		for _, l := range labelsRaw {
			if labelMap, ok := l.(map[string]interface{}); ok {
				if lname, ok := labelMap["name"].(string); ok && lname != "" {
					labelNames = append(labelNames, lname)
				}
			}
		}
	}
	if labelNames == nil {
		labelNames = []string{}
	}

	// Fetch the list name.
	var listName string
	if listID != "" {
		listData, err := c.doRequest(ctx, http.MethodGet, "/lists/"+listID+"?fields=name", nil)
		if err != nil {
			log.Printf("[trello] client: GetCard GET /lists/%s failed (non-fatal): %v", listID, err)
		} else {
			listName, _ = listData["name"].(string)
		}
	}

	result := map[string]interface{}{
		"id":          cardID,
		"name":        name,
		"description": desc,
		"due_date":    due,
		"labels":      labelNames,
		"list_id":     listID,
		"list_name":   listName,
		"board_id":    boardID,
		"url":         cardURL,
	}

	log.Printf("[trello] client: GetCard OK — name=%q listName=%q", name, listName)
	return result, nil
}

// CreateCard creates a new card in the given list.
// Returns (cardID, url, error).
func (c *Client) CreateCard(ctx context.Context, listID, name, description, dueDate string, labels []string) (string, string, error) {
	log.Printf("[trello] client: CreateCard — listID=%q name=%q", listID, name)

	body := map[string]interface{}{
		"idList": listID,
		"name":   name,
	}
	if description != "" {
		body["desc"] = description
	}
	if dueDate != "" {
		body["due"] = dueDate
	}
	if len(labels) > 0 {
		body["idLabels"] = labels
	}

	resp, err := c.doRequest(ctx, http.MethodPost, "/cards", body)
	if err != nil {
		log.Printf("[trello] client: CreateCard POST /cards failed: %v", err)
		return "", "", err
	}

	cardID, _ := resp["id"].(string)
	cardURL, _ := resp["url"].(string)
	log.Printf("[trello] client: CreateCard OK — cardID=%q url=%q", cardID, cardURL)
	return cardID, cardURL, nil
}

// MoveCard moves a card to a different list.
func (c *Client) MoveCard(ctx context.Context, cardID, listID string) error {
	log.Printf("[trello] client: MoveCard — cardID=%q listID=%q", cardID, listID)

	_, err := c.doRequest(ctx, http.MethodPut, "/cards/"+cardID, map[string]interface{}{
		"idList": listID,
	})
	if err != nil {
		log.Printf("[trello] client: MoveCard PUT /cards/%s failed: %v", cardID, err)
		return err
	}

	log.Printf("[trello] client: MoveCard OK")
	return nil
}

// doRequest performs an authenticated HTTP request against the Trello API.
// Auth query params (key, token) are appended to every URL.
// For POST/PUT, body is JSON-encoded and Content-Type is set accordingly.
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) (map[string]interface{}, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("trello: marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, trelloBaseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("trello: build request: %w", err)
	}
	c.appendAuth(req)

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("trello: http request: %w", err)
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("trello: read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("[trello] client: HTTP %d — path=%s body=%s", resp.StatusCode, path, string(respData))
		return nil, mapTrelloError(resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respData, &result); err != nil {
		return nil, fmt.Errorf("trello: parse response: %w", err)
	}

	return result, nil
}

// appendAuth adds the Trello API key and token as query parameters to the request URL.
func (c *Client) appendAuth(req *http.Request) {
	q := req.URL.Query()
	q.Set("key", c.apiKey)
	q.Set("token", c.apiToken)
	req.URL.RawQuery = q.Encode()
}

// mapTrelloError converts well-known Trello HTTP status codes into descriptive errors.
func mapTrelloError(statusCode int) error {
	switch statusCode {
	case http.StatusBadRequest:
		return fmt.Errorf("invalid request — check parameters")
	case http.StatusUnauthorized:
		return fmt.Errorf("trello auth failed — check api_key and api_token in config")
	case http.StatusNotFound:
		return fmt.Errorf("board, list, or card not found — check the ID")
	case http.StatusTooManyRequests:
		return fmt.Errorf("Trello API rate limit exceeded — try again later")
	default:
		return fmt.Errorf("trello api error: HTTP %d", statusCode)
	}
}
