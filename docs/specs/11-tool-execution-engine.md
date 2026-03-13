# Spec 11: Tool Execution Engine [BACKEND]

## Overview

Implement a provider-agnostic agentic loop in `internal/ai/tool_loop.go` that works with any LLM provider (Claude, Gemini, etc.). The loop manages multi-turn tool execution: calling the LLM provider's `GenerateWithTools` method, parsing tool calls from the response, executing registered tools, and feeding results back to the LLM until a final response is produced. Each LLM provider implements tool calling internally (Claude via Anthropic's `tool_use`/`tool_result`, Gemini via `functionCall`/`functionResponse`, etc.), but the loop itself is agnostic to the provider.

## Phase

**Phase 1** (Weeks 1–6)

## Prerequisites

- LLM provider abstraction exists with `GenerateResponse` method (`internal/ai/service.go` or provider-specific files)
- Asynq worker task processor exists (`internal/worker/processor.go`)
- Domain models defined (`internal/domain/`)
- Tool registry implemented (Spec 12)
- `LLMService` interface updated with `GenerateWithTools` method

## Deliverables

**Files to Create:**
- `internal/ai/tool_loop.go` — provider-agnostic agentic loop handler

**Files to Modify:**
- `internal/ai/service.go` (or provider interface) — add `GenerateWithTools` method to `LLMService` interface
- `internal/ai/claude.go` — implement `GenerateWithTools` that parses provider-specific tool calls and normalizes to `ToolCall` objects
- `internal/ai/gemini.go` — implement `GenerateWithTools` for Gemini (if added in this phase)
- `internal/worker/processor.go` — use `llmService.GenerateWithTools` in agentic loop; loop code is provider-agnostic
- `internal/domain/message.go` — add `Message.Type` field (one of: "text", "tool_call", "tool_result")

## Acceptance Criteria

- [ ] `LLMService.GenerateWithTools(ctx, messages, tools)` method exists and is implemented by all providers
- [ ] Provider implementation parses tool calls from LLM response and returns normalized `ToolCall` objects
- [ ] `AgentLoop.Run(ctx, llmService, toolRegistry, messages)` is provider-agnostic (lives in `tool_loop.go`)
- [ ] Loop extracts tool name, input parameters, and call ID from LLM response
- [ ] Loop passes tool call to registry for execution (Spec 12)
- [ ] Loop formats tool result and feeds back to LLM provider in next request
- [ ] Loop continues until LLM returns non-tool-call response (ends with final text)
- [ ] Loop times out after 10 retries (prevents infinite loops)
- [ ] Error from tool execution is formatted as error tool result and sent back to LLM
- [ ] Loop logs all tool calls (name, input, output, latency, success/failure, provider)

## API / Component Contract

**`internal/ai/service.go` (LLMService interface)**:
```go
type ToolCall struct {
	ID    string                 // Unique ID from LLM provider
	Name  string                 // Tool name to invoke
	Input map[string]interface{} // Tool parameters
}

type LLMService interface {
	// GenerateResponse: basic completion without tools
	GenerateResponse(ctx context.Context, messages []interface{}) (string, error)

	// GenerateWithTools: returns text response and any tool calls to execute
	GenerateWithTools(ctx context.Context, messages []interface{}, tools []ToolDefinition) (*ToolCallResponse, error)
}

type ToolCallResponse struct {
	Text      string      // Final text response (empty if tool calls pending)
	ToolCalls []ToolCall  // Tool calls requested by LLM
	Complete  bool        // true if response is final (no more tool calls expected)
}
```

**`internal/ai/claude.go` (provider-specific implementation)**:
```go
func (c *ClaudeService) GenerateWithTools(ctx context.Context, messages []interface{}, tools []ToolDefinition) (*ToolCallResponse, error) {
	// Call Anthropic API with tool definitions
	// Parse tool_use blocks from response
	// Return normalized ToolCall objects in ToolCallResponse
	// Provider-specific: uses Anthropic's tool_use/tool_result format internally
}
```

**`internal/ai/tool_loop.go` (provider-agnostic loop)**:
```go
func RunAgentLoop(ctx context.Context, llm LLMService, registry *tools.Registry, messages []interface{}, maxRetries int) (string, error) {
	toolDefs := registry.GetDefinitions() // Get normalized tool definitions

	for i := 0; i < maxRetries; i++ {
		resp, err := llm.GenerateWithTools(ctx, messages, toolDefs)
		if err != nil {
			return "", err
		}

		if resp.Complete || len(resp.ToolCalls) == 0 {
			// Final response
			return resp.Text, nil
		}

		// Execute tool calls and feed results back (provider-agnostic)
		for _, call := range resp.ToolCalls {
			result, err := registry.Execute(ctx, call.Name, call.Input)
			// Format as tool result message (provider handles format)
			messages = append(messages, Message{
				Role: "user",
				Type: "tool_result",
				ToolResult: &ToolResult{
					ID:      call.ID,
					Content: result,
					IsError: err != nil,
				},
			})
		}
	}
	return "", ErrMaxRetriesExceeded
}
```

## Out of Scope

- Individual tool implementations (see Specs 13–21)
- Tool registry design (see Spec 12)
- Confirmation/approval logic (see Spec 20)
- Multi-provider LLM abstraction (Phase 2, Spec 31)
