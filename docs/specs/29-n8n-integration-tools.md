# Spec 31: n8n Integration Tools [BACKEND]

## Overview

Implement four integration patterns that allow Bruce (the AI assistant) to call external HTTP
endpoints and n8n workflows. The patterns form a layered stack — from the most generic
(`http_request`) to fully dynamic, self-describing tools discovered at runtime via MCP.

1. **`http_request`** — Generic HTTP client tool; the LLM constructs any request.
2. **`n8n_webhook_call`** — Higher-level wrapper for n8n Webhook Trigger nodes; the LLM
   only knows the webhook path, not the full URL or auth secrets.
3. **`n8n_api_trigger`** — Uses the n8n management REST API to enqueue an existing workflow
   by ID; fire-and-forget.
4. **MCP Client provider** — Connects to an n8n "MCP Server Trigger" node over SSE at
   startup, discovers available tools, and registers each as a native `tools.Tool`.

## Phase

**Phase 2** (Weeks 7–12)

## Prerequisites

- Tool registry exists (Spec 12)
- `internal/tools/executor.go` with `logExecution` / `executeWithTimeout` in place
- `internal/config/config.go` — `ToolsConfig` struct established
- No confirmation system required (tools are read-heavy / fire-and-forget)

## Deliverables

**Files to Create:**

- `internal/tools/httpclient/http_request.go` — `http_request` tool
- `internal/tools/n8n/client.go` — shared HTTP client (base URL, auth, timeout helpers)
- `internal/tools/n8n/webhook.go` — `n8n_webhook_call` tool
- `internal/tools/n8n/api_trigger.go` — `n8n_api_trigger` tool
- `internal/tools/n8n/mcp_provider.go` — SSE connection, tool discovery, reconnect loop
- `internal/tools/n8n/mcp_tool.go` — `MCPTool` adapter implementing `tools.Tool`

**Files to Modify:**

- `internal/config/config.go` — add `HTTPClientConfig`, `N8nConfig`, embedded `MCPConfig`
- `config.example.yml` — document all new config keys
- `cmd/bruce/main.go` — register new tools + MCP provider at startup

## Acceptance Criteria

### `http_request`
- [ ] Accepts `method`, `url`, `headers` (optional), `body` (optional), `timeout_seconds` (optional, default 30, max 120)
- [ ] Returns `status_code`, `body` (string), `headers` (map), `latency_ms`, `truncated` (bool)
- [ ] Blocks RFC 1918 and loopback addresses by default (SSRF protection)
- [ ] DNS rebinding mitigation: resolve hostname, check resolved IP against CIDR blocklist before connecting
- [ ] Config flag `allow_private_networks: true` bypasses the CIDR blocklist
- [ ] Truncates response body at `max_response_bytes` (default 512 KB); sets `truncated: true`
- [ ] Scrubs `Authorization` and `X-Api-Key` headers from `tool_executions` log entry
- [ ] Honors `follow_redirects: false` by default (does not follow 3xx)
- [ ] Disabled when `tools.http_client.enabled: false`

### `n8n_webhook_call`
- [ ] Accepts `webhook_path`, `payload` (optional JSON object), `timeout_seconds` (optional)
- [ ] Assembles full URL from `tools.n8n.base_url` + `webhook_path`
- [ ] Applies configured auth (`none` / `basic` / `header`) centrally — LLM never sees secrets
- [ ] Returns `status_code`, `response` (parsed JSON or null), `raw_response` (string if not JSON), `latency_ms`
- [ ] Surfaces n8n-specific errors: webhook not registered (workflow disabled/missing → 404), auth failure (401/403)
- [ ] Disabled when `tools.n8n.enabled: false`

### `n8n_api_trigger`
- [ ] Accepts `workflow_id`, `data` (optional JSON object)
- [ ] Calls `POST {base_url}/api/v1/workflows/{workflow_id}/run` with `X-N8N-API-KEY` header
- [ ] Returns `execution_id`, `workflow_id`, `queued_at` (ISO 8601 string)
- [ ] Fire-and-forget — does not poll or wait for execution completion
- [ ] Returns descriptive error if workflow not found (404) or API key invalid (401)
- [ ] Disabled when `tools.n8n.enabled: false`

### MCP Client provider
- [ ] Connects to `tools.n8n.mcp.sse_url` at startup; authenticates with `Bearer {bearer_token}`
- [ ] Sends `tools/list` JSON-RPC over SSE; registers each returned tool as `MCPTool` with `n8n_mcp_` prefix
- [ ] Tool names are sanitized: non-alphanumeric characters replaced with `_`, max 64 chars
- [ ] Rejects tools whose sanitized name collides with an already-registered tool; logs warning
- [ ] Startup failure is non-fatal — Bruce starts without MCP tools, logs warning
- [ ] Background reconnect loop with exponential backoff (max `max_reconnect_attempts`, default 5)
- [ ] `MCPTool.Execute` proxies to `tools/call` JSON-RPC over the persistent SSE connection
- [ ] Disabled when `tools.n8n.mcp.enabled: false`

