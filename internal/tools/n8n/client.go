// Package n8n provides tools.Tool implementations for n8n workflow automation.
package n8n

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"time"

	"bruce/internal/config"
)

// N8nClient wraps *http.Client with n8n-specific auth injection.
type N8nClient struct {
	baseURL     string
	httpClient  *http.Client
	authMethod  string
	username    string
	password    string
	headerName  string
	headerValue string
}

// NewN8nClient creates a new N8nClient from config.
func NewN8nClient(cfg config.N8nConfig) *N8nClient {
	timeout := time.Duration(cfg.WebhookTimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	return &N8nClient{
		baseURL:     cfg.BaseURL,
		httpClient:  &http.Client{Timeout: timeout},
		authMethod:  cfg.WebhookAuth.Method,
		username:    cfg.WebhookAuth.Username,
		password:    cfg.WebhookAuth.Password,
		headerName:  cfg.WebhookAuth.HeaderName,
		headerValue: cfg.WebhookAuth.HeaderValue,
	}
}

// Do executes an HTTP request against the n8n instance.
// The path is appended to the configured baseURL and auth headers are injected
// according to the configured method (none | basic | header).
func (c *N8nClient) Do(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	url := c.baseURL + path

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("n8n client build request: %w", err)
	}

	// Inject auth.
	switch c.authMethod {
	case "basic":
		encoded := base64.StdEncoding.EncodeToString([]byte(c.username + ":" + c.password))
		req.Header.Set("Authorization", "Basic "+encoded)
	case "header":
		if c.headerName != "" {
			req.Header.Set(c.headerName, c.headerValue)
		}
		// "none" and anything else: no auth header.
	}

	return c.httpClient.Do(req)
}
