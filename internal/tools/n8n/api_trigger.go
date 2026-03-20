package n8n

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"bruce/internal/ai"
	"bruce/internal/config"
)

// APITriggerTool implements tools.Tool for triggering n8n workflows via the REST API.
type APITriggerTool struct {
	client *N8nClient
	cfg    config.N8nConfig
}

// NewAPITriggerTool creates a new APITriggerTool.
func NewAPITriggerTool(client *N8nClient, cfg config.N8nConfig) *APITriggerTool {
	return &APITriggerTool{client: client, cfg: cfg}
}

// Name returns the tool name.
func (t *APITriggerTool) Name() string { return "n8n_api_trigger" }

// Definition returns the AI tool definition for n8n_api_trigger.
func (t *APITriggerTool) Definition() ai.ToolDefinition {
	return ai.ToolDefinition{
		Name:        "n8n_api_trigger",
		Description: "Trigger an n8n workflow execution via the n8n REST API using a workflow ID. Requires an API key.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"workflow_id": map[string]interface{}{
					"type":        "string",
					"description": "The n8n workflow ID to execute",
				},
				"data": map[string]interface{}{
					"type":        "object",
					"description": "Optional input data passed to the workflow",
				},
			},
			"required": []string{"workflow_id"},
		},
	}
}

// APITriggerResponse is the structured response returned by n8n_api_trigger.
type APITriggerResponse struct {
	ExecutionID string `json:"execution_id"`
	WorkflowID  string `json:"workflow_id"`
	QueuedAt    string `json:"queued_at"` // RFC3339
}

// Execute triggers the workflow via the n8n REST API.
func (t *APITriggerTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	workflowID, ok := input["workflow_id"].(string)
	if !ok || workflowID == "" {
		return nil, fmt.Errorf("workflow_id is required and must be a non-empty string")
	}

	// Build request body.
	reqBody := map[string]interface{}{
		"workflowData": map[string]interface{}{},
	}
	if data, ok := input["data"].(map[string]interface{}); ok && data != nil {
		reqBody["workflowData"] = data
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("n8n_api_trigger: marshal request body: %w", err)
	}

	path := "/api/v1/workflows/" + workflowID + "/run"
	log.Printf("[n8n] n8n_api_trigger: POST %s", path)

	// Build the request with API key header directly (not through client auth).
	url := t.cfg.BaseURL + path
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("n8n_api_trigger: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-N8N-API-KEY", t.cfg.APIKey)

	start := time.Now()
	resp, err := t.client.httpClient.Do(req)
	latencyMs := time.Since(start).Milliseconds()
	if err != nil {
		return nil, fmt.Errorf("n8n_api_trigger: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("n8n_api_trigger: read response body: %w", err)
	}

	// Map error status codes.
	switch resp.StatusCode {
	case 404:
		return nil, fmt.Errorf("n8n_api_trigger: workflow not found (404)")
	case 401:
		return nil, fmt.Errorf("n8n_api_trigger: invalid API key (401)")
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("n8n_api_trigger: unexpected status %d: %s", resp.StatusCode, string(respBytes))
	}

	log.Printf("[n8n] n8n_api_trigger: %s → %d (%dms)", path, resp.StatusCode, latencyMs)

	// Extract execution_id from response.
	var respData map[string]interface{}
	executionID := ""
	if err := json.Unmarshal(respBytes, &respData); err == nil {
		if eid, ok := respData["execution_id"].(string); ok {
			executionID = eid
		} else if eid, ok := respData["executionId"].(string); ok {
			executionID = eid
		} else if data, ok := respData["data"].(map[string]interface{}); ok {
			if eid, ok := data["execution_id"].(string); ok {
				executionID = eid
			} else if eid, ok := data["id"].(string); ok {
				executionID = eid
			}
		}
	}

	result := APITriggerResponse{
		ExecutionID: executionID,
		WorkflowID:  workflowID,
		QueuedAt:    time.Now().UTC().Format(time.RFC3339),
	}

	outBytes, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("n8n_api_trigger: marshal response: %w", err)
	}

	return string(outBytes), nil
}