### General
- [ ] All tools compile with `go build ./...`
- [ ] All new config keys documented in `config.example.yml`
- [ ] `cmd/bruce/main.go` wires all tools and the MCP provider
- [ ] `tool_executions` log entries do not contain raw secret values

## API / Component Contract

### `http_request` Tool Schema

```json
{
  "name": "http_request",
  "description": "Make an HTTP request to any URL and return the response. Blocked for private/loopback IPs by default.",
  "input_schema": {
    "type": "object",
    "properties": {
      "method": {
        "type": "string",
        "description": "HTTP method: GET, POST, PUT, PATCH, DELETE",
        "enum": ["GET", "POST", "PUT", "PATCH", "DELETE"]
      },
      "url": {
        "type": "string",
        "description": "Full URL including scheme (https:// required for external hosts)"
      },
      "headers": {
        "type": "object",
        "description": "Optional map of request headers (string → string)",
        "additionalProperties": { "type": "string" }
      },
      "body": {
        "type": "string",
        "description": "Optional request body (raw string; set Content-Type header accordingly)"
      },
      "timeout_seconds": {
        "type": "integer",
        "description": "Request timeout in seconds (default 30, max 120)"
      }
    },
    "required": ["method", "url"]
  }
}
```

**Output** (JSON object):
```json
{
  "status_code": 200,
  "body": "response body string (may be truncated)",
  "headers": { "Content-Type": "application/json" },
  "latency_ms": 142,
  "truncated": false
}
```

---

### `n8n_webhook_call` Tool Schema

```json
{
  "name": "n8n_webhook_call",
  "description": "Trigger an n8n Webhook Trigger node by path and optionally pass a JSON payload.",
  "input_schema": {
    "type": "object",
    "properties": {
      "webhook_path": {
        "type": "string",
        "description": "n8n webhook path (e.g. 'my-workflow' for /webhook/my-workflow)"
      },
      "payload": {
        "type": "object",
        "description": "Optional JSON payload sent as the request body"
      },
      "timeout_seconds": {
        "type": "integer",
        "description": "Request timeout in seconds (default from config, max 120)"
      }
    },
    "required": ["webhook_path"]
  }
}
```

**Output** (JSON object):
```json
{
  "status_code": 200,
  "response": { "result": "ok" },
  "raw_response": null,
  "latency_ms": 85
}
```
When the response body is not valid JSON, `response` is `null` and `raw_response` holds the string.

---

### `n8n_api_trigger` Tool Schema

```json
{
  "name": "n8n_api_trigger",
  "description": "Start an n8n workflow execution by workflow ID via the n8n REST API (fire-and-forget).",
  "input_schema": {
    "type": "object",
    "properties": {
      "workflow_id": {
        "type": "string",
        "description": "n8n workflow ID (numeric string, e.g. '42')"
      },
      "data": {
        "type": "object",
        "description": "Optional JSON data passed as the workflow trigger payload"
      }
    },
    "required": ["workflow_id"]
  }
}
```

**Output** (JSON object):
```json
{
  "execution_id": "1234",
  "workflow_id": "42",
  "queued_at": "2026-03-16T12:00:00Z"
}
```

---

### `n8n/client.go` — Shared HTTP Client

```go
// N8nClient wraps an *http.Client pre-configured with base URL, auth, and timeout.
type N8nClient struct {
    baseURL    string
    httpClient *http.Client
    authMethod string // "none" | "basic" | "header"
    // auth fields populated from config
}

func NewN8nClient(cfg config.N8nConfig) *N8nClient

// Do executes a request, injecting base URL and auth headers.
func (c *N8nClient) Do(ctx context.Context, method, path string, body io.Reader) (*http.Response, error)
```

---

### `n8n/mcp_provider.go` — MCP Tool Discovery

```go
// MCPProvider connects to an n8n MCP Server Trigger node over SSE, discovers tools,
// and returns a slice of tools.Tool ready for registration.
type MCPProvider struct {
    cfg      config.MCPConfig
    registry ToolRegistry // interface — RegisterTool(tools.Tool) error
}

func NewMCPProvider(cfg config.MCPConfig, registry ToolRegistry) *MCPProvider

// Start performs tool discovery and begins the background reconnect loop.
// Non-fatal: logs a warning on failure instead of returning an error.
func (p *MCPProvider) Start(ctx context.Context)
```

---

### `n8n/mcp_tool.go` — MCPTool Adapter

```go
// MCPTool wraps a tool definition received from n8n's MCP endpoint and
// proxies Execute calls back via JSON-RPC tools/call.
type MCPTool struct {
    name        string // sanitized, with prefix
    description string
    schema      json.RawMessage
    provider    *MCPProvider
}

func (t *MCPTool) Name() string
func (t *MCPTool) Definition() anthropic.ToolDefinition
func (t *MCPTool) Execute(ctx context.Context, input json.RawMessage) (string, error)
```

---

### Config Additions (`internal/config/config.go`)

