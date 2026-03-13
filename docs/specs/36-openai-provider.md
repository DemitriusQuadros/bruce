# Spec 36: OpenAI Provider [BACKEND + FRONTEND]

## Overview

Add OpenAI as the third LLM provider in Bruce's multi-provider architecture. Follows the same patterns established by Claude (`internal/ai/claude.go`) and Gemini (`internal/ai/gemini.go`): raw `net/http` calls, provider-specific request/response mapping, shared `sanitizeHistory()`, and sentinel error mapping. OpenAI uses the Chat Completions API with system prompt as the first message in the `messages` array.

## Phase

**Phase 2** (continuation of multi-provider work from Spec 31)

## Prerequisites

- Multi-provider architecture exists (Spec 07/31 implemented)
- `ProviderRegistry` with `ListProviders()` exists (`internal/ai/registry.go`)
- `BuildProviders()` factory exists (`internal/ai/factory.go`)
- `OpenAIConfig` struct already exists in `internal/config/config.go`
- `Config.OpenAI` field already wired in top-level struct
- Warning message in `config.go` already references `openai.api_key`
- Settings UI exists (`web/public/index.html`, `web/public/js/modules/settings.js`)

## Deliverables

**Files to Create:**
- `internal/ai/openai.go` -- OpenAI provider implementation
- `internal/ai/openai_test.go` -- Unit tests for OpenAI provider

**Files to Modify:**
- `internal/ai/provider.go` -- Add `ProviderOpenAI` constant
- `internal/ai/factory.go` -- Register OpenAI provider when API key is present
- `internal/ai/registry.go` -- Include OpenAI in `ListProviders()`
- `config.example.yml` -- Add `openai` config section
- `web/public/index.html` -- Add OpenAI settings fields (API key, model, max_tokens)
- `web/public/js/modules/settings.js` -- Add OpenAI number field validation

**Files Already Done (no changes needed):**
- `internal/config/config.go` -- `OpenAIConfig` struct and `Config.OpenAI` field already exist

## API / Component Contract

### OpenAI Chat Completions API

**Endpoint:** `POST https://api.openai.com/v1/chat/completions`

**Request Headers:**
```
Content-Type: application/json
Authorization: Bearer {api_key}
```

**Request Body:**
```json
{
  "model": "gpt-4o",
  "max_tokens": 1024,
  "messages": [
    {"role": "system", "content": "You are Bruce..."},
    {"role": "user", "content": "Hello"},
    {"role": "assistant", "content": "Hi there!"},
    {"role": "user", "content": "What can you do?"}
  ]
}
```

**Response Body:**
```json
{
  "id": "chatcmpl-...",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "I can help you with..."
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 50,
    "completion_tokens": 100,
    "total_tokens": 150
  }
}
```

**Error Mapping:**
| HTTP Status | Sentinel Error | Retry? |
|-------------|---------------|--------|
| 429 | `ErrRateLimited` | Yes (backoff) |
| 500, 502, 503 | `ErrProviderDown` | Yes (backoff) |
| 400 | `ErrBadRequest` | No |
| 401 | Generic error (bad API key) | No |

### Provider Implementation (`internal/ai/openai.go`)

```go
type openaiProvider struct {
    apiKey     string
    model      string
    maxTokens  int
    httpClient *http.Client
}

func NewOpenAIProvider(cfg config.OpenAIConfig) LLMService { ... }
func (o *openaiProvider) GenerateResponse(ctx context.Context, systemPrompt string, history []domain.Message) (string, error) { ... }
```

**Key differences from Claude/Gemini:**
- System prompt is a message with `role: "system"` prepended to the messages array (not a separate field like Claude's `system` or Gemini's `system_instruction`)
- Role names match domain model directly: `"user"`, `"assistant"`, `"system"` -- no mapping needed
- Auth via `Authorization: Bearer` header (not `x-api-key` like Claude, not query param like Gemini)
- Response extraction: `choices[0].message.content`

