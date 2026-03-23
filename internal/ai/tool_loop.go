package ai

import (
	"context"
	"fmt"

	"bruce/internal/domain"
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

		// Append assistant's tool_use turn to messages.
		// Providers like Claude require the exact tool_use blocks in the assistant message
		// to match the tool_use_ids in the subsequent tool_result messages.
		domainCalls := make([]domain.ToolCall, len(resp.ToolCalls))
		for i, c := range resp.ToolCalls {
			domainCalls[i] = domain.ToolCall{ID: c.ID, Name: c.Name, Input: c.Input}
		}
		toolUseMsg := domain.Message{
			Role:      "assistant",
			Type:      "tool_call",
			Content:   resp.Text, // may carry partial text alongside tool calls
			ToolCalls: domainCalls,
		}
		currentMessages = append(currentMessages, toolUseMsg)

		// Execute each tool and collect results
		for _, call := range resp.ToolCalls {
			result, execErr := registry.Execute(ctx, call.Name, call.Input)

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
