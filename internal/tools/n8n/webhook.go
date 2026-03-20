package n8n

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"time"

	"bruce/internal/ai"
	"bruce/internal/config"
)

// WebhookTool implements tools.Tool for calling n8n webhooks.
type WebhookTool struct {
	client *N8nClient
	cfg    config.N8nConfig
}

// NewWebhookTool creates a new WebhookTool.
func NewWebhookTool(client *N8nClient, cfg config.N8nConfig) *WebhookTool {
	return &WebhookTool{client: client, cfg: cfg}
}

// Name returns the tool name.
func (t *WebhookTool) Name() string { return "n8n_webhook_call" }

// Definition returns the AI tool definition for n8n_webhook_call.
func (t *WebhookTool) Definition() ai.ToolDefinition {
	return ai.ToolDefinition{
		Name:        "n8n_webhook_call",
		Description: "Trigger an n8n workflow by calling its webhook endpoint. Sends a POST request to the configured n8n instance.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"webhook_path": map[string]interface{}{
					"type":        "string",
					"description": "Webhook path registered in n8n (e.g. 'my-workflow' for /webhook/my-workflow)",
				},
				"payload": map[string]interface{}{
					"type":        "object",
					"description": "Optional JSON payload to send as the request body",
				},
				"timeout_seconds": map[string]interface{}{
					"type":        "integer",
					"description": "Request timeout in seconds (default uses webhook_timeout_seconds from config, max 120)",
				},
			},
			"required": []string{"webhook_path"},
		},
	}
}

// WebhookResponse is the structured response returned by n8n_webhook_call.
type WebhookResponse struct {
	StatusCode  int         `json:"status_code"`
	Response    interface{} `json:"response,omitempty"`
	RawResponse *string     `json:"raw_response,omitempty"`
	LatencyMs   int64       `json:"latency_ms"`
}

// Execute calls the n8n webhook and returns the structured response.
func (t *WebhookTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	// Extract webhook_path (required).
	webhookPath, ok := input["webhook_path"].(string)
	if !ok || webhookPath == "" {
		return nil, fmt.Errorf("webhook_path is required and must be a non-empty string")
	}

	// Determine timeout.
	timeoutSec := t.cfg.WebhookTimeoutSeconds
	if timeoutSec <= 0 {
		timeoutSec = 30
	}
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
	if timeoutSec > 120 {
		timeoutSec = 120
	}
	if timeoutSec <= 0 {
		timeoutSec = 30
	}

	reqCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
	defer cancel()

	// Build body.
	var bodyBuf *bytes.Buffer
	if payload, ok := input["payload"].(map[string]interface{}); ok && payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("marshal payload: %w", err)
		}
		bodyBuf = bytes.NewBuffer(data)
	} else {
		bodyBuf = bytes.NewBufferString("{}")
	}

	path := "/webhook/" + webhookPath
	log.Printf("[n8n] n8n_webhook_call: POST %s", path)

	start := time.Now()
	resp, err := t.client.Do(reqCtx, "POST", path, bodyBuf)
	latencyMs := time.Since(start).Milliseconds()
	if err != nil {
		return nil, fmt.Errorf("n8n_webhook_call: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("n8n_webhook_call: read response body: %w", err)
	}
	bodyStr := string(bodyBytes)

	// Map error status codes to friendly messages.
	switch resp.StatusCode {
	case 404:
		return nil, fmt.Errorf("n8n_webhook_call: webhook not registered (404)")
	case 401, 403:
		return nil, fmt.Errorf("n8n_webhook_call: auth failure (%d)", resp.StatusCode)
	}

	log.Printf("[n8n] n8n_webhook_call: %s → %d (%dms)", path, resp.StatusCode, latencyMs)

	result := WebhookResponse{
		StatusCode: resp.StatusCode,
		LatencyMs:  latencyMs,
	}

	// Try to parse as JSON; fall back to raw string.
	var parsed interface{}
	if err := json.Unmarshal(bodyBytes, &parsed); err == nil {
		result.Response = parsed
	} else {
		result.RawResponse = &bodyStr
	}

	data, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("n8n_webhook_call: marshal response: %w", err)
	}

	return string(data), nil
}
