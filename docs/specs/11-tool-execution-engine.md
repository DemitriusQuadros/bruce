# Spec 11: Tool Execution Engine [BACKEND]

## Overview

Extend the Claude integration in `internal/ai/claude.go` to handle Anthropic's `tool_use` and `tool_result` message types. The worker processes Claude's tool calls, executes registered tools, integrates results back into the message stream, and loops until Claude produces a final (non-tool-use) response. This is the foundational agentic loop that all other tools depend on.

## Phase

**Phase 1** (Weeks 1–6)

## Prerequisites

- Claude API integration already in place (`internal/ai/claude.go` exists)
- Asynq worker task processor exists (`internal/worker/processor.go`)
- Domain models defined (`internal/domain/`)
- Tool registry implemented (Spec 12)

## Deliverables

**Files to Create:**
- `internal/ai/tool_loop.go` — multi-turn Claude loop handler

**Files to Modify:**
- `internal/ai/claude.go` — extend `GenerateResponse` to parse `tool_use` blocks; return `ToolCall` objects
- `internal/worker/processor.go` — add agentic loop that calls tool registry and feeds results back to Claude
- `internal/domain/message.go` — add `Message.Type` field (one of: "text", "tool_use", "tool_result")

## Acceptance Criteria

- [ ] Claude successfully parses `tool_use` message blocks from API response
- [ ] Worker extracts tool name, input parameters, and ID from `tool_use` block
- [ ] Worker passes tool call to registry for execution (Spec 12)
- [ ] Worker formats tool result as `tool_result` message type and sends back to Claude in next request
- [ ] Loop continues until Claude returns non-`tool_use` response (ends with "text" or "end_turn")
- [ ] Loop times out after 10 retries (prevents infinite loops from buggy Claude)
- [ ] Error from tool execution is formatted as `tool_result` with `is_error: true` and sent back to Claude
- [ ] Worker logs all tool calls (name, input, output, latency, success/failure)

## API / Component Contract

**`internal/ai/claude.go`**:
```go
type ToolCall struct {
	ID    string                 // Unique ID from Claude
	Name  string                 // Tool name to invoke
	Input map[string]interface{} // Tool parameters
}

// GenerateResponse now returns both text and any tool calls to execute
type GenerateResponse struct {
	Text      string
	ToolCalls []ToolCall
	Complete  bool // true if response is final (no more tool calls expected)
}

func (c *ClaudeService) GenerateResponse(ctx context.Context, messages []interface{}) (*GenerateResponse, error) {
	// ... existing code ...
	// NEW: detect tool_use blocks, parse them, return in ToolCall slice
}
```

**`internal/worker/processor.go`** (pseudo-code):
```go
func processMessage(ctx context.Context, session *Session, messages []Message) error {
	for i := 0; i < 10; i++ { // loop safety
		resp, err := claudeService.GenerateResponse(ctx, messages)
		if err != nil {
			return err
		}

		if resp.Complete || len(resp.ToolCalls) == 0 {
			// Final response: save to DB and dispatch
			return sendMessage(ctx, session, resp.Text)
		}

		// Execute tool calls
		for _, call := range resp.ToolCalls {
			result, err := toolRegistry.Execute(ctx, call.Name, call.Input)
			// Format as tool_result message
			messages = append(messages, Message{
				Role: "user",
				Type: "tool_result",
				ToolUse: &ToolResult{
					ID:      call.ID,
					Content: result,
					IsError: err != nil,
				},
			})
		}
	}
	return ErrMaxRetriesExceeded
}
```

## Out of Scope

- Individual tool implementations (see Specs 13–21)
- Tool registry design (see Spec 12)
- Confirmation/approval logic (see Spec 20)
- Multi-provider LLM abstraction (Phase 2, Spec 31)
