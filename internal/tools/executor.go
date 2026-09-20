package tools

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"bruce/internal/ai"
)

const defaultTimeout = 150 * time.Second

// executeWithTimeout runs a tool with a timeout, panic recovery, and execution logging.
func executeWithTimeout(ctx context.Context, tool Tool, input map[string]interface{}, db *sql.DB) (string, error) {
	startTime := time.Now()

	// Create a child context with timeout.
	execCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	// Channel to receive the result or panic recovery.
	type result struct {
		output interface{}
		err    error
	}
	resultChan := make(chan result, 1)

	// Run the tool in a goroutine to catch panics.
	go func() {
		defer func() {
			if r := recover(); r != nil {
				resultChan <- result{
					output: nil,
					err:    fmt.Errorf("tool panicked: %v", r),
				}
			}
		}()

		output, err := tool.Execute(execCtx, input)
		resultChan <- result{output: output, err: err}
	}()

	// Wait for result or timeout.
	var output interface{}
	var execErr error

	select {
	case res := <-resultChan:
		output = res.output
		execErr = res.err
	case <-execCtx.Done():
		execErr = fmt.Errorf("tool execution timed out")
	}

	// Log the execution to the database.
	latencyMs := int64(time.Since(startTime).Milliseconds())
	safeInput := scrubSecrets(input)
	inputJSON, _ := json.Marshal(safeInput)
	outputJSON := ""
	if output != nil {
		if b, err := json.Marshal(output); err == nil {
			outputJSON = string(b)
		}
	}
	errorMsg := ""
	if execErr != nil {
		errorMsg = execErr.Error()
	}

	sessionID, _ := ai.SessionIDFromContext(ctx)
	logExecution(db, sessionID, tool.Name(), string(inputJSON), outputJSON, latencyMs, execErr == nil, errorMsg)

	// Convert output to JSON string for the return value.
	if execErr != nil {
		return "", execErr
	}

	// Convert interface{} to JSON string.
	outJSON, err := json.Marshal(output)
	if err != nil {
		return "", fmt.Errorf("marshal tool output: %w", err)
	}

	return string(outJSON), nil
}

// scrubSecrets returns a shallow copy of input with sensitive header values redacted.
// Keys matched case-insensitively: authorization, x-api-key, x-n8n-api-key.
func scrubSecrets(input map[string]interface{}) map[string]interface{} {
	sensitiveKeys := map[string]bool{
		"authorization": true,
		"x-api-key":     true,
		"x-n8n-api-key": true,
	}

	out := make(map[string]interface{}, len(input))
	for k, v := range input {
		if sensitiveKeys[strings.ToLower(k)] {
			if _, isStr := v.(string); isStr {
				out[k] = "[REDACTED]"
				continue
			}
		}
		// Recursively scrub nested maps.
		if nested, ok := v.(map[string]interface{}); ok {
			out[k] = scrubSecrets(nested)
		} else {
			out[k] = v
		}
	}
	return out
}

// logExecution logs a tool execution to the tool_executions table.
func logExecution(db *sql.DB, sessionID string, toolName string, inputJSON string, outputJSON string, latencyMs int64, success bool, errorMsg string) {
	if db == nil {
		return
	}

	id := uuid.New().String()
	successInt := 0
	if success {
		successInt = 1
	}

	_, _ = db.Exec(
		`INSERT INTO tool_executions (id, session_id, tool_name, input, output, latency_ms, success, error_msg)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id, sessionID, toolName, inputJSON, outputJSON, latencyMs, successInt, errorMsg,
	)
}
