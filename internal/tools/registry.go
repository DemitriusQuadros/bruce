package tools

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"sync"

	"bruce/internal/ai"
)

// Registry holds all registered tools and provides methods to query and execute them.
type Registry struct {
	mu    sync.RWMutex
	tools map[string]Tool
	db    *sql.DB
}

// NewRegistry creates a new tool registry.
func NewRegistry(db *sql.DB) *Registry {
	return &Registry{
		tools: make(map[string]Tool),
		db:    db,
	}
}

// Register registers a tool. Returns an error if a tool with the same name is already registered.
func (r *Registry) Register(tool Tool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := tool.Name()
	if _, exists := r.tools[name]; exists {
		return fmt.Errorf("tool %q already registered", name)
	}

	r.tools[name] = tool
	return nil
}

// Get retrieves a tool by name. Returns an error if the tool is not found.
func (r *Registry) Get(name string) (Tool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, exists := r.tools[name]
	if !exists {
		return nil, fmt.Errorf("tool %q not found", name)
	}

	return tool, nil
}

// List returns a sorted slice of ToolMetadata for all registered tools.
func (r *Registry) List() []ToolMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()

	metadata := make([]ToolMetadata, 0, len(r.tools))
	for _, tool := range r.tools {
		def := tool.Definition()
		metadata = append(metadata, ToolMetadata{
			Name:        def.Name,
			Description: def.Description,
			InputSchema: def.InputSchema,
		})
	}

	// Sort by name for consistent ordering.
	sort.Slice(metadata, func(i, j int) bool {
		return metadata[i].Name < metadata[j].Name
	})

	return metadata
}

// GetDefinitions returns the tool definitions for all registered tools, sorted by name.
// This satisfies the ai.ToolRegistry interface.
func (r *Registry) GetDefinitions() []ai.ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	defs := make([]ai.ToolDefinition, 0, len(r.tools))
	for _, tool := range r.tools {
		defs = append(defs, tool.Definition())
	}

	// Sort by name for consistent ordering.
	sort.Slice(defs, func(i, j int) bool {
		return defs[i].Name < defs[j].Name
	})

	return defs
}

// Execute runs a tool by name and returns the result as a JSON string.
// This satisfies the ai.ToolRegistry interface.
func (r *Registry) Execute(ctx context.Context, name string, input map[string]interface{}) (string, error) {
	tool, err := r.Get(name)
	if err != nil {
		return "", err
	}

	// Execute the tool with timeout and panic recovery.
	return executeWithTimeout(ctx, tool, input, r.db)
}
