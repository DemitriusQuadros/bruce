package n8n

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"bruce/internal/ai"
)

// nameReplaceRe matches characters not allowed in sanitized tool names.
var nameReplaceRe = regexp.MustCompile(`[^a-zA-Z0-9_]`)

// collapseUnderscoreRe collapses consecutive underscores.
var collapseUnderscoreRe = regexp.MustCompile(`_{2,}`)

// sanitizeName converts an MCP tool name to a safe tool registry name.
// It prepends the configured prefix, replaces invalid chars with underscores,
// collapses consecutive underscores, and truncates to 64 characters.
func sanitizeName(original, prefix string) string {
	name := prefix + original
	name = nameReplaceRe.ReplaceAllString(name, "_")
	name = collapseUnderscoreRe.ReplaceAllString(name, "_")
	if len(name) > 64 {
		name = name[:64]
	}
	return name
}

// MCPTool implements tools.Tool for a single tool discovered from an MCP endpoint.
type MCPTool struct {
	sanitizedName string
	originalName  string
	description   string
	schema        map[string]interface{}
	provider      *MCPProvider
}

// Name returns the sanitized tool name.
func (t *MCPTool) Name() string { return t.sanitizedName }

// Definition returns the AI tool definition for this MCP-discovered tool.
func (t *MCPTool) Definition() ai.ToolDefinition {
	schema := t.schema
	if schema == nil {
		schema = map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}
	}
	return ai.ToolDefinition{
		Name:        t.sanitizedName,
		Description: t.description,
		InputSchema: schema,
	}
}

// jsonRPCRequest is a JSON-RPC 2.0 request envelope.
type jsonRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	ID      int         `json:"id"`
	Params  interface{} `json:"params"`
}

// Execute calls the MCP tool via JSON-RPC over SSE.
func (t *MCPTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	rpcReq := jsonRPCRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		ID:      1,
		Params: map[string]interface{}{
			"name":      t.originalName,
			"arguments": input,
		},
	}

	body, err := json.Marshal(rpcReq)
	if err != nil {
		return nil, fmt.Errorf("mcp_tool: marshal rpc request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", t.provider.cfg.SSEURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("mcp_tool: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	if t.provider.cfg.BearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+t.provider.cfg.BearerToken)
	}

	httpClient := &http.Client{Timeout: 60 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("mcp_tool: request: %w", err)
	}
	defer resp.Body.Close()

	result, err := parseSSEResponse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("mcp_tool: parse SSE response: %w", err)
	}

	// Extract content from JSON-RPC result.
	if res, ok := result["result"]; ok {
		if content, err := json.Marshal(res); err == nil {
			return string(content), nil
		}
	}

	out, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("mcp_tool: marshal result: %w", err)
	}

	return string(out), nil
}

// parseSSEResponse scans an SSE stream and returns the first valid JSON-RPC result.
func parseSSEResponse(r io.Reader) (map[string]interface{}, error) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		dataStr := strings.TrimPrefix(line, "data: ")
		var result map[string]interface{}
		if err := json.Unmarshal([]byte(dataStr), &result); err == nil {
			// Check for JSON-RPC error.
			if rpcErr, ok := result["error"]; ok {
				errBytes, _ := json.Marshal(rpcErr)
				return nil, fmt.Errorf("JSON-RPC error: %s", string(errBytes))
			}
			return result, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan SSE stream: %w", err)
	}
	return nil, fmt.Errorf("no valid JSON-RPC data in SSE stream")
}

// discoverTools calls tools/list on the MCP SSE endpoint and returns the raw tool list.
func discoverTools(ctx context.Context, sseURL, bearerToken string) ([]map[string]interface{}, error) {
	rpcReq := jsonRPCRequest{
		JSONRPC: "2.0",
		Method:  "tools/list",
		ID:      1,
		Params:  map[string]interface{}{},
	}

	body, err := json.Marshal(rpcReq)
	if err != nil {
		return nil, fmt.Errorf("marshal tools/list request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", sseURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build tools/list request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	if bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+bearerToken)
	}

	httpClient := &http.Client{Timeout: 30 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tools/list request: %w", err)
	}
	defer resp.Body.Close()

	result, err := parseSSEResponse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse tools/list response: %w", err)
	}

	// Navigate result.result.tools or result.tools.
	var toolList []interface{}
	if res, ok := result["result"].(map[string]interface{}); ok {
		if tools, ok := res["tools"].([]interface{}); ok {
			toolList = tools
		}
	} else if tools, ok := result["tools"].([]interface{}); ok {
		toolList = tools
	}

	if toolList == nil {
		log.Printf("[n8n] discoverTools: no tools found in response")
		return nil, nil
	}

	out := make([]map[string]interface{}, 0, len(toolList))
	for _, t := range toolList {
		if m, ok := t.(map[string]interface{}); ok {
			out = append(out, m)
		}
	}
	return out, nil
}
