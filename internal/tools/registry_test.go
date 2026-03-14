package tools

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"bruce/internal/ai"
	"bruce/internal/database"
)

// mockTool is a test implementation of the Tool interface.
type mockTool struct {
	name        string
	description string
	inputSchema map[string]interface{}
	execFunc    func(ctx context.Context, input map[string]interface{}) (interface{}, error)
}

func (m *mockTool) Name() string {
	return m.name
}

func (m *mockTool) Definition() ai.ToolDefinition {
	return ai.ToolDefinition{
		Name:        m.name,
		Description: m.description,
		InputSchema: m.inputSchema,
	}
}

func (m *mockTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	if m.execFunc != nil {
		return m.execFunc(ctx, input)
	}
	return map[string]string{"result": "ok"}, nil
}

// TestRegisterAndGet tests registering and retrieving tools.
func TestRegisterAndGet(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	registry := NewRegistry(db)

	tool := &mockTool{
		name:        "test_tool",
		description: "A test tool",
		inputSchema: map[string]interface{}{"type": "object"},
	}

	// Register the tool
	err := registry.Register(tool)
	require.NoError(t, err)

	// Retrieve the tool
	retrieved, err := registry.Get("test_tool")
	require.NoError(t, err)
	assert.Equal(t, tool, retrieved)
}

