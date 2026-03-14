package ai

import (
	"context"
	"fmt"

	"bruce/internal/domain"
	"bruce/internal/monitoring"
)

// ToolRegistry is the interface the agentic loop uses to look up and execute tools.
// Spec 12 will provide a concrete implementation.
type ToolRegistry interface {
	GetDefinitions() []ToolDefinition
	Execute(ctx context.Context, name string, input map[string]interface{}) (string, error)
}

// RunAgentLoop runs the provider-agnostic agentic loop.
// It calls llm.GenerateWithTools, executes any requested tools via registry,
// feeds results back, and repeats until the LLM returns a final text response
// or maxRetries is reached.
func RunAgentLoop(
	ctx context.Context,
	llm LLMService,
	registry ToolRegistry,
	systemPrompt string,
	messages []domain.Message,
	maxRetries int,
	logger *monitoring.StructuredLogger,
) (string, error) {
	currentMessages := make([]domain.Message, len(messages))
	copy(currentMessages, messages)

	for i := 0; i < maxRetries; i++ {
		// Get tool definitions from registry
		toolDefs := registry.GetDefinitions()

		// Call LLM with tools
		resp, err := llm.GenerateWithTools(ctx, systemPrompt, currentMessages, toolDefs)
		if err != nil {
			return "", fmt.Errorf("generate with tools: %w", err)
		}

		// If no tool calls or complete, return final text
		if resp.Complete || len(resp.ToolCalls) == 0 {
			return resp.Text, nil
		}

		// Append assistant's tool_use turn to messages (for API continuity)
		// This tracks that the assistant requested these tools
		toolUseMsg := domain.Message{
			Role:    "assistant",
			Type:    "tool_call",
			Content: "", // Tool calls don't have text content in tool_call type
		}
		currentMessages = append(currentMessages, toolUseMsg)

		// Execute each tool and collect results
		for _, call := range resp.ToolCalls {
			result, execErr := registry.Execute(ctx, call.Name, call.Input)

			// Log execution
			if logger != nil {
				logFields := map[string]interface{}{
					"tool":    call.Name,
					"success": execErr == nil,
				}
				if execErr != nil {
					logFields["error"] = execErr.Error()
				} else {
					logFields["result_len"] = len(result)
				}
				logger.Info(ctx, "tool_loop", "tool_executed", logFields)
			}

			// Append tool result as message
			toolResult := &domain.ToolResult{
				ID:      call.ID,
				Content: result,
				IsError: execErr != nil,
			}
			if execErr != nil {
				toolResult.Content = execErr.Error()
			}

			resultMsg := domain.Message{
				Role:       "user",
				Type:       "tool_result",
				ToolResult: toolResult,
				Content:    "", // Tool results don't have text content
			}
			currentMessages = append(currentMessages, resultMsg)
		}
	}

	return "", ErrMaxRetriesExceeded
}