### Provider Constant (`internal/ai/provider.go`)

```go
const (
    ProviderClaude ProviderName = "claude"
    ProviderGemini ProviderName = "gemini"
    ProviderOpenAI ProviderName = "openai"  // NEW
)
```

### Factory Registration (`internal/ai/factory.go`)

```go
if cfg.OpenAI.APIKey != "" {
    providers[ProviderOpenAI] = NewOpenAIProvider(cfg.OpenAI)
    logging.Infof("ai: registered provider %s (model: %s)", ProviderOpenAI, cfg.OpenAI.Model)
}
```

Update warning message:
```go
if len(providers) == 0 {
    logging.Warn("no LLM providers configured. Set claude.api_key, gemini.api_key, or openai.api_key in config.")
}
```

### Registry Update (`internal/ai/registry.go`)

`ListProviders()` must include OpenAI:
```go
func (r *ProviderRegistry) ListProviders() []ProviderInfo {
    r.mu.RLock()
    defer r.mu.RUnlock()

    _, claudeAvailable := r.providers[ProviderClaude]
    _, geminiAvailable := r.providers[ProviderGemini]
    _, openaiAvailable := r.providers[ProviderOpenAI]

    result := []ProviderInfo{
        {Name: string(ProviderClaude), Available: claudeAvailable, Model: r.cfg.Claude.Model},
        {Name: string(ProviderGemini), Available: geminiAvailable, Model: r.cfg.Gemini.Model},
        {Name: string(ProviderOpenAI), Available: openaiAvailable, Model: r.cfg.OpenAI.Model},
    }
    return result
}
```

### Config (`config.example.yml`)

```yaml
openai:
  api_key: ""
  model: "gpt-4o"
  max_tokens: 1024
```

### Settings UI (`web/public/index.html`)

Add OpenAI section after existing Claude fields, before the system prompt textarea. Use provider-grouped fieldsets:

```html
<!-- OpenAI Settings -->
<h3>OpenAI</h3>
<div class="form-group">
    <label>API Key</label>
    <input type="password" name="openai.api_key" />
</div>
<div class="form-group">
    <label>Model</label>
    <select name="openai.model">
        <option>gpt-4o</option>
        <option>gpt-4o-mini</option>
        <option>gpt-4-turbo</option>
        <option>o1</option>
        <option>o3-mini</option>
    </select>
</div>
<div class="form-group">
    <label>Max Tokens</label>
    <input type="number" name="openai.max_tokens" />
</div>
```

### Settings JS (`web/public/js/modules/settings.js`)

Add `openai.max_tokens` to the number validation block alongside `claude.max_tokens` and `claude.context_window`:

```javascript
if (key === 'claude.max_tokens' || key === 'claude.context_window' || key === 'openai.max_tokens') {
    const num = parseInt(value, 10);
    if (isNaN(num) || num <= 0) {
        continue;
    }
    configValue = String(num);
}
```

## Detailed Implementation

### `internal/ai/openai.go`

