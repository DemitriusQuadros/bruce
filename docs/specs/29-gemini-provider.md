# Spec 31: Gemini Provider [BACKEND]

## Overview

Implement support for Google Gemini as an alternative LLM provider alongside Claude. Refactor `internal/ai/` to define a multi-provider abstraction. Create `internal/ai/provider.go` interface, implement both Claude and Gemini providers, and add provider selection to config and session model. Worker is provider-agnostic; provider is selected at request time. Phase 2 adds Gemini support; both Claude and Gemini work equally for all tools.

## Phase

**Phase 2** (Weeks 7–14)

## Prerequisites

- Claude integration exists (`internal/ai/claude.go`)
- Session model exists (`internal/domain/session.go`)
- Config system exists (`internal/config/config.go`)
- Tool execution engine works with Claude (Spec 11)

## Deliverables

**Files to Create:**
- `internal/ai/provider.go` — LLMProvider interface
- `internal/ai/providers/gemini.go` — Gemini implementation
- `internal/ai/providers/claude.go` — Claude refactored into provider package

**Files to Modify:**
- `internal/ai/claude.go` — refactor into provider implementation
- `internal/worker/processor.go` — accept provider from session, route through provider registry
- `internal/config/config.go` — add `llm.provider` (default: "claude"), `gemini.api_key`
- `internal/domain/session.go` — add `provider_override` field (nullable) for per-session provider choice
- `cmd/bruce/main.go` — register both providers at startup
- `config.example.yml` — document Gemini API key setup

## Acceptance Criteria

- [ ] `LLMProvider` interface has methods: `GenerateResponse(ctx, messages, tools)`, `Name()`, `Provider()`
- [ ] Claude provider implements full interface (all tool_use, tool_result logic from Spec 11)
- [ ] Gemini provider implements full interface with equivalent tool support (Google's function calling API)
- [ ] Config has `llm.provider: "claude"` or `"gemini"` (global default)
- [ ] Session model has nullable `provider_override` field; if set, overrides global default
- [ ] Worker reads `session.provider_override` (or global config) at request time, selects provider
- [ ] Both providers return compatible response format for Spec 11 tool loop
- [ ] Gemini tool calls parse to same `ToolCall` struct as Claude
- [ ] Settings UI shows provider selection dropdown (Spec 32); updates session `provider_override`
- [ ] Error handling is consistent across providers (Spec 24 applies to both)
- [ ] Latency: both providers <5s p95 for typical requests (mocked)

## API / Component Contract

**`internal/ai/provider.go`**:
```go
type LLMProvider interface {
	Name() string // "claude" or "gemini"
	GenerateResponse(ctx context.Context, messages []interface{}, tools []ToolSpec) (*GenerateResponse, error)
}

type ToolSpec struct {
	Name        string
	Description string
	InputSchema map[string]interface{}
}

type GenerateResponse struct {
	Text      string
	ToolCalls []ToolCall
	Complete  bool
}
```

**Config**:
```yaml
llm:
  provider: "claude" # or "gemini"

claude:
  api_key: "sk-ant-..."
  model: "claude-opus-4-6"
  max_tokens: 2048

gemini:
  api_key: "AIzaSy..."
  model: "gemini-1.5-pro"
  max_tokens: 2048
```

**Provider Registration** (`cmd/bruce/main.go`):
```go
claudeProvider := providers.NewClaude(config.Claude.APIKey, config.Claude.Model)
geminiProvider := providers.NewGemini(config.Gemini.APIKey, config.Gemini.Model)

providerRegistry := ai.NewProviderRegistry()
providerRegistry.Register(claudeProvider)
providerRegistry.Register(geminiProvider)

worker.SetProviderRegistry(providerRegistry)
worker.SetDefaultProvider(config.LLM.Provider)
```

## Out of Scope

- Other providers (OpenAI, Anthropic Haiku, etc. — future)
- Provider cost tracking (mentioned in Phase 3 ideas)
- Provider A/B testing framework
- Provider-specific model selection beyond config