// TestRegisterDuplicate tests that registering a duplicate tool returns an error.
func TestRegisterDuplicate(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	registry := NewRegistry(db)

	tool := &mockTool{
		name:        "test_tool",
		description: "A test tool",
		inputSchema: map[string]interface{}{"type": "object"},
	}

	// Register the tool
	err := registry.Register(tool)
	require.NoError(t, err)

	// Try to register again
	err = registry.Register(tool)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

// TestGetNotFound tests that retrieving a non-existent tool returns an error.
func TestGetNotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	registry := NewRegistry(db)

	_, err := registry.Get("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestList tests listing all registered tools sorted by name.
func TestList(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	registry := NewRegistry(db)

	tools := []Tool{
		&mockTool{name: "zebra", description: "Z tool", inputSchema: map[string]interface{}{}},
		&mockTool{name: "apple", description: "A tool", inputSchema: map[string]interface{}{}},
		&mockTool{name: "middle", description: "M tool", inputSchema: map[string]interface{}{}},
	}

	for _, tool := range tools {
		err := registry.Register(tool)
		require.NoError(t, err)
	}

	metadata := registry.List()
	require.Len(t, metadata, 3)

	// Verify sorted by name
	assert.Equal(t, "apple", metadata[0].Name)
	assert.Equal(t, "middle", metadata[1].Name)
	assert.Equal(t, "zebra", metadata[2].Name)
}

// TestGetDefinitions tests that GetDefinitions returns definitions in sorted order.
func TestGetDefinitions(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	registry := NewRegistry(db)

	tools := []Tool{
		&mockTool{name: "zebra", description: "Z tool", inputSchema: map[string]interface{}{"z": "value"}},
		&mockTool{name: "apple", description: "A tool", inputSchema: map[string]interface{}{"a": "value"}},
	}

	for _, tool := range tools {
		err := registry.Register(tool)
		require.NoError(t, err)
	}

	defs := registry.GetDefinitions()
	require.Len(t, defs, 2)

	// Verify sorted by name
	assert.Equal(t, "apple", defs[0].Name)
	assert.Equal(t, "zebra", defs[1].Name)
}

// TestExecuteHappyPath tests successful tool execution.
func TestExecuteHappyPath(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	registry := NewRegistry(db)

	tool := &mockTool{
		name:        "test_tool",
		description: "A test tool",
		inputSchema: map[string]interface{}{},
		execFunc: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
			return map[string]string{"status": "success"}, nil
		},
	}

	err := registry.Register(tool)
	require.NoError(t, err)

	result, err := registry.Execute(context.Background(), "test_tool", map[string]interface{}{})
	require.NoError(t, err)

	// Verify result is JSON-encoded
	var decoded map[string]string
	err = json.Unmarshal([]byte(result), &decoded)
	require.NoError(t, err)
	assert.Equal(t, "success", decoded["status"])
}

// TestExecuteNotFound tests executing a non-existent tool.
func TestExecuteNotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	registry := NewRegistry(db)

	_, err := registry.Execute(context.Background(), "nonexistent", map[string]interface{}{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestExecuteTimeout tests that tool execution times out correctly.
func TestExecuteTimeout(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	registry := NewRegistry(db)

	tool := &mockTool{
		name: "slow_tool",
		execFunc: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
			// Simulate a slow tool that exceeds the timeout.
			time.Sleep(10 * time.Second)
			return nil, nil
		},
	}

	err := registry.Register(tool)
	require.NoError(t, err)

	_, err = registry.Execute(context.Background(), "slow_tool", map[string]interface{}{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timed out")
}

// TestExecutePanic tests that panicking tools are caught and reported.
func TestExecutePanic(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	registry := NewRegistry(db)

	tool := &mockTool{
		name: "panic_tool",
		execFunc: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
			panic("something went wrong")
		},
	}

	err := registry.Register(tool)
	require.NoError(t, err)

	_, err = registry.Execute(context.Background(), "panic_tool", map[string]interface{}{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "panicked")
	assert.Contains(t, err.Error(), "something went wrong")
}

// TestExecuteLogging tests that tool executions are logged to the database.
func TestExecuteLogging(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	registry := NewRegistry(db)

	tool := &mockTool{
		name: "logged_tool",
		execFunc: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
			return map[string]string{"result": "ok"}, nil
		},
	}

	err := registry.Register(tool)
	require.NoError(t, err)

	input := map[string]interface{}{"key": "value"}
	_, err = registry.Execute(context.Background(), "logged_tool", input)
	require.NoError(t, err)

	// Verify the execution was logged to the database
	var toolName, inputStr, outputStr string
	var latencyMs int64
	var success int

	err = db.QueryRow(
		`SELECT tool_name, input, output, latency_ms, success FROM tool_executions ORDER BY executed_at DESC LIMIT 1`,
	).Scan(&toolName, &inputStr, &outputStr, &latencyMs, &success)

	require.NoError(t, err)
	assert.Equal(t, "logged_tool", toolName)
	assert.Contains(t, inputStr, "key")
	assert.Contains(t, outputStr, "ok")
	assert.GreaterOrEqual(t, latencyMs, int64(0))
	assert.Equal(t, 1, success)
}

// TestExecuteError tests that tool errors are logged correctly.
func TestExecuteError(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	registry := NewRegistry(db)

	tool := &mockTool{
		name: "error_tool",
		execFunc: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
			return nil, fmt.Errorf("test error")
		},
	}

	err := registry.Register(tool)
	require.NoError(t, err)

	_, err = registry.Execute(context.Background(), "error_tool", map[string]interface{}{})
	assert.Error(t, err)

	// Verify the error was logged to the database
	var errorMsg string
	var success int

	err = db.QueryRow(
		`SELECT error_msg, success FROM tool_executions ORDER BY executed_at DESC LIMIT 1`,
	).Scan(&errorMsg, &success)

	require.NoError(t, err)
	assert.Contains(t, errorMsg, "test error")
	assert.Equal(t, 0, success)
}

// TestConcurrentRegistration tests that concurrent registration is safe.
func TestConcurrentRegistration(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	registry := NewRegistry(db)

	// Register tools concurrently
	done := make(chan error, 3)

	for i := 0; i < 3; i++ {
		go func(idx int) {
			tool := &mockTool{
				name: fmt.Sprintf("tool_%d", idx),
			}
			done <- registry.Register(tool)
		}(i)
	}

	for i := 0; i < 3; i++ {
		assert.NoError(t, <-done)
	}

	// Verify all tools are registered
	metadata := registry.List()
	assert.Len(t, metadata, 3)
}

// TestConcurrentExecution tests that concurrent execution is safe.
func TestConcurrentExecution(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	registry := NewRegistry(db)

	tool := &mockTool{
		name: "concurrent_tool",
		execFunc: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
			return map[string]interface{}{"id": input["id"]}, nil
		},
	}

	err := registry.Register(tool)
	require.NoError(t, err)

	// Execute concurrently
	done := make(chan error, 5)

	for i := 0; i < 5; i++ {
		go func(idx int) {
			input := map[string]interface{}{"id": idx}
			_, err := registry.Execute(context.Background(), "concurrent_tool", input)
			done <- err
		}(i)
	}

	for i := 0; i < 5; i++ {
		assert.NoError(t, <-done)
	}

	// Verify all executions were logged
	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM tool_executions`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 5, count)
}

// setupTestDB creates an in-memory SQLite database with the schema.
func setupTestDB(t *testing.T) (*sql.DB, func()) {
	db, err := database.NewSQLiteDB(":memory:")
	require.NoError(t, err)

	// Run migrations to set up the schema
	err = database.RunMigrations(db)
	require.NoError(t, err)

	cleanup := func() {
		db.Close()
	}

	return db, cleanup
}