```go
package ai

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"

    "bruce/internal/config"
    "bruce/internal/domain"
)

type openaiProvider struct {
    apiKey     string
    model      string
    maxTokens  int
    httpClient *http.Client
}

func NewOpenAIProvider(cfg config.OpenAIConfig) LLMService {
    return &openaiProvider{
        apiKey:     cfg.APIKey,
        model:      cfg.Model,
        maxTokens:  cfg.MaxTokens,
        httpClient: &http.Client{Timeout: 60 * time.Second},
    }
}

// OpenAI API request types
type openaiRequest struct {
    Model     string          `json:"model"`
    MaxTokens int             `json:"max_tokens"`
    Messages  []openaiMessage `json:"messages"`
}

type openaiMessage struct {
    Role    string `json:"role"` // "system" | "user" | "assistant"
    Content string `json:"content"`
}

// OpenAI API response types
type openaiResponse struct {
    Choices []openaiChoice `json:"choices"`
}

type openaiChoice struct {
    Message openaiMessage `json:"message"`
}

func (o *openaiProvider) GenerateResponse(ctx context.Context, systemPrompt string, history []domain.Message) (string, error) {
    cleaned := sanitizeHistory(history)

    // Build messages array: system prompt first, then history
    msgs := make([]openaiMessage, 0, len(cleaned)+1)

    if systemPrompt != "" {
        msgs = append(msgs, openaiMessage{Role: "system", Content: systemPrompt})
    }

    for _, m := range cleaned {
        msgs = append(msgs, openaiMessage{Role: m.Role, Content: m.Content})
    }

    reqBody := openaiRequest{
        Model:     o.model,
        MaxTokens: o.maxTokens,
        Messages:  msgs,
    }

    bodyBytes, err := json.Marshal(reqBody)
    if err != nil {
        return "", fmt.Errorf("marshal request: %w", err)
    }

    httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
        "https://api.openai.com/v1/chat/completions", bytes.NewReader(bodyBytes))
    if err != nil {
        return "", fmt.Errorf("build request: %w", err)
    }
    httpReq.Header.Set("Content-Type", "application/json")
    httpReq.Header.Set("Authorization", "Bearer "+o.apiKey)

    resp, err := o.httpClient.Do(httpReq)
    if err != nil {
        return "", fmt.Errorf("http do: %w", err)
    }
    defer resp.Body.Close()

    respBytes, err := io.ReadAll(resp.Body)
    if err != nil {
        return "", fmt.Errorf("read response: %w", err)
    }

    switch resp.StatusCode {
    case http.StatusOK:
        // handled below
    case 429:
        return "", ErrRateLimited
    case 500, 502, 503:
        return "", ErrProviderDown
    case 400:
        return "", fmt.Errorf("%w: %s", ErrBadRequest, string(respBytes))
    default:
        return "", fmt.Errorf("openai API %d: %s", resp.StatusCode, string(respBytes))
    }

    var apiResp openaiResponse
    if err := json.Unmarshal(respBytes, &apiResp); err != nil {
        return "", fmt.Errorf("decode response: %w", err)
    }
    if len(apiResp.Choices) == 0 {
        return "", fmt.Errorf("openai returned empty choices")
    }
    return apiResp.Choices[0].Message.Content, nil
}
```

### `internal/ai/openai_test.go`

Tests should follow the same pattern as existing provider tests. Use `httptest.NewServer` to mock the OpenAI API.

**Test cases:**

1. **Happy path** -- valid request returns assistant message content
2. **System prompt prepended** -- verify system message is first in the messages array
3. **Empty system prompt** -- no system message in array when prompt is empty
4. **Rate limited (429)** -- returns `ErrRateLimited`
5. **Server error (500)** -- returns `ErrProviderDown`
6. **Bad request (400)** -- returns `ErrBadRequest` with body
7. **Empty choices** -- returns error
8. **History sanitization** -- system messages in history are stripped, consecutive same-role messages merged
9. **Authorization header** -- verify `Bearer` token format

```go
func TestOpenAIProvider_GenerateResponse(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Verify auth header
        assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))

        // Decode and verify request
        var req openaiRequest
        json.NewDecoder(r.Body).Decode(&req)
        assert.Equal(t, "gpt-4o", req.Model)
        assert.Equal(t, "system", req.Messages[0].Role)

        // Return mock response
        json.NewEncoder(w).Encode(openaiResponse{
            Choices: []openaiChoice{
                {Message: openaiMessage{Role: "assistant", Content: "Hello!"}},
            },
        })
    }))
    defer server.Close()

    // Create provider pointing at test server
    // (requires making baseURL configurable or using the test server URL)
}
```

**Note:** To make the provider testable, either:
- Add an unexported `baseURL` field (like `geminiProvider` does) and set it in tests, or
- Accept a base URL in the constructor for test override

Recommended approach: add `baseURL` field defaulting to `"https://api.openai.com"`, consistent with `geminiProvider.baseURL`.

