# Spec 12: Tool Registry [BACKEND]

## Overview

Create `internal/tools/` package with a centralized `Tool` interface and registry. Each tool implements `Name()`, `Schema()`, and `Execute()`. The registry stores all tools by name, isolates errors per tool (one tool failure doesn't crash others), enforces timeouts, and provides introspection (list all available tools + their parameters for Claude). Tools are registered at startup in `cmd/bruce/main.go`.

## Phase

**Phase 1** (Weeks 1–6)

## Prerequisites

- Domain models exist (`internal/domain/`)
- Worker processor exists (`internal/worker/processor.go`)

## Deliverables

**Files to Create:**
- `internal/tools/registry.go` — central registry with Lock, map[string]Tool, Add/Get/Execute methods
- `internal/tools/tool.go` — Tool interface definition
- `internal/tools/executor.go` — generic execution wrapper with timeout, error handling, logging

**Files to Modify:**
- `cmd/bruce/main.go` — register all tools at startup

## Acceptance Criteria

- [ ] `Tool` interface has methods: `Name()`, `Schema()`, `Execute(ctx context.Context, input map[string]interface{}) (interface{}, error)`
- [ ] `Tool.Schema()` returns JSON Schema of tool parameters (for Claude introspection)
- [ ] Registry has `Register(tool Tool)`, `Get(name string) Tool`, `List() []ToolMetadata` methods
- [ ] Registry is thread-safe (uses `sync.RWMutex`)
- [ ] `Execute` method wraps tool call in timeout (5s default, configurable per tool)
- [ ] If tool times out, returns error `"Tool execution timed out"` without panic
- [ ] If tool panics, recovers and returns error `"Tool panicked: [details]"`
- [ ] All tool executions are logged to SQLite (Spec 11) with: name, input, output, latency, success/failure
- [ ] Registry is populated by `main.go` at startup; tools can be enabled/disabled via config

## API / Component Contract

**`internal/tools/tool.go`**:
```go
type Tool interface {
	Name() string
	Schema() map[string]interface{} // JSON Schema
	Execute(ctx context.Context, input map[string]interface{}) (interface{}, error)
}

type ToolMetadata struct {
	Name        string
	Description string
	InputSchema map[string]interface{}
}
```

**`internal/tools/registry.go`**:
```go
type Registry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

func (r *Registry) Register(tool Tool) error { /* ... */ }
func (r *Registry) Get(name string) (Tool, error) { /* ... */ }
func (r *Registry) Execute(ctx context.Context, name string, input map[string]interface{}) (interface{}, error) {
	// timeout, panic recovery, logging
}
func (r *Registry) List() []ToolMetadata { /* ... */ }
```

**`cmd/bruce/main.go`** (startup):
```go
registry := tools.NewRegistry()
if config.Tools.Gmail.Enabled {
	registry.Register(gmail.NewTool(gmailClient))
}
if config.Tools.Calendar.Enabled {
	registry.Register(calendar.NewTool(calendarClient))
}
// ... more tools
workerService.SetToolRegistry(registry)
```

## Out of Scope

- Specific tool implementations (Specs 13–31)
- Credential storage or OAuth flows (Spec 13)
- Tool approval/confirmation (Spec 20)
