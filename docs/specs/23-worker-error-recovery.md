# Spec 24: Worker Error Recovery [BACKEND]

## Overview

Implement comprehensive error recovery for tool execution. When a tool fails (timeout, API error, network error), the error message and context are fed back to Claude as a system message in the next request. Worker retries failing tasks with exponential backoff (1s, 2s, 4s, 8s max 3 retries). Rate limits (429) and timeouts (5xx) trigger backoff; permanent errors (400, 403, 404) are not retried. Worker logs all errors with full context for debugging.

## Phase

**Phase 2** (Weeks 7–14)

## Prerequisites

- Tool execution engine exists (Spec 11)
- Tool registry exists (Spec 12)
- Worker processor exists with basic error handling

## Deliverables

**Files to Modify:**
- `internal/worker/processor.go` — add retry loop with exponential backoff
- `internal/worker/executor.go` (or new file) — error context formatter
- `internal/ai/claude.go` — accept error context in message stream (system message)
- Logging: extend worker logs to include error details

## Acceptance Criteria

- [ ] Tool timeout errors are retried up to 3 times with backoff (1s, 2s, 4s)
- [ ] Rate limit errors (429) retry indefinitely with long backoff (cap at 60s between retries)
- [ ] Permanent errors (400, 403, 404, permission denied) are not retried; error is sent to Claude
- [ ] Network errors (connection refused, DNS error) are retried up to 2 times
- [ ] Error context includes: tool name, input (sanitized), error message, timestamp, retry attempt
- [ ] Error context is formatted as system message and fed back to Claude: `"Tool 'gmail' failed: Connection timeout. Attempt 2/3. Retrying..."`
- [ ] Claude receives error context and can suggest alternatives (e.g., "Try using Calendar instead") or retry with different parameters
- [ ] All errors are logged to SQLite (tool_executions table) with: status (success/failure), error_message, retry_count
- [ ] If max retries exceeded, worker returns error to user and logs `ERROR` level
- [ ] Backoff jitter is applied (±10%) to prevent thundering herd

## API / Component Contract

**Error Context Structure**:
```go
type ExecutionError struct {
	ToolName     string
	Input        map[string]interface{}
	ErrorMessage string
	Timestamp    time.Time
	Attempt      int
	MaxAttempts  int
	IsRetryable  bool
}

func (e *ExecutionError) ToSystemMessage() string {
	return fmt.Sprintf(
		"Tool '%s' failed: %s. Attempt %d/%d.",
		e.ToolName, e.ErrorMessage, e.Attempt, e.MaxAttempts,
	)
}
```

**Retry Logic** (pseudo-code):
```go
func executeWithRetry(ctx context.Context, tool Tool, input map[string]interface{}) (interface{}, error) {
	maxRetries := 3
	for attempt := 1; attempt <= maxRetries; attempt++ {
		result, err := tool.Execute(ctx, input)
		if err == nil {
			return result, nil
		}

		// Check if retryable
		if !isRetryableError(err) {
			return nil, err // Permanent error, give up
		}

		// Calculate backoff with jitter
		if attempt < maxRetries {
			backoff := exponentialBackoff(attempt)
			time.Sleep(backoff)
			// Feed error to Claude, continue loop
		}
	}
	return nil, fmt.Errorf("Tool failed after %d retries", maxRetries)
}
```

## Out of Scope

- Circuit breaker pattern (Phase 3+)
- Adaptive retry strategy based on service health
- Metrics/alerting on high error rates
- User notification on repeated failures