## Acceptance Criteria

- [ ] `ProviderOpenAI` constant defined in `provider.go`
- [ ] `openaiProvider` struct implements `LLMService` interface
- [ ] System prompt sent as first message with `role: "system"`
- [ ] Domain roles (`user`, `assistant`) pass through without mapping
- [ ] `sanitizeHistory()` called to strip system messages from history and merge consecutive same-role
- [ ] Auth uses `Authorization: Bearer {key}` header
- [ ] HTTP 429 maps to `ErrRateLimited`
- [ ] HTTP 500/502/503 maps to `ErrProviderDown`
- [ ] HTTP 400 maps to `ErrBadRequest` with response body
- [ ] Empty choices array returns descriptive error
- [ ] `BuildProviders()` registers OpenAI when `openai.api_key` is non-empty
- [ ] `ListProviders()` includes OpenAI with availability status and model
- [ ] `config.example.yml` has `openai` section with `api_key`, `model`, `max_tokens`
- [ ] Settings UI has OpenAI API key (password input), model (select), max_tokens (number input)
- [ ] Settings JS validates `openai.max_tokens` as positive integer
- [ ] Unit tests cover happy path, error mapping, system prompt handling, and history sanitization
- [ ] `go build ./...` passes
- [ ] `go test ./internal/ai/...` passes
- [ ] `go vet ./...` clean

## Implementation Steps

### Step 1: Add `ProviderOpenAI` Constant (5 min)
- Edit `internal/ai/provider.go`
- Add `ProviderOpenAI ProviderName = "openai"` to const block
- Deliverable: constant exists, `go build` passes

### Step 2: Create `internal/ai/openai.go` (30 min)
- Create file with `openaiProvider` struct, request/response types, `NewOpenAIProvider()`, `GenerateResponse()`
- Include `baseURL` field (default `https://api.openai.com`) for testability
- Follow exact patterns from `claude.go` for error handling
- Deliverable: file compiles, implements `LLMService`

### Step 3: Register in Factory (5 min)
- Edit `internal/ai/factory.go`
- Add OpenAI registration block after Gemini block
- Update warning message to include `openai.api_key`
- Deliverable: provider auto-registers when key is configured

### Step 4: Update `ListProviders()` (5 min)
- Edit `internal/ai/registry.go`
- Add OpenAI to the hardcoded provider list in `ListProviders()`
- Deliverable: `/api/v1/providers` returns OpenAI entry

### Step 5: Update `config.example.yml` (2 min)
- Add `openai` section after `gemini` section
- Deliverable: example config documents all three providers

### Step 6: Update Settings UI (15 min)
- Edit `web/public/index.html` -- add OpenAI fieldset with API key, model select, max_tokens
- Edit `web/public/js/modules/settings.js` -- add `openai.max_tokens` to number validation
- Deliverable: Settings tab shows OpenAI fields, values persist on save/reload

### Step 7: Write Unit Tests (30 min)
- Create `internal/ai/openai_test.go`
- Test happy path, error codes, system prompt, empty prompt, empty choices, auth header
- Use `httptest.NewServer` to mock OpenAI API
- Deliverable: `go test ./internal/ai/...` passes with new tests

### Step 8: Verify End-to-End (10 min)
- Set `openai.api_key` in `config.yml`
- Start server, verify OpenAI appears in provider list
- Select OpenAI as provider, send a chat message
- Verify response renders in web chat
- Deliverable: manual smoke test passes

## Out of Scope

- OpenAI streaming responses (`stream: true`) -- future enhancement
- OpenAI function calling / tool use integration -- requires Spec 11 tool loop adaptation per provider
- OpenAI-specific features: JSON mode, vision, file uploads
- Token usage tracking or cost estimation
- Model-specific parameter tuning (temperature, top_p) -- all providers use defaults for now
- o1/o3 reasoning model special handling (these models ignore `max_tokens`, use `max_completion_tokens`) -- can be addressed later if needed