```go
type HTTPClientConfig struct {
    Enabled             bool   `mapstructure:"enabled"`
    MaxTimeoutSeconds   int    `mapstructure:"max_timeout_seconds"`
    MaxResponseBytes    int    `mapstructure:"max_response_bytes"`
    FollowRedirects     bool   `mapstructure:"follow_redirects"`
    AllowPrivateNetworks bool  `mapstructure:"allow_private_networks"`
}

type WebhookAuthConfig struct {
    Method      string `mapstructure:"method"` // none | basic | header
    Username    string `mapstructure:"username"`
    Password    string `mapstructure:"password"`
    HeaderName  string `mapstructure:"header_name"`
    HeaderValue string `mapstructure:"header_value"`
}

type MCPConfig struct {
    Enabled              bool   `mapstructure:"enabled"`
    SSEURL               string `mapstructure:"sse_url"`
    BearerToken          string `mapstructure:"bearer_token"`
    ToolNamePrefix       string `mapstructure:"tool_name_prefix"`
    MaxReconnectAttempts int    `mapstructure:"max_reconnect_attempts"`
}

type N8nConfig struct {
    Enabled                bool              `mapstructure:"enabled"`
    BaseURL                string            `mapstructure:"base_url"`
    WebhookTimeoutSeconds  int               `mapstructure:"webhook_timeout_seconds"`
    WebhookAuth            WebhookAuthConfig `mapstructure:"webhook_auth"`
    APIKey                 string            `mapstructure:"api_key"`
    MCP                    MCPConfig         `mapstructure:"mcp"`
}

// Add to ToolsConfig:
type ToolsConfig struct {
    // ... existing fields ...
    HTTPClient HTTPClientConfig `mapstructure:"http_client"`
    N8n        N8nConfig        `mapstructure:"n8n"`
}
```

## Configuration Schema

Full `config.example.yml` additions:

```yaml
tools:
  http_client:
    enabled: true
    max_timeout_seconds: 120
    max_response_bytes: 524288   # 512 KB
    follow_redirects: false
    allow_private_networks: false

  n8n:
    enabled: false
    base_url: "https://your-n8n.example.com"
    webhook_timeout_seconds: 30
    webhook_auth:
      method: "none"          # none | basic | header
      username: ""
      password: ""
      header_name: ""
      header_value: ""
    api_key: ""               # X-N8N-API-KEY for /api/v1/workflows/:id/run
    mcp:
      enabled: false
      sse_url: "https://your-n8n.example.com/mcp"
      bearer_token: ""
      tool_name_prefix: "n8n_mcp_"
      max_reconnect_attempts: 5
```

## Architectural Decisions

### SSRF protection on `http_request`
Block RFC 1918 (10.x, 172.16–31.x, 192.168.x), loopback (127.x, ::1), and link-local
(169.254.x) addresses by default. DNS rebinding mitigation: after DNS resolution, check
each resolved IP against the CIDR blocklist before opening a TCP connection. Set
`allow_private_networks: true` to disable this check (e.g., for internal n8n instances).

### Timeout layering
The tool registry's `executeWithTimeout` has a 5-second default backstop. HTTP tools
derive a child context with the user-supplied or config-specified timeout, which is
always longer. The registry backstop is a last-resort safety net only.

### No tool-calls-tool
`http_request` and `n8n_webhook_call` both use `net/http` directly. They share
`n8n/client.go` config helpers but do not call each other. This prevents double-counting
in `tool_executions` logs.

### Secret scrubbing in `logExecution`
Before persisting a `tool_executions` row, scan the raw input JSON for keys matching
`authorization`, `x-api-key`, `x-n8n-api-key` (case-insensitive) and replace the value
with `"[REDACTED]"`.

### MCP tool name sanitization
Replace any character outside `[a-zA-Z0-9_]` with `_`. Collapse multiple consecutive
underscores to one. Truncate to 64 characters. Prepend the configured prefix (default
`n8n_mcp_`). If the resulting name collides with an existing registered tool, skip
registration and log a warning — do not panic or fail startup.

### MCP reconnect backoff
Initial delay: 1 second. Each subsequent attempt doubles (1s, 2s, 4s, 8s, 16s). After
`max_reconnect_attempts` failures, the background goroutine exits and logs an error. A
future restart of Bruce will reattempt discovery.

### Intentional layering overlap
`http_request` and `n8n_webhook_call` can both call an n8n webhook. The separation is
intentional: `http_request` is for ad-hoc, open-ended HTTP calls; `n8n_webhook_call` is
for structured, pre-configured n8n interactions where the LLM should not know the base
URL or credentials.

## Out of Scope

- Polling n8n execution status after `n8n_api_trigger` fires (fire-and-forget only)
- Streaming / chunked response support for `http_request`
- WebSocket connections
- mTLS / client certificates
- n8n OAuth2 webhook auth (only `none`, `basic`, `header`)
- MCP tool schema validation beyond name sanitization
- UI settings panel for n8n credentials (handled by Spec 23)
