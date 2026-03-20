// Package httpclient provides a tools.Tool implementation for making arbitrary
// outbound HTTP requests with SSRF protection.
package httpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"bruce/internal/ai"
	"bruce/internal/config"
)

// privateCIDRs is the list of CIDR blocks considered private/loopback.
// Requests to these addresses are blocked unless AllowPrivateNetworks is true.
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

// isPrivateIP returns true when the given IP falls within one of the private
// CIDR ranges.
func isPrivateIP(ip net.IP) bool {
	for _, block := range privateCIDRs {
		if block.Contains(ip) {
			return true
		}
	}
	return false
}

// HTTPRequestTool implements tools.Tool for making outbound HTTP requests.
type HTTPRequestTool struct {
	cfg config.HTTPClientConfig
}

// NewHTTPRequestTool creates a new HTTPRequestTool with the given config.
func NewHTTPRequestTool(cfg config.HTTPClientConfig) *HTTPRequestTool {
	return &HTTPRequestTool{cfg: cfg}
}

// Name returns the tool name.
func (t *HTTPRequestTool) Name() string { return "http_request" }

// Definition returns the AI tool definition for http_request.
func (t *HTTPRequestTool) Definition() ai.ToolDefinition {
	return ai.ToolDefinition{
		Name:        "http_request",
		Description: "Make an outbound HTTP request to any public URL. Supports GET, POST, PUT, PATCH, DELETE with optional headers and body.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"method": map[string]interface{}{
					"type":        "string",
					"description": "HTTP method",
					"enum":        []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
				},
				"url": map[string]interface{}{
					"type":        "string",
					"description": "Full URL to request (must be a public endpoint)",
				},
				"headers": map[string]interface{}{
					"type":                 "object",
					"description":          "Optional HTTP headers to include in the request",
					"additionalProperties": map[string]interface{}{"type": "string"},
				},
				"body": map[string]interface{}{
					"type":        "string",
					"description": "Optional request body (used with POST, PUT, PATCH)",
				},
				"timeout_seconds": map[string]interface{}{
					"type":        "integer",
					"description": "Request timeout in seconds (default 30, max determined by config)",
				},
			},
			"required": []string{"method", "url"},
		},
	}
}

// HTTPResponse is the structured response returned by the http_request tool.
type HTTPResponse struct {
	StatusCode int                 `json:"status_code"`
	Body       string              `json:"body"`
	Headers    map[string][]string `json:"headers"`
	LatencyMs  int64               `json:"latency_ms"`
	Truncated  bool                `json:"truncated"`
}

// Execute performs the HTTP request and returns the response as an HTTPResponse.
func (t *HTTPRequestTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	// Validate method.
	method, ok := input["method"].(string)
	if !ok || method == "" {
		return nil, fmt.Errorf("method is required")
	}
	method = strings.ToUpper(method)
	validMethods := map[string]bool{"GET": true, "POST": true, "PUT": true, "PATCH": true, "DELETE": true}
	if !validMethods[method] {
		return nil, fmt.Errorf("method must be one of GET, POST, PUT, PATCH, DELETE; got %q", method)
	}

	// Validate URL.
	rawURL, ok := input["url"].(string)
	if !ok || rawURL == "" {
		return nil, fmt.Errorf("url is required")
	}

	// Determine timeout.
	timeoutSec := 30
	if v, ok := input["timeout_seconds"]; ok {
		switch n := v.(type) {
		case float64:
			timeoutSec = int(n)
		case int:
			timeoutSec = n
		case int64:
			timeoutSec = int(n)
		}
	}
	maxTimeout := t.cfg.MaxTimeoutSeconds
	if maxTimeout <= 0 {
		maxTimeout = 120
	}
	if timeoutSec > maxTimeout {
		timeoutSec = maxTimeout
	}
	if timeoutSec <= 0 {
		timeoutSec = 30
	}

	// Create child context with this tool's own timeout.
	reqCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
	defer cancel()

	// Build optional body.
	var bodyReader io.Reader
	if bodyStr, ok := input["body"].(string); ok && bodyStr != "" {
		bodyReader = strings.NewReader(bodyStr)
	}

	// Build request.
	req, err := http.NewRequestWithContext(reqCtx, method, rawURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	// Apply optional headers.
	if hdrs, ok := input["headers"].(map[string]interface{}); ok {
		for k, v := range hdrs {
			if s, ok := v.(string); ok {
				req.Header.Set(k, s)
			}
		}
	}

	// Build SSRF-aware HTTP client.
	allowPrivate := t.cfg.AllowPrivateNetworks
	dialer := &net.Dialer{Timeout: time.Duration(timeoutSec) * time.Second}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, fmt.Errorf("parse addr %q: %w", addr, err)
			}

			if !allowPrivate {
				ips, err := net.LookupHost(host)
				if err != nil {
					return nil, fmt.Errorf("dns lookup %q: %w", host, err)
				}
				for _, ipStr := range ips {
					ip := net.ParseIP(ipStr)
					if ip != nil && isPrivateIP(ip) {
						return nil, fmt.Errorf("SSRF blocked: %q resolves to private IP %s", host, ipStr)
					}
				}

				// Dial to the first resolved IP directly (prevents DNS rebinding).
				if len(ips) > 0 {
					directAddr := net.JoinHostPort(ips[0], port)
					return dialer.DialContext(ctx, network, directAddr)
				}
			}

			return dialer.DialContext(ctx, network, addr)
		},
	}

	httpClient := &http.Client{
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if t.cfg.FollowRedirects {
				if len(via) >= 10 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			}
			return http.ErrUseLastResponse
		},
	}

	start := time.Now()
	resp, err := httpClient.Do(req)
	latencyMs := time.Since(start).Milliseconds()
	if err != nil {
		log.Printf("[httpclient] http_request: error calling %s %s: %v", method, rawURL, err)
		return nil, fmt.Errorf("http_request: %w", err)
	}
	defer resp.Body.Close()

	// Read response up to MaxResponseBytes.
	maxBytes := t.cfg.MaxResponseBytes
	if maxBytes <= 0 {
		maxBytes = 524288 // 512 KB default
	}
	limitedReader := io.LimitReader(resp.Body, int64(maxBytes)+1)
	bodyBytes, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	truncated := false
	if len(bodyBytes) > maxBytes {
		bodyBytes = bodyBytes[:maxBytes]
		truncated = true
	}

	result := HTTPResponse{
		StatusCode: resp.StatusCode,
		Body:       string(bodyBytes),
		Headers:    map[string][]string(resp.Header),
		LatencyMs:  latencyMs,
		Truncated:  truncated,
	}

	log.Printf("[httpclient] http_request: %s %s → %d (%dms, truncated=%v)", method, rawURL, resp.StatusCode, latencyMs, truncated)

	data, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("marshal response: %w", err)
	}

	return string(data), nil
}
